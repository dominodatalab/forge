package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:validation:Enum=Inline;Secret
type BasicAuthSource string
type BuildState string

const (
	BasicAuthInline BasicAuthSource = "Inline"
	BasicAuthSecret BasicAuthSource = "Secret"

	Building  BuildState = "Building"
	Completed BuildState = "Completed"
	Failed    BuildState = "Failed"
)

type Registry struct {
	// URL where an image should be pushed at the end of a successful build.
	// +kubebuilder:validation:MinLength=1
	URL string `json:"url"`

	// Push image to a plain HTTP registry.
	// +kubebuilder:validation:Optional
	Insecure bool `json:"insecure"`

	// Push to a registry with native basic auth enabled.
	// +kubebuilder:validation:Optional
	BasicAuth BasicAuthSource `json:"basicAuth"`

	// Inline basic auth username.
	// +kubebuilder:validation:Optional
	Username string `json:"username"`

	// Inline basic auth password.
	// +kubebuilder:validation:Optional
	Password string `json:"password"`

	// Name of secret containing dockerconfigjson credentials to registry.
	// +kubebuilder:validation:Optional
	SecretName string `json:"secretName"`

	// Namespace where credentials secret resides.
	// +kubebuilder:validation:Optional
	SecretNamespace string `json:"secretNamespace"`
}

// ContainerImageBuildSpec defines the desired state of ContainerImageBuild
type ContainerImageBuildSpec struct {
	// Push to one or more registries.
	// +kubebuilder:validation:MinItems=1
	Registries []Registry `json:"registries"`

	// Name used to build an image.
	// +kubebuilder:validation:MinLength=1
	ImageName string `json:"imageName"`

	// Build context for the image. This can be a local path or url.
	Context string `json:"context"`

	// Image build arguments.
	// +kubebuilder:validation:Optional
	BuildArgs []string `json:"buildArgs"`

	// Labels added to the image during build.
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels"`

	// Disable the use of cache layers for a build.
	// +kubebuilder:validation:Optional
	NoCache bool `json:"noCache"`

	// Limits build cpu consumption (value should be some value from 0 to 100_000).
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=10000
	// +kubebuilder:validation:Maximum=100000
	CpuQuota uint16 `json:"cpuQuota"`

	// Limits build memory consumption.
	// +kubebuilder:validation:Optional
	Memory string `json:"memory"`

	// Optional deadline in seconds for image build to complete (defaults to 300).
	// +kubebuilder:validation:Optional
	TimeoutSeconds uint16 `json:"timeoutSeconds"`

	// Prevents images larger than this size (in bytes) from being pushed to a registry. By default,
	// an image of any size will be pushed.
	// +kubebuilder:validation:Optional
	ImageSizeLimit uint64 `json:"imageSizeLimit"`
}

// ContainerImageBuildStatus defines the observed state of ContainerImageBuild
type ContainerImageBuildStatus struct {
	State            BuildState   `json:"state,omitempty"`
	ImageURLs        []string     `json:"imageURLs,omitempty"`
	ErrorMessage     string       `json:"errorMessage,omitempty"`
	BuildStartedAt   *metav1.Time `json:"buildStartedAt,omitempty"`
	BuildCompletedAt *metav1.Time `json:"buildCompletedAt,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=cib
// +kubebuilder:subresource:status

// ContainerImageBuild is the Schema for the containerimagebuilds API
type ContainerImageBuild struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ContainerImageBuildSpec   `json:"spec,omitempty"`
	Status ContainerImageBuildStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ContainerImageBuildList contains a list of ContainerImageBuild
type ContainerImageBuildList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ContainerImageBuild `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ContainerImageBuild{}, &ContainerImageBuildList{})
}
