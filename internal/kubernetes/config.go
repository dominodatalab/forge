package kubernetes

import (
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// LoadKubernetesConfig returns the canonical kubernetes config for use with a client
func LoadKubernetesConfig() (*rest.Config, error) {
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	if cfg, err := kubeconfig.ClientConfig(); err == nil {
		return cfg, nil
	}

	return rest.InClusterConfig()
}
