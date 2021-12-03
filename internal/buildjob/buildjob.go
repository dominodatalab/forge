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

func (j *Job) Run() error {
	ctx := context.Background()

	j.log.Info("Fetching ContainerImageBuild resource", "Name", j.name, "Namespace", j.namespace)
	cib, err := j.clientforge.ContainerImageBuilds(j.namespace).Get(ctx, j.name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "cannot find containerimagebuild %s", j.name)
	}

	if cib, err = j.transitionToBuilding(ctx, cib); err != nil {
		return err
	}

	j.log = j.log.WithValues("annotations", cib.Annotations)

	j.log.Info("Creating build options using custom resource fields")
	opts, err := j.generateBuildOptions(ctx, cib)
	if err != nil {
		err = errors.Wrap(err, "failed to generate build options")

		if iErr := j.transitionToFailure(ctx, cib, err); iErr != nil {
			err = errors.Wrap(err, iErr.Error())
		}
		return err
	}

	j.builder.SetLogger(j.log)
	images, err := j.builder.BuildAndPush(ctx, opts)
	if err != nil {
		logError(j.log, err)

		if iErr := j.transitionToFailure(ctx, cib, err); iErr != nil {
			err = errors.Wrap(err, iErr.Error())
		}
		return err
	}

	return j.transitionToComplete(ctx, cib, images)
}

func (j *Job) Cleanup(forced bool) {
	if forced {
		j.log.Info("Caught kill signal, cleaning up")
	}

	for _, fn := range j.cleanupSteps {
		fn()
	}
}

func (j *Job) generateBuildOptions(ctx context.Context, cib *v1alpha1.ContainerImageBuild) (*config.BuildOptions, error) {
	registries, err := j.buildRegistryConfigs(ctx, cib.Spec.Registries)
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
func (j *Job) buildRegistryConfigs(ctx context.Context, apiRegs []v1alpha1.Registry) (allRegistryConfigs []config.Registry, err error) {
	allRegistryConfigs = []config.Registry{}
	explicitlyConfiguredRegistries := map[string]bool{}

	logNewHostConfig := func(host string, source string) {
		j.log.Info("configured auth for host", "Host", host, "Source", source)
	}

	for _, apiReg := range apiRegs {
		// NOTE: move BasicAuth validation into an admission webhook at a later time
		if err := apiReg.BasicAuth.Validate(); err != nil {
			return nil, errors.Wrap(err, "basic auth validation failed")
		}

		explicitlyConfiguredRegistries[apiReg.Server] = true

		registryConfig := config.Registry{
			Host:   apiReg.Server,
			NonSSL: apiReg.NonSSL,
		}

		var fetchConfigs func() ([]config.Registry, error)

		switch {
		case apiReg.BasicAuth.IsInline():
			fetchConfigs = func() ([]config.Registry, error) {
				logNewHostConfig(apiReg.Server, "inline BasicAuth entry")
				registryConfig.Username = apiReg.BasicAuth.Username
				registryConfig.Password = apiReg.BasicAuth.Password
				return []config.Registry{registryConfig}, nil
			}

		case apiReg.BasicAuth.IsSecret():
			fetchConfigs = func() ([]config.Registry, error) {
				authConfigs, err := j.getDockerAuthsFromSecret(ctx, apiReg.BasicAuth.SecretName, apiReg.BasicAuth.SecretNamespace)
				if err != nil {
					return nil, err
				}

				username, password, err := j.getBasicAuthForHost(authConfigs, apiReg.Server)
				if err != nil {
					return nil, err
				}

				registryConfig.Username = username
				registryConfig.Password = password

				configs := []config.Registry{registryConfig}

				// load *all* hosts in the secret that are not already explicitly configured
				for host, authConfig := range authConfigs {
					if _, hostConfigured := explicitlyConfiguredRegistries[host]; hostConfigured {
						continue
					}

					logNewHostConfig(
						host,
						fmt.Sprintf("secret (%s) in namespace (%s)", apiReg.BasicAuth.SecretName, apiReg.BasicAuth.SecretNamespace))

					// Assume SSL. To configure NonSSL auth, user must manually specify
					// the specific host in the custom resource.
					configs = append(configs, config.Registry{
						Host:     host,
						NonSSL:   false,
						Username: authConfig.Username,
						Password: authConfig.Password,
					})
				}

				return configs, nil
			}

		case apiReg.DynamicCloudCredentials:
			fetchConfigs = func() ([]config.Registry, error) {
				authConfigs, err := j.getDockerAuthsFromFS(apiReg.Server)
				if err != nil {
					return nil, err
				}

				username, password, err := j.getBasicAuthForHost(authConfigs, apiReg.Server)
				if err != nil {
					return nil, err
				}

				registryConfig.Username = username
				registryConfig.Password = password

				logNewHostConfig(apiReg.Server, "dynamic cloud credentials")
				return []config.Registry{registryConfig}, nil
			}

		default:
			// If no recognizable auth config is present, configure host without authentication.
			fetchConfigs = func() ([]config.Registry, error) {
				logNewHostConfig(apiReg.Server, "no source, configured without auth")
				return []config.Registry{registryConfig}, nil
			}

		}

		registryConfigs, err := fetchConfigs()
		if err != nil {
			return nil, err
		}

		allRegistryConfigs = append(allRegistryConfigs, registryConfigs...)
	}

	return allRegistryConfigs, nil
}

func (j *Job) getDockerAuthsFromFS(host string) (credentials.AuthConfigs, error) {
	if _, err := os.Stat(config.DynamicCredentialsFilepath); os.IsNotExist(err) {
		return nil, errors.Wrap(err, "filesystem docker credentials missing")
	}

	input, err := ioutil.ReadFile(config.DynamicCredentialsFilepath)
	if err != nil {
		return nil, errors.Wrap(err, "cannot read docker credentials")
	}

	authConfigs, err := credentials.ExtractAuthConfigs(input)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to read docker auth from filesystem")
	}

	return authConfigs, nil
}

func (j *Job) getDockerAuthsFromSecret(ctx context.Context, secretName string, secretNamespace string) (credentials.AuthConfigs, error) {
	secret, err := j.clientk8s.CoreV1().Secrets(secretNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "cannot find registry auth secret")
	}
	if secret.Type != corev1.SecretTypeDockerConfigJson {
		return nil, fmt.Errorf("registry auth secret must be %v, not %v", corev1.SecretTypeDockerConfigJson, secret.Type)
	}

	authConfigs, err := credentials.ExtractAuthConfigs(secret.Data[corev1.DockerConfigJsonKey])
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read docker auth from secret")
	}

	return authConfigs, nil
}

func (j *Job) getBasicAuthForHost(authConfigs credentials.AuthConfigs, host string) (string, string, error) {
	username, password, err := credentials.ExtractBasicAuthForHost(authConfigs, host)
	if err != nil {
		return "", "", errors.Wrapf(err, "Failed to extract docker auth for specified host (%q)", host)
	}

	return username, password, nil
}
