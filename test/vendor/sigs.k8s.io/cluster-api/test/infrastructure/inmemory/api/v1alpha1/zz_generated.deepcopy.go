//go:build !ignore_autogenerated

/*
Copyright The Kubernetes Authors.

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
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/api/v1beta1"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *APIEndpoint) DeepCopyInto(out *APIEndpoint) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new APIEndpoint.
func (in *APIEndpoint) DeepCopy() *APIEndpoint {
	if in == nil {
		return nil
	}
	out := new(APIEndpoint)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CommonProvisioningSettings) DeepCopyInto(out *CommonProvisioningSettings) {
	*out = *in
	out.StartupDuration = in.StartupDuration
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CommonProvisioningSettings.
func (in *CommonProvisioningSettings) DeepCopy() *CommonProvisioningSettings {
	if in == nil {
		return nil
	}
	out := new(CommonProvisioningSettings)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InMemoryAPIServerBehaviour) DeepCopyInto(out *InMemoryAPIServerBehaviour) {
	*out = *in
	out.Provisioning = in.Provisioning
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InMemoryAPIServerBehaviour.
func (in *InMemoryAPIServerBehaviour) DeepCopy() *InMemoryAPIServerBehaviour {
	if in == nil {
		return nil
	}
	out := new(InMemoryAPIServerBehaviour)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InMemoryCluster) DeepCopyInto(out *InMemoryCluster) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InMemoryCluster.
func (in *InMemoryCluster) DeepCopy() *InMemoryCluster {
	if in == nil {
		return nil
	}
	out := new(InMemoryCluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InMemoryCluster) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InMemoryClusterList) DeepCopyInto(out *InMemoryClusterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]InMemoryCluster, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InMemoryClusterList.
func (in *InMemoryClusterList) DeepCopy() *InMemoryClusterList {
	if in == nil {
		return nil
	}
	out := new(InMemoryClusterList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InMemoryClusterList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InMemoryClusterSpec) DeepCopyInto(out *InMemoryClusterSpec) {
	*out = *in
	out.ControlPlaneEndpoint = in.ControlPlaneEndpoint
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InMemoryClusterSpec.
func (in *InMemoryClusterSpec) DeepCopy() *InMemoryClusterSpec {
	if in == nil {
		return nil
	}
	out := new(InMemoryClusterSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InMemoryClusterStatus) DeepCopyInto(out *InMemoryClusterStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make(v1beta1.Conditions, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InMemoryClusterStatus.
func (in *InMemoryClusterStatus) DeepCopy() *InMemoryClusterStatus {
	if in == nil {
		return nil
	}
	out := new(InMemoryClusterStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InMemoryClusterTemplate) DeepCopyInto(out *InMemoryClusterTemplate) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InMemoryClusterTemplate.
func (in *InMemoryClusterTemplate) DeepCopy() *InMemoryClusterTemplate {
	if in == nil {
		return nil
	}
	out := new(InMemoryClusterTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InMemoryClusterTemplate) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InMemoryClusterTemplateList) DeepCopyInto(out *InMemoryClusterTemplateList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]InMemoryClusterTemplate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InMemoryClusterTemplateList.
func (in *InMemoryClusterTemplateList) DeepCopy() *InMemoryClusterTemplateList {
	if in == nil {
		return nil
	}
	out := new(InMemoryClusterTemplateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InMemoryClusterTemplateList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InMemoryClusterTemplateResource) DeepCopyInto(out *InMemoryClusterTemplateResource) {
	*out = *in
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InMemoryClusterTemplateResource.
func (in *InMemoryClusterTemplateResource) DeepCopy() *InMemoryClusterTemplateResource {
	if in == nil {
		return nil
	}
	out := new(InMemoryClusterTemplateResource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InMemoryClusterTemplateSpec) DeepCopyInto(out *InMemoryClusterTemplateSpec) {
	*out = *in
	in.Template.DeepCopyInto(&out.Template)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InMemoryClusterTemplateSpec.
func (in *InMemoryClusterTemplateSpec) DeepCopy() *InMemoryClusterTemplateSpec {
	if in == nil {
		return nil
	}
	out := new(InMemoryClusterTemplateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InMemoryEtcdBehaviour) DeepCopyInto(out *InMemoryEtcdBehaviour) {
	*out = *in
	out.Provisioning = in.Provisioning
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InMemoryEtcdBehaviour.
func (in *InMemoryEtcdBehaviour) DeepCopy() *InMemoryEtcdBehaviour {
	if in == nil {
		return nil
	}
	out := new(InMemoryEtcdBehaviour)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InMemoryMachine) DeepCopyInto(out *InMemoryMachine) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InMemoryMachine.
func (in *InMemoryMachine) DeepCopy() *InMemoryMachine {
	if in == nil {
		return nil
	}
	out := new(InMemoryMachine)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InMemoryMachine) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InMemoryMachineBehaviour) DeepCopyInto(out *InMemoryMachineBehaviour) {
	*out = *in
	if in.VM != nil {
		in, out := &in.VM, &out.VM
		*out = new(InMemoryVMBehaviour)
		**out = **in
	}
	if in.Node != nil {
		in, out := &in.Node, &out.Node
		*out = new(InMemoryNodeBehaviour)
		**out = **in
	}
	if in.APIServer != nil {
		in, out := &in.APIServer, &out.APIServer
		*out = new(InMemoryAPIServerBehaviour)
		**out = **in
	}
	if in.Etcd != nil {
		in, out := &in.Etcd, &out.Etcd
		*out = new(InMemoryEtcdBehaviour)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InMemoryMachineBehaviour.
func (in *InMemoryMachineBehaviour) DeepCopy() *InMemoryMachineBehaviour {
	if in == nil {
		return nil
	}
	out := new(InMemoryMachineBehaviour)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InMemoryMachineList) DeepCopyInto(out *InMemoryMachineList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]InMemoryMachine, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InMemoryMachineList.
func (in *InMemoryMachineList) DeepCopy() *InMemoryMachineList {
	if in == nil {
		return nil
	}
	out := new(InMemoryMachineList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InMemoryMachineList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InMemoryMachineSpec) DeepCopyInto(out *InMemoryMachineSpec) {
	*out = *in
	if in.ProviderID != nil {
		in, out := &in.ProviderID, &out.ProviderID
		*out = new(string)
		**out = **in
	}
	if in.Behaviour != nil {
		in, out := &in.Behaviour, &out.Behaviour
		*out = new(InMemoryMachineBehaviour)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InMemoryMachineSpec.
func (in *InMemoryMachineSpec) DeepCopy() *InMemoryMachineSpec {
	if in == nil {
		return nil
	}
	out := new(InMemoryMachineSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InMemoryMachineStatus) DeepCopyInto(out *InMemoryMachineStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make(v1beta1.Conditions, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InMemoryMachineStatus.
func (in *InMemoryMachineStatus) DeepCopy() *InMemoryMachineStatus {
	if in == nil {
		return nil
	}
	out := new(InMemoryMachineStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InMemoryMachineTemplate) DeepCopyInto(out *InMemoryMachineTemplate) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InMemoryMachineTemplate.
func (in *InMemoryMachineTemplate) DeepCopy() *InMemoryMachineTemplate {
	if in == nil {
		return nil
	}
	out := new(InMemoryMachineTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InMemoryMachineTemplate) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InMemoryMachineTemplateList) DeepCopyInto(out *InMemoryMachineTemplateList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]InMemoryMachineTemplate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InMemoryMachineTemplateList.
func (in *InMemoryMachineTemplateList) DeepCopy() *InMemoryMachineTemplateList {
	if in == nil {
		return nil
	}
	out := new(InMemoryMachineTemplateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InMemoryMachineTemplateList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InMemoryMachineTemplateResource) DeepCopyInto(out *InMemoryMachineTemplateResource) {
	*out = *in
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InMemoryMachineTemplateResource.
func (in *InMemoryMachineTemplateResource) DeepCopy() *InMemoryMachineTemplateResource {
	if in == nil {
		return nil
	}
	out := new(InMemoryMachineTemplateResource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InMemoryMachineTemplateSpec) DeepCopyInto(out *InMemoryMachineTemplateSpec) {
	*out = *in
	in.Template.DeepCopyInto(&out.Template)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InMemoryMachineTemplateSpec.
func (in *InMemoryMachineTemplateSpec) DeepCopy() *InMemoryMachineTemplateSpec {
	if in == nil {
		return nil
	}
	out := new(InMemoryMachineTemplateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InMemoryNodeBehaviour) DeepCopyInto(out *InMemoryNodeBehaviour) {
	*out = *in
	out.Provisioning = in.Provisioning
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InMemoryNodeBehaviour.
func (in *InMemoryNodeBehaviour) DeepCopy() *InMemoryNodeBehaviour {
	if in == nil {
		return nil
	}
	out := new(InMemoryNodeBehaviour)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InMemoryVMBehaviour) DeepCopyInto(out *InMemoryVMBehaviour) {
	*out = *in
	out.Provisioning = in.Provisioning
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InMemoryVMBehaviour.
func (in *InMemoryVMBehaviour) DeepCopy() *InMemoryVMBehaviour {
	if in == nil {
		return nil
	}
	out := new(InMemoryVMBehaviour)
	in.DeepCopyInto(out)
	return out
}
