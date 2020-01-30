/*
Copyright 2020 Domino Data Lab, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BuildMetadata encapsulates the information required to perform a container image build.
type BuildMetadata struct {
	// +kubebuilder:validation:Pattern=.+:.+

	// The name used to build the image in the following format: <registry>/<image>:<tag>.
	// The image will be pushed to the registry at the end of a successful build.
	ImageURL string `json:"imageURL"`

	// +kubebuilder:validation:MinItems=1

	// The commands used to assemble an image, see https://docs.docker.com/engine/reference/builder/.
	Commands []string `json:"commands"`
}

// ContainerImageBuildSpec defines the desired state of ContainerImageBuild
type ContainerImageBuildSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Build BuildMetadata `json:"build"`
}

// ContainerImageBuildStatus defines the observed state of ContainerImageBuild
type ContainerImageBuildStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=cib

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
