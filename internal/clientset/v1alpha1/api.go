package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"github.com/dominodatalab/forge/api/v1alpha1"
)

type Client struct {
	restClient rest.Interface
}

func (c *Client) ContainerImageBuilds(namespace string) ContainerImageBuildInterface {
	return &containerImageBuilds{
		client: c.restClient,
		ns:     namespace,
	}
}

func NewForConfig(cfg *rest.Config) (*Client, error) {
	config := *cfg
	config.APIPath = "/apis"
	config.UserAgent = rest.DefaultKubernetesUserAgent()
	config.GroupVersion = &v1alpha1.GroupVersion
	config.NegotiatedSerializer = serializer.NewCodecFactory(scheme.Scheme)

	restClient, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	return &Client{restClient: restClient}, nil
}

func init() {
	if err := v1alpha1.AddToScheme(scheme.Scheme); err != nil {
		panic(err)
	}
}
