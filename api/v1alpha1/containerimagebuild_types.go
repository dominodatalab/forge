package v1alpha1

import (
	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type BasicAuthConfig struct {
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

func (auth BasicAuthConfig) Validate() error {
	username := auth.Username != ""
	password := auth.Password != ""
	secretName := auth.SecretName != ""
	secretNamespace := auth.SecretNamespace != ""

	switch {
	case !username && !password && !secretName && !secretNamespace: // no basic auth provided
		return nil
	case (username && !password) || (!username && password): // inline auth validation
		return errors.New("inline basic auth requires both username and password")
	case (secretName && !secretNamespace) || (!secretName && secretNamespace): // secret auth validation
		return errors.New("secret basic auth requires both secret name and namespace")
	case (username && (secretName || secretNamespace)) ||
		(password && (secretName || secretNamespace)) ||
		(secretName && (username || password)) ||
		(secretNamespace && (username || password)):
		return errors.New("basic auth cannot be both inline and secret-based")
	}

	return nil
}

func (auth BasicAuthConfig) IsInline() bool {
	return auth.Username != "" && auth.Password != ""
}

func (auth BasicAuthConfig) IsSecret() bool {
	return auth.SecretName != "" && auth.SecretNamespace != ""
}

type Registry struct {
	// Registry hostname.
	// +kubebuilder:validation:MinLength=1
	Server string `json:"server"`

	// Push image to a plain HTTP registry.
	// +kubebuilder:validation:Optional
	NonSSL bool `json:"nonSSL"`

	// Configure basic authentication credentials for a registry.
	// +kubebuilder:validation:Optional
	BasicAuth BasicAuthConfig `json:"basicAuth"`
}

// ContainerImageBuildSpec defines the desired state of ContainerImageBuild
type ContainerImageBuildSpec struct {
	// Name used to build an image.
	// +kubebuilder:validation:MinLength=1
	ImageName string `json:"imageName"`

	// Build context for the image. This can be a local path or url.
	Context string `json:"context"`

	// Push to one or more registries.
	// +kubebuilder:validation:MinItems=1
	PushRegistries []string `json:"pushTo"`

	// Configure one or more registry hosts with special requirements.
	// +kubebuilder:validation:Optional
	Registries []Registry `json:"registries"`

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
	PreviousState    BuildState   `json:"-"` // NOTE: should we persist this value?
	State            BuildState   `json:"state,omitempty"`
	ImageURLs        []string     `json:"imageURLs,omitempty"`
	ErrorMessage     string       `json:"errorMessage,omitempty"`
	BuildStartedAt   *metav1.Time `json:"buildStartedAt,omitempty"`
	BuildCompletedAt *metav1.Time `json:"buildCompletedAt,omitempty"`
}

func (s *ContainerImageBuildStatus) SetState(state BuildState) {
	// NOTE: try to leverage kubebuilder default values on State later; currently doesn't work
	if s.State == "" {
		s.State = Initialized
	}

	s.PreviousState = s.State
	s.State = state
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
