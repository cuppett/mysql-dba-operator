//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright 2023.

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
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AdminConnection) DeepCopyInto(out *AdminConnection) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AdminConnection.
func (in *AdminConnection) DeepCopy() *AdminConnection {
	if in == nil {
		return nil
	}
	out := new(AdminConnection)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AdminConnection) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AdminConnectionList) DeepCopyInto(out *AdminConnectionList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]AdminConnection, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AdminConnectionList.
func (in *AdminConnectionList) DeepCopy() *AdminConnectionList {
	if in == nil {
		return nil
	}
	out := new(AdminConnectionList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AdminConnectionList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AdminConnectionRef) DeepCopyInto(out *AdminConnectionRef) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AdminConnectionRef.
func (in *AdminConnectionRef) DeepCopy() *AdminConnectionRef {
	if in == nil {
		return nil
	}
	out := new(AdminConnectionRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AdminConnectionSpec) DeepCopyInto(out *AdminConnectionSpec) {
	*out = *in
	if in.AdminUser != nil {
		in, out := &in.AdminUser, &out.AdminUser
		*out = new(SecretKeySource)
		(*in).DeepCopyInto(*out)
	}
	if in.AdminPassword != nil {
		in, out := &in.AdminPassword, &out.AdminPassword
		*out = new(SecretKeySource)
		(*in).DeepCopyInto(*out)
	}
	if in.AllowedNamespaces != nil {
		in, out := &in.AllowedNamespaces, &out.AllowedNamespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AdminConnectionSpec.
func (in *AdminConnectionSpec) DeepCopy() *AdminConnectionSpec {
	if in == nil {
		return nil
	}
	out := new(AdminConnectionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AdminConnectionStatus) DeepCopyInto(out *AdminConnectionStatus) {
	*out = *in
	in.SyncTime.DeepCopyInto(&out.SyncTime)
	if in.AvailableCharsets != nil {
		in, out := &in.AvailableCharsets, &out.AvailableCharsets
		*out = make([]Charset, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AdminConnectionStatus.
func (in *AdminConnectionStatus) DeepCopy() *AdminConnectionStatus {
	if in == nil {
		return nil
	}
	out := new(AdminConnectionStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Charset) DeepCopyInto(out *Charset) {
	*out = *in
	if in.Collations != nil {
		in, out := &in.Collations, &out.Collations
		*out = make([]Collation, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Charset.
func (in *Charset) DeepCopy() *Charset {
	if in == nil {
		return nil
	}
	out := new(Charset)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Collation) DeepCopyInto(out *Collation) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Collation.
func (in *Collation) DeepCopy() *Collation {
	if in == nil {
		return nil
	}
	out := new(Collation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Database) DeepCopyInto(out *Database) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Database.
func (in *Database) DeepCopy() *Database {
	if in == nil {
		return nil
	}
	out := new(Database)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Database) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DatabaseList) DeepCopyInto(out *DatabaseList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Database, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DatabaseList.
func (in *DatabaseList) DeepCopy() *DatabaseList {
	if in == nil {
		return nil
	}
	out := new(DatabaseList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DatabaseList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DatabasePermission) DeepCopyInto(out *DatabasePermission) {
	*out = *in
	if in.Grants != nil {
		in, out := &in.Grants, &out.Grants
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DatabasePermission.
func (in *DatabasePermission) DeepCopy() *DatabasePermission {
	if in == nil {
		return nil
	}
	out := new(DatabasePermission)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DatabaseSpec) DeepCopyInto(out *DatabaseSpec) {
	*out = *in
	out.AdminConnection = in.AdminConnection
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DatabaseSpec.
func (in *DatabaseSpec) DeepCopy() *DatabaseSpec {
	if in == nil {
		return nil
	}
	out := new(DatabaseSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DatabaseStatus) DeepCopyInto(out *DatabaseStatus) {
	*out = *in
	in.CreationTime.DeepCopyInto(&out.CreationTime)
	in.SyncTime.DeepCopyInto(&out.SyncTime)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DatabaseStatus.
func (in *DatabaseStatus) DeepCopy() *DatabaseStatus {
	if in == nil {
		return nil
	}
	out := new(DatabaseStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DatabaseUser) DeepCopyInto(out *DatabaseUser) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DatabaseUser.
func (in *DatabaseUser) DeepCopy() *DatabaseUser {
	if in == nil {
		return nil
	}
	out := new(DatabaseUser)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DatabaseUser) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DatabaseUserList) DeepCopyInto(out *DatabaseUserList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]DatabaseUser, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DatabaseUserList.
func (in *DatabaseUserList) DeepCopy() *DatabaseUserList {
	if in == nil {
		return nil
	}
	out := new(DatabaseUserList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DatabaseUserList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DatabaseUserSpec) DeepCopyInto(out *DatabaseUserSpec) {
	*out = *in
	out.AdminConnection = in.AdminConnection
	if in.Identification != nil {
		in, out := &in.Identification, &out.Identification
		*out = new(Identification)
		(*in).DeepCopyInto(*out)
	}
	if in.DatabaseList != nil {
		in, out := &in.DatabaseList, &out.DatabaseList
		*out = make([]DatabasePermission, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	out.TlsOptions = in.TlsOptions
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DatabaseUserSpec.
func (in *DatabaseUserSpec) DeepCopy() *DatabaseUserSpec {
	if in == nil {
		return nil
	}
	out := new(DatabaseUserSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DatabaseUserStatus) DeepCopyInto(out *DatabaseUserStatus) {
	*out = *in
	in.CreationTime.DeepCopyInto(&out.CreationTime)
	in.SyncTime.DeepCopyInto(&out.SyncTime)
	if in.DatabaseList != nil {
		in, out := &in.DatabaseList, &out.DatabaseList
		*out = make([]DatabasePermission, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Grants != nil {
		in, out := &in.Grants, &out.Grants
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Identification != nil {
		in, out := &in.Identification, &out.Identification
		*out = new(Identification)
		(*in).DeepCopyInto(*out)
	}
	out.TlsOptions = in.TlsOptions
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DatabaseUserStatus.
func (in *DatabaseUserStatus) DeepCopy() *DatabaseUserStatus {
	if in == nil {
		return nil
	}
	out := new(DatabaseUserStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Identification) DeepCopyInto(out *Identification) {
	*out = *in
	if in.AuthString != nil {
		in, out := &in.AuthString, &out.AuthString
		*out = new(SecretKeySource)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Identification.
func (in *Identification) DeepCopy() *Identification {
	if in == nil {
		return nil
	}
	out := new(Identification)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecretKeySource) DeepCopyInto(out *SecretKeySource) {
	*out = *in
	in.SecretKeyRef.DeepCopyInto(&out.SecretKeyRef)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecretKeySource.
func (in *SecretKeySource) DeepCopy() *SecretKeySource {
	if in == nil {
		return nil
	}
	out := new(SecretKeySource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TlsOptions) DeepCopyInto(out *TlsOptions) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TlsOptions.
func (in *TlsOptions) DeepCopy() *TlsOptions {
	if in == nil {
		return nil
	}
	out := new(TlsOptions)
	in.DeepCopyInto(out)
	return out
}
