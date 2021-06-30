package buildjob

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/dominodatalab/forge/api/forge/v1alpha1"
	"github.com/dominodatalab/forge/internal/builder"
	"github.com/dominodatalab/forge/internal/clientset"
	forgev1alpha1 "github.com/dominodatalab/forge/internal/clientset/typed/forge/v1alpha1"

	"github.com/dominodatalab/forge/internal/config"
	"github.com/dominodatalab/forge/internal/credentials"
	forgek8s "github.com/dominodatalab/forge/internal/kubernetes"
	"github.com/dominodatalab/forge/internal/message"
	"github.com/dominodatalab/forge/plugins/preparer"
)

type Job struct {
	log logr.Logger

	clientk8s   kubernetes.Interface
	clientforge forgev1alpha1.ForgeV1alpha1Interface

	producer message.Producer

	plugins []*preparer.Plugin

	builder builder.OCIImageBuilder

	name      string
	namespace string

	cleanupSteps []func()
}

func New(cfg Config) (*Job, error) {
	log := NewLogger()

	// initialize kubernetes clients
	log.Info("Initializing Kubernetes clients")

	restCfg, err := forgek8s.LoadKubernetesConfig()
	if err != nil {
		return nil, errors.Wrap(err, "cannot load k8s config")
	}
	clientsk8s, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create k8s api client")
	}
	client, err := clientset.NewForConfig(restCfg)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create forge api client")
	}

	var cleanupSteps []func()

	// setup message publisher
	var producer message.Producer
	if cfg.BrokerOpts != nil {
		log.Info("Initializing status update message publisher")

		if producer, err = message.NewProducer(cfg.BrokerOpts, log); err != nil {
			return nil, err
		}
		cleanupSteps = append(cleanupSteps, func() {
			log.Info("Closing message producer")
			producer.Close()
		})
	}

	// setup preparer plugins
	log.Info("Loading configured preparer plugins")

	preparerPlugins, err := preparer.LoadPlugins(cfg.PreparerPluginsPath)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to load preparer plugins path %q", cfg.PreparerPluginsPath)
	}
	cleanupSteps = append(cleanupSteps, func() {
		log.Info("Killing preparer plugins")
		for _, preparerPlugin := range preparerPlugins {
			preparerPlugin.Kill()
		}
	})

	// instantiate the image builder
	log.Info("Initializing OCI image builder")

	ociBuilder, err := builder.New(preparerPlugins, cfg.EnableLayerCaching, log)
	if err != nil {
		return nil, errors.Wrap(err, "image builder initialization failed")
	}

	return &Job{
		log:          log,
		name:         cfg.ResourceName,
		namespace:    cfg.ResourceNamespace,
		clientk8s:    clientsk8s,
		clientforge:  client.ForgeV1alpha1(),
		producer:     producer,
		plugins:      preparerPlugins,
		builder:      ociBuilder,
		cleanupSteps: cleanupSteps,
	}, nil
}

func (job *Job) Run() error {
	ctx := context.Background()

	job.log.Info("Fetching ContainerImageBuild resource", "Name", job.name, "Namespace", job.namespace)
	cib, err := job.clientforge.ContainerImageBuilds(job.namespace).Get(ctx, job.name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "cannot find containerimagebuild %s", job.name)
	}

	if cib, err = job.transitionToBuilding(ctx, cib); err != nil {
		return err
	}

	job.log = job.log.WithValues("annotations", cib.Annotations)

	job.log.Info("Creating build options using custom resource fields")
	opts, err := job.generateBuildOptions(ctx, cib)
	if err != nil {
		err = errors.Wrap(err, "failed to generate build options")

		if iErr := job.transitionToFailure(ctx, cib, err); iErr != nil {
			err = errors.Wrap(err, iErr.Error())
		}
		return err
	}

	job.builder.SetLogger(job.log)
	images, err := job.builder.BuildAndPush(ctx, opts)
	if err != nil {
		logError(job.log, err)

		if iErr := job.transitionToFailure(ctx, cib, err); iErr != nil {
			err = errors.Wrap(err, iErr.Error())
		}
		return err
	}

	return job.transitionToComplete(ctx, cib, images)
}

func (job *Job) Cleanup(forced bool) {
	if forced {
		job.log.Info("Caught kill signal, cleaning up")
	}

	for _, fn := range job.cleanupSteps {
		fn()
	}
}

func (job *Job) generateBuildOptions(ctx context.Context, cib *v1alpha1.ContainerImageBuild) (*config.BuildOptions, error) {
	registries, err := job.buildRegistryConfigs(ctx, cib.Spec.Registries)
	if err != nil {
		return nil, errors.Wrap(err, "cannot build registry config")
	}

	opts := &config.BuildOptions{
		ContextURL:              cib.Spec.Context,
		ContextTimeout:          time.Duration(cib.Spec.ContextTimeoutSeconds) * time.Second,
		ImageName:               cib.Spec.ImageName,
		ImageSizeLimit:          cib.Spec.ImageSizeLimit,
		Labels:                  cib.Spec.Labels,
		BuildArgs:               cib.Spec.BuildArgs,
		DisableBuildCache:       cib.Spec.DisableBuildCache,
		DisableLayerCacheExport: cib.Spec.DisableLayerCacheExport,
		PushRegistries:          cib.Spec.PushRegistries,
		PluginData:              cib.Spec.PluginData,
		Timeout:                 time.Duration(cib.Spec.TimeoutSeconds) * time.Second,
		Registries:              registries,
	}
	return opts, nil
}

// uses api registry directives to generate a list of registry configurations for image building
func (job *Job) buildRegistryConfigs(ctx context.Context, apiRegs []v1alpha1.Registry) (configs []config.Registry, err error) {
	for _, apiReg := range apiRegs {
		conf := config.Registry{
			Host:   apiReg.Server,
			NonSSL: apiReg.NonSSL,
		}

		// NOTE: move BasicAuth validation into an admission webhook at a later time
		if err := apiReg.BasicAuth.Validate(); err != nil {
			return nil, errors.Wrap(err, "basic auth validation failed")
		}

		var fetchAuth func() (username, password string, err error)
		switch {
		case apiReg.BasicAuth.IsInline():
			fetchAuth = func() (string, string, error) {
				return apiReg.BasicAuth.Username, apiReg.BasicAuth.Password, nil
			}
		case apiReg.BasicAuth.IsSecret():
			fetchAuth = func() (string, string, error) {
				return job.getDockerAuthFromSecret(ctx, apiReg.Server, apiReg.BasicAuth.SecretName, apiReg.BasicAuth.SecretNamespace)
			}
		case apiReg.DynamicCloudCredentials:
			fetchAuth = func() (string, string, error) {
				return job.getDockerAuthFromFS(apiReg.Server)
			}
		}

		if fetchAuth != nil {
			var err error
			conf.Username, conf.Password, err = fetchAuth()
			if err != nil {
				return nil, err
			}
		}

		configs = append(configs, conf)
	}

	return configs, nil
}

func (job *Job) getDockerAuthFromFS(host string) (string, string, error) {
	if _, err := os.Stat(config.DynamicCredentialsFilepath); os.IsNotExist(err) {
		return "", "", errors.Wrap(err, "filesystem docker credentials missing")
	}

	input, err := ioutil.ReadFile(config.DynamicCredentialsFilepath)
	if err != nil {
		return "", "", errors.Wrap(err, "cannot read docker credentials")
	}

	username, password, err := credentials.ExtractDockerAuth(input, host)
	if err != nil {
		return "", "", errors.Wrapf(err, "cannot process filesystem docker auth for host %q", host)
	}

	return username, password, nil
}

func (job *Job) getDockerAuthFromSecret(ctx context.Context, host, name, namespace string) (string, string, error) {
	secret, err := job.clientk8s.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", "", errors.Wrap(err, "cannot find registry auth secret")
	}
	if secret.Type != corev1.SecretTypeDockerConfigJson {
		return "", "", fmt.Errorf("registry auth secret must be %v, not %v", corev1.SecretTypeDockerConfigJson, secret.Type)
	}

	username, password, err := credentials.ExtractDockerAuth(secret.Data[corev1.DockerConfigJsonKey], host)
	if err != nil {
		return "", "", errors.Wrap(err, "cannot process docker auth from secret")
	}

	return username, password, nil
}
