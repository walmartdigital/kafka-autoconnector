// +build !ignore_autogenerated

// Code generated by operator-sdk. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ESSinkConnector) DeepCopyInto(out *ESSinkConnector) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ESSinkConnector.
func (in *ESSinkConnector) DeepCopy() *ESSinkConnector {
	if in == nil {
		return nil
	}
	out := new(ESSinkConnector)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ESSinkConnector) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ESSinkConnectorList) DeepCopyInto(out *ESSinkConnectorList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ESSinkConnector, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ESSinkConnectorList.
func (in *ESSinkConnectorList) DeepCopy() *ESSinkConnectorList {
	if in == nil {
		return nil
	}
	out := new(ESSinkConnectorList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ESSinkConnectorList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ESSinkConnectorSpec) DeepCopyInto(out *ESSinkConnectorSpec) {
	*out = *in
	out.Config = in.Config
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ESSinkConnectorSpec.
func (in *ESSinkConnectorSpec) DeepCopy() *ESSinkConnectorSpec {
	if in == nil {
		return nil
	}
	out := new(ESSinkConnectorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ESSinkConnectorStatus) DeepCopyInto(out *ESSinkConnectorStatus) {
	*out = *in
	in.LastUpdate.DeepCopyInto(&out.LastUpdate)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ESSinkConnectorStatus.
func (in *ESSinkConnectorStatus) DeepCopy() *ESSinkConnectorStatus {
	if in == nil {
		return nil
	}
	out := new(ESSinkConnectorStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KafkaConnectConfig) DeepCopyInto(out *KafkaConnectConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KafkaConnectConfig.
func (in *KafkaConnectConfig) DeepCopy() *KafkaConnectConfig {
	if in == nil {
		return nil
	}
	out := new(KafkaConnectConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *KafkaConnectConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KafkaConnectConfigList) DeepCopyInto(out *KafkaConnectConfigList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]KafkaConnectConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KafkaConnectConfigList.
func (in *KafkaConnectConfigList) DeepCopy() *KafkaConnectConfigList {
	if in == nil {
		return nil
	}
	out := new(KafkaConnectConfigList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *KafkaConnectConfigList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KafkaConnectConfigSpec) DeepCopyInto(out *KafkaConnectConfigSpec) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KafkaConnectConfigSpec.
func (in *KafkaConnectConfigSpec) DeepCopy() *KafkaConnectConfigSpec {
	if in == nil {
		return nil
	}
	out := new(KafkaConnectConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KafkaConnectConfigStatus) DeepCopyInto(out *KafkaConnectConfigStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KafkaConnectConfigStatus.
func (in *KafkaConnectConfigStatus) DeepCopy() *KafkaConnectConfigStatus {
	if in == nil {
		return nil
	}
	out := new(KafkaConnectConfigStatus)
	in.DeepCopyInto(out)
	return out
}
