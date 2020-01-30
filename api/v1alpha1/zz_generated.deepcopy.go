// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BuildMetadata) DeepCopyInto(out *BuildMetadata) {
	*out = *in
	if in.Commands != nil {
		in, out := &in.Commands, &out.Commands
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BuildMetadata.
func (in *BuildMetadata) DeepCopy() *BuildMetadata {
	if in == nil {
		return nil
	}
	out := new(BuildMetadata)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ContainerImageBuild) DeepCopyInto(out *ContainerImageBuild) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ContainerImageBuild.
func (in *ContainerImageBuild) DeepCopy() *ContainerImageBuild {
	if in == nil {
		return nil
	}
	out := new(ContainerImageBuild)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ContainerImageBuild) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ContainerImageBuildList) DeepCopyInto(out *ContainerImageBuildList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ContainerImageBuild, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ContainerImageBuildList.
func (in *ContainerImageBuildList) DeepCopy() *ContainerImageBuildList {
	if in == nil {
		return nil
	}
	out := new(ContainerImageBuildList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ContainerImageBuildList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ContainerImageBuildSpec) DeepCopyInto(out *ContainerImageBuildSpec) {
	*out = *in
	in.Build.DeepCopyInto(&out.Build)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ContainerImageBuildSpec.
func (in *ContainerImageBuildSpec) DeepCopy() *ContainerImageBuildSpec {
	if in == nil {
		return nil
	}
	out := new(ContainerImageBuildSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ContainerImageBuildStatus) DeepCopyInto(out *ContainerImageBuildStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ContainerImageBuildStatus.
func (in *ContainerImageBuildStatus) DeepCopy() *ContainerImageBuildStatus {
	if in == nil {
		return nil
	}
	out := new(ContainerImageBuildStatus)
	in.DeepCopyInto(out)
	return out
}