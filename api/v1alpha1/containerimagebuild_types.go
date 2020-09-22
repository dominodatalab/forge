package v1alpha1

import (
	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BasicAuthConfig contains credentials either inline or a reference to a dockerconfigjson secret.
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

func (auth BasicAuthConfig) IsInline() bool {
	return auth.Username != "" && auth.Password != ""
}

func (auth BasicAuthConfig) IsSecret() bool {
	return auth.SecretName != "" && auth.SecretNamespace != ""
}

func (auth BasicAuthConfig) Validate() error {
	switch {
	case auth.Username == "" && auth.Password == "" && auth.SecretName == "" && auth.SecretNamespace == "":
		// no basic auth
		return nil
	case (auth.Username != "" && auth.Password == "") || (auth.Username == "" && auth.Password != ""):
		// partial inline auth
		return errors.New("inline basic auth requires both username and password")
	case (auth.SecretName != "" && auth.SecretNamespace == "") || (auth.SecretName == "" && auth.SecretNamespace != ""):
		// partial secret auth
		return errors.New("secret basic auth requires both secret name and namespace")
	case (auth.IsInline() || auth.IsSecret()) && (auth.IsInline() && auth.IsSecret()):
		// multiple credential types
		return errors.New("basic auth cannot be both inline and secret-based")
	}

	return nil
}

// Registry contains the parameters required to pull and/or push from an OCI distribution registry.
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

	// When enabled, the controller will request credentials from the specific cloud registry (AWS, GCP, Azure Cloud)
	// and provide them to the build job for authentication.
	// +kubebuilder:validation:Optional
	DynamicCloudCredentials bool `json:"dynamicCloudCredentials"`
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

	// Limits build cpu consumption.
	// +kubebuilder:validation:Optional
	CPU string `json:"cpu"`

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

	// Provide arbitrary data for use in plugins that extend default capabilities.
	// +kubebuilder:validation:Optional
	PluginData map[string]string `json:"pluginData"`

	// Disable the use of layer caches during build.
	// +kubebuilder:validation:Optional
	DisableBuildCache bool `json:"disableBuildCache"`

	// Disable export of layer cache when it is enabled.
	// +kubebuilder:validation:Optional
	DisableLayerCacheExport bool `json:"disableLayerCacheExport"`
}

// ContainerImageBuildStatus defines the observed state of ContainerImageBuild
type ContainerImageBuildStatus struct {
	PreviousState    BuildState   `json:"-"` // NOTE: should we persist this value?
	State            BuildState   `json:"state,omitempty"`
	ImageURLs        []string     `json:"imageURLs,omitempty"`
	ImageSize        uint64       `json:"imageSize,omitempty"`
	ErrorMessage     string       `json:"errorMessage,omitempty"`
	BuildStartedAt   *metav1.Time `json:"buildStartedAt,omitempty"`
	BuildCompletedAt *metav1.Time `json:"buildCompletedAt,omitempty"`
}

// SetStatus will set a new build state and preserve the previous state in a transient field.
// An initialized state will be set when no state is provided.
func (s *ContainerImageBuildStatus) SetState(state BuildState) {
	// NOTE: try to leverage kubebuilder default values on State later; currently doesn't work
	if s.State == "" {
		s.State = BuildStateInitialized
	}

	s.PreviousState = s.State
	s.State = state
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,shortName=cib
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Image Name",type="string",JSONPath=".spec.imageName"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state"
// +kubebuilder:printcolumn:name="Image URLs",type="string",priority=1,JSONPath=".status.imageURLs"

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
