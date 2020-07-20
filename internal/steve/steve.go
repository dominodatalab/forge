package steve

import (
	"os"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	v1alpha12 "github.com/dominodatalab/forge/api/v1alpha1"
	"github.com/dominodatalab/forge/internal/clientset/v1alpha1"
	"github.com/dominodatalab/forge/internal/config"
	"github.com/dominodatalab/forge/internal/message"
)

//import _ "github.com/dominodatalab/forge/internal/unshare"

// TODO: turn these values into params
var (
	preparerPluginsPath = ""
	enableLayerCaching  = true
	brokerOpts          = &message.Options{}
)

func GoBuildSomething(name string, opts *config.BuildOptions) error {
	zapLog, err := zap.NewDevelopment()
	if err != nil {
		return err
	}
	log := zapr.NewLogger(zapLog)

	cfg, err := loadConfig()
	if err != nil {
		log.Error(err, "Could not initialize in-cluster Kubernetes config")
		os.Exit(1)
	}
	//clientset, err := kubernetes.NewForConfig(cfg)
	//if err != nil {
	//	log.Error(err, "Could not initialize Kubernetes client")
	//	os.Exit(1)
	//}
	//fmt.Println(clientset)

	client, err := v1alpha1.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}

	cib, err := client.ContainerImageBuilds("default").Get("test-tls-with-basic-auth")
	if err != nil {
		panic(err)
	}

	cib.Status.State = v1alpha12.Completed
	_, err = client.ContainerImageBuilds(cib.Namespace).UpdateStatus(cib)
	if err != nil {
		panic(err)
	}

	//
	//preparerPlugins, err := preparer.LoadPlugins(preparerPluginsPath)
	//if err != nil {
	//	log.Error(err, fmt.Sprintf("Unable to load preparer plugins path %q", preparerPluginsPath))
	//	os.Exit(1)
	//}
	//
	//defer func() {
	//	for _, preparerPlugin := range preparerPlugins {
	//		preparerPlugin.Kill()
	//	}
	//}()
	//
	//log.Info("Initializing OCI builder")
	//ociBuilder, err := builder.New(preparerPlugins, enableLayerCaching, log)
	//if err != nil {
	//	log.Error(err, "Image builder initialization failed")
	//	return err
	//}
	//
	//var publisher message.Producer
	//if brokerOpts != nil {
	//	log.Info("Initializing message publisher")
	//
	//	if publisher, err = message.NewProducer(brokerOpts); err != nil {
	//		log.Error(err, "Message publisher initialization failed")
	//		return err
	//	}
	//	defer publisher.Close()
	//}
	//
	//cfg, err := loadConfig()
	//if err != nil {
	//	log.Error(err, "Could not initialize in-cluster Kubernetes config")
	//	os.Exit(1)
	//}
	//clientset, err := kubernetes.NewForConfig(cfg)
	//if err != nil {
	//	log.Error(err, "Could not initialize Kubernetes client")
	//	os.Exit(1)
	//}
	//fmt.Println(clientset)
	//
	//ctx := context.Background()
	//images, err := ociBuilder.BuildAndPush(ctx, opts)
	//if err != nil {
	//	return err
	//}
	//
	//fmt.Println(images)
	return nil
}

func loadConfig() (*rest.Config, error) {
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	if cfg, err := kubeconfig.ClientConfig(); err == nil {
		return cfg, nil
	}

	return rest.InClusterConfig()
}

//func (r *ContainerImageBuildReconciler) getDockerAuthFromSecret(ctx context.Context, host, name, namespace string) (string, string, error) {
//	// ctx is currently unused: https://github.com/kubernetes/kubernetes/pull/87299
//	secret, err := r.Clientset.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
//	if err != nil {
//		return "", "", errors.Wrap(err, "cannot find registry auth secret")
//	}
//
//	if secret.Type != corev1.SecretTypeDockerConfigJson {
//		return "", "", fmt.Errorf("registry auth secret must be %v, not %v", corev1.SecretTypeDockerConfigJson, secret.Type)
//	}
//
//	input := secret.Data[corev1.DockerConfigJsonKey]
//	var output credentials.DockerConfigJSON
//	if err := json.Unmarshal(input, &output); err != nil {
//		return "", "", errors.Wrap(err, "cannot parse docker config in registry secret")
//	}
//
//	auth, ok := output.Auths[host]
//	if !ok {
//		var urls []string
//		for k, _ := range output.Auths {
//			urls = append(urls, k)
//		}
//		return "", "", fmt.Errorf("registry server %q is not in registry secret %q: server list %v", host, name, urls)
//	}
//
//	return auth.Username, auth.Password, nil
//}
