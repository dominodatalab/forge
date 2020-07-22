package buildjob

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	apiv1alpha1 "github.com/dominodatalab/forge/api/v1alpha1"
	"github.com/dominodatalab/forge/internal/builder"
	clientv1alpha1 "github.com/dominodatalab/forge/internal/clientset/v1alpha1"
	"github.com/dominodatalab/forge/internal/config"
	"github.com/dominodatalab/forge/internal/credentials"
	"github.com/dominodatalab/forge/internal/message"
	"github.com/dominodatalab/forge/plugins/preparer"

	_ "github.com/dominodatalab/forge/internal/unshare"
)

// NOTE: do we need/want a type here?

type Job struct {
	logger logr.Logger

	clientk8s   kubernetes.Interface
	clientforge clientv1alpha1.Interface

	producer message.Producer

	plugins []*preparer.Plugin

	builder builder.OCIImageBuilder

	name string

	cleanupSteps []func()
}

func New(cfg Config) (*Job, error) {
	reexec()

	log := NewLogger()

	// initialize kubernetes clients
	restCfg, err := loadKubernetesConfig()
	if err != nil {
		return nil, errors.Wrap(err, "cannot load k8s config")
	}
	clientsk8s, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create k8s api client")
	}
	clientforge, err := clientv1alpha1.NewForConfig(restCfg)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create forge api client")
	}

	var cleanupSteps []func()

	// setup message publisher
	var producer message.Producer
	if cfg.BrokerOpts != nil {
		if producer, err = message.NewProducer(cfg.BrokerOpts); err != nil {
			return nil, err
		}
		cleanupSteps = append(cleanupSteps, func() {
			log.Info("Closing message producer")
			producer.Close()
		})
	}

	// setup preparer plugins
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
	ociBuilder, err := builder.New(preparerPlugins, cfg.EnableLayerCaching, log)
	if err != nil {
		return nil, errors.Wrap(err, "image builder initialization failed")
	}

	return &Job{
		name:         cfg.ResourceName,
		logger:       log,
		clientk8s:    clientsk8s,
		clientforge:  clientforge,
		producer:     producer,
		plugins:      preparerPlugins,
		builder:      ociBuilder,
		cleanupSteps: cleanupSteps,
	}, nil
}

func (j *Job) Run() error {
	// fetch the build resource from k8s
	cib, err := j.clientforge.ContainerImageBuilds("default").Get(j.name) // NOTE: maybe we want to build two separate cibs???
	if err != nil {
		return errors.Wrapf(err, "cannot find containerimagebuild %s", j.name)
	}

	opts, err := j.generateBuildOptions(cib)
	if err != nil {
		return errors.Wrapf(err, "failed to generate build options")
	}
	fmt.Printf("%+v\n", opts)

	ctx := context.Background()
	images, err := j.builder.BuildAndPush(ctx, opts)
	if err != nil {
		return err
	}
	fmt.Println(images) // NOTE: this should be applied to the cib status

	return nil
}

func (j *Job) Cleanup() {
	for _, fn := range j.cleanupSteps {
		fn()
	}
}

func (j *Job) addCleanupStep(fn func()) {
	j.cleanupSteps = append(j.cleanupSteps, fn)
}

func (j *Job) generateBuildOptions(cib *apiv1alpha1.ContainerImageBuild) (*config.BuildOptions, error) {
	registries, err := j.buildRegistryConfig(context.TODO(), cib.Spec.Registries)
	if err != nil {
		return nil, errors.Wrap(err, "cannot build registry config")
	}

	opts := &config.BuildOptions{
		ContextURL:     cib.Spec.Context,
		ImageName:      cib.Spec.ImageName,
		ImageSizeLimit: cib.Spec.ImageSizeLimit,
		Labels:         cib.Spec.Labels,
		BuildArgs:      cib.Spec.BuildArgs,
		NoCache:        cib.Spec.NoCache,
		PushRegistries: cib.Spec.PushRegistries,
		PluginData:     cib.Spec.PluginData,
		Timeout:        time.Duration(cib.Spec.TimeoutSeconds) * time.Second,
		Registries:     registries,
	}
	return opts, nil
}

func (j *Job) buildRegistryConfig(ctx context.Context, apiRegs []apiv1alpha1.Registry) ([]config.Registry, error) {
	var configs []config.Registry
	for _, apiReg := range apiRegs {
		regConf := config.Registry{
			Host:   apiReg.Server,
			NonSSL: apiReg.NonSSL,
		}

		// NOTE: move BasicAuth validation into an admission webhook at a later time
		if err := apiReg.BasicAuth.Validate(); err != nil {
			return nil, errors.Wrap(err, "basic auth validation failed")
		}

		switch {
		case apiReg.BasicAuth.IsInline():
			regConf.Username = apiReg.BasicAuth.Username
			regConf.Password = apiReg.BasicAuth.Password
		case apiReg.BasicAuth.IsSecret():
			var err error
			regConf.Username, regConf.Password, err = j.getDockerAuthFromSecret(ctx, apiReg.Server, apiReg.BasicAuth.SecretName, apiReg.BasicAuth.SecretNamespace)
			if err != nil {
				return nil, err
			}
		}

		configs = append(configs, regConf)
	}

	return configs, nil
}

func (j *Job) getDockerAuthFromSecret(ctx context.Context, host, name, namespace string) (string, string, error) {
	// ctx is currently unused: https://github.com/kubernetes/kubernetes/pull/87299
	secret, err := j.clientk8s.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return "", "", errors.Wrap(err, "cannot find registry auth secret")
	}

	if secret.Type != corev1.SecretTypeDockerConfigJson {
		return "", "", fmt.Errorf("registry auth secret must be %v, not %v", corev1.SecretTypeDockerConfigJson, secret.Type)
	}

	input := secret.Data[corev1.DockerConfigJsonKey]
	var output credentials.DockerConfigJSON
	if err := json.Unmarshal(input, &output); err != nil {
		return "", "", errors.Wrap(err, "cannot parse docker config in registry secret")
	}

	auth, ok := output.Auths[host]
	if !ok {
		var urls []string
		for k, _ := range output.Auths {
			urls = append(urls, k)
		}
		return "", "", fmt.Errorf("registry server %q is not in registry secret %q: server list %v", host, name, urls)
	}

	return auth.Username, auth.Password, nil
}
