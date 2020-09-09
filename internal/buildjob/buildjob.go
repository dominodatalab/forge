package buildjob

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
)

type Job struct {
	log logr.Logger

	clientk8s   kubernetes.Interface
	clientforge clientv1alpha1.Interface

	publisher message.Publisher

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

	var publisher message.Publisher
	// setup message publisher
	if cfg.BrokerOpts != nil {
		log.Info("Initializing status update message publisher")

		publisher, _ := message.NewPublisher(cfg.BrokerOpts, log)

		cleanupSteps = append(cleanupSteps, func() {
			log.Info("Closing message publisher")
			publisher.Close()
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
		clientforge:  clientforge,
		publisher:     publisher,
		plugins:      preparerPlugins,
		builder:      ociBuilder,
		cleanupSteps: cleanupSteps,
	}, nil
}

func (j *Job) Run() error {
	ctx := context.TODO()

	j.log.Info("Fetching ContainerImageBuild resource", "Name", j.name, "Namespace", j.namespace)
	cib, err := j.clientforge.ContainerImageBuilds(j.namespace).Get(j.name)
	if err != nil {
		return errors.Wrapf(err, "cannot find containerimagebuild %s", j.name)
	}

	if cib, err = j.transitionToBuilding(cib); err != nil {
		return err
	}

	j.log = j.log.WithValues("annotations", cib.Annotations)

	j.log.Info("Creating build options using custom resource fields")
	opts, err := j.generateBuildOptions(ctx, cib)
	if err != nil {
		err = errors.Wrap(err, "failed to generate build options")

		if iErr := j.transitionToFailure(cib, err); iErr != nil {
			err = errors.Wrap(err, iErr.Error())
		}
		return err
	}

	j.builder.SetLogger(j.log)
	images, err := j.builder.BuildAndPush(ctx, opts)
	if err != nil {
		logError(j.log, err)

		if iErr := j.transitionToFailure(cib, err); iErr != nil {
			err = errors.Wrap(err, iErr.Error())
		}
		return err
	}

	return j.transitionToComplete(cib, images)
}

func (j *Job) Cleanup(forced bool) {
	if forced {
		j.log.Info("Caught kill signal, cleaning up")
	}

	for _, fn := range j.cleanupSteps {
		fn()
	}
}

func (j *Job) generateBuildOptions(ctx context.Context, cib *apiv1alpha1.ContainerImageBuild) (*config.BuildOptions, error) {
	registries, err := j.buildRegistryConfig(ctx, cib.Spec.Registries)
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
		for url := range output.Auths {
			urls = append(urls, url)
		}
		return "", "", fmt.Errorf("registry server %q is not in registry secret %q: server list %v", host, name, urls)
	}

	return auth.Username, auth.Password, nil
}

func (j *Job) transitionToBuilding(cib *apiv1alpha1.ContainerImageBuild) (*apiv1alpha1.ContainerImageBuild, error) {
	cib.Status.SetState(apiv1alpha1.BuildStateBuilding)
	cib.Status.BuildStartedAt = &metav1.Time{Time: time.Now()}

	return j.updateStatus(cib)
}

func (j *Job) transitionToComplete(cib *apiv1alpha1.ContainerImageBuild, images []string) error {
	cib.Status.SetState(apiv1alpha1.BuildStateCompleted)
	cib.Status.ImageURLs = images
	cib.Status.BuildCompletedAt = &metav1.Time{Time: time.Now()}

	_, err := j.updateStatus(cib)
	return err
}

func (j *Job) transitionToFailure(cib *apiv1alpha1.ContainerImageBuild, err error) error {
	cib.Status.SetState(apiv1alpha1.BuildStateFailed)
	cib.Status.ErrorMessage = err.Error()
	cib.Status.BuildCompletedAt = &metav1.Time{Time: time.Now()}

	_, err = j.updateStatus(cib)
	return err
}

func (j *Job) updateStatus(cib *apiv1alpha1.ContainerImageBuild) (*apiv1alpha1.ContainerImageBuild, error) {
	cib, err := j.clientforge.ContainerImageBuilds(j.namespace).UpdateStatus(cib)
	if err != nil {
		return nil, errors.Wrap(err, "unable to update status")
	}

	if j.publisher != nil {
		update := &StatusUpdate{
			Name:          cib.Name,
			Annotations:   cib.Annotations,
			ObjectLink:    strings.TrimSuffix(cib.GetSelfLink(), "/status"),
			PreviousState: string(cib.Status.PreviousState),
			CurrentState:  string(cib.Status.State),
			ImageURLs:     cib.Status.ImageURLs,
			ErrorMessage:  cib.Status.ErrorMessage,
		}
		if err := j.publisher.Push(update); err != nil {
			return nil, errors.Wrap(err, "unable to publish message")
		}
	}

	return cib, nil
}
