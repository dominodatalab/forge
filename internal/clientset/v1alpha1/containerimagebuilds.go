package v1alpha1

import (
	"k8s.io/client-go/rest"

	"github.com/dominodatalab/forge/api/v1alpha1"
)

const resourceName = "containerimagebuilds"

type ContainerImageBuildInterface interface {
	Get(name string) (*v1alpha1.ContainerImageBuild, error)
	Update(*v1alpha1.ContainerImageBuild) (*v1alpha1.ContainerImageBuild, error)
	UpdateStatus(*v1alpha1.ContainerImageBuild) (*v1alpha1.ContainerImageBuild, error)
}

type containerImageBuilds struct {
	client rest.Interface
	ns     string
}

func (c *containerImageBuilds) Get(name string) (*v1alpha1.ContainerImageBuild, error) {
	result := &v1alpha1.ContainerImageBuild{}
	err := c.client.
		Get().
		Namespace(c.ns).
		Resource(resourceName).
		Name(name).
		Do().
		Into(result)
	return result, err
}

func (c *containerImageBuilds) Update(cib *v1alpha1.ContainerImageBuild) (*v1alpha1.ContainerImageBuild, error) {
	result := &v1alpha1.ContainerImageBuild{}
	err := c.client.Put().
		Namespace(c.ns).
		Resource(resourceName).
		Name(cib.Name).
		Body(cib).
		Do().
		Into(result)
	return result, err
}

func (c *containerImageBuilds) UpdateStatus(cib *v1alpha1.ContainerImageBuild) (*v1alpha1.ContainerImageBuild, error) {
	result := &v1alpha1.ContainerImageBuild{}
	err := c.client.Put().
		Namespace(c.ns).
		Resource(resourceName).
		Name(cib.Name).
		SubResource("status").
		Body(cib).
		Do().
		Into(result)
	return result, err
}
