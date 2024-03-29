/*
Copyright 2022.

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
	"reflect"
)

// DatabaseUserSpec defines the desired state of DatabaseUser
type DatabaseUserSpec struct {
	AdminConnection AdminConnectionRef `json:"adminConnection"`
	// +kubebuilder:validation:MaxLength:=32
	// +kubebuilder:validation:MinLength:=1
	Username string `json:"username"`
	// +kubebuilder:validation:Optional
	// +nullable
	Identification *Identification `json:"identification,omitEmpty"`
	// GRANT PRIVILEGES to the databases listed here
	// +kubebuilder:validation:Optional
	// +nullable
	DatabaseList []DatabasePermission `json:"databasePermissions,omitEmpty"`
	// +kubebuilder:validation:Optional
	// +nullable
	TlsOptions TlsOptions `json:"tlsOptions,omitEmpty"`
}

type DatabasePermission struct {
	Name string `json:"databaseName"`
	// Allows specifying a specific permission list here (empty string indicates ALL)
	// +kubebuilder:validation:Optional
	Grants []string `json:"grants"`
}

type Identification struct {
	// Relates to auth_plugin, See: MySQL CREATE USER
	// +kubebuilder:validation:Optional
	AuthPlugin string `json:"authPlugin"`
	// Relates to auth_string, See: MySQL CREATE USER
	// +kubebuilder:validation:Optional
	// +nullable
	AuthString *SecretKeySource `json:"authString,omitEmpty"`
	// Indicates stored authString is not already hashed by the auth_plugin
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=true
	ClearText bool `json:"clearText"`
}

type TlsOptions struct {
	// Whether REQUIRE SSL or NONE
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	Required bool `json:"required"`
}

// DatabaseUserStatus defines the observed state of DatabaseUser
type DatabaseUserStatus struct {
	// Timestamp identifying when the database was successfully created
	// +kubebuilder:validation:Optional
	// +nullable
	CreationTime metav1.Time `json:"creationTime,omitEmpty"`
	// +kubebuilder:validation:Optional
	// +nullable
	SyncTime metav1.Time `json:"syncTime,omitEmpty"`
	// Indicates current state, phase or issue
	// +kubebuilder:validation:Optional
	Message string `json:"message,omitEmpty"`
	// Indicates the current username we're working with in the database.
	// +kubebuilder:validation:MaxLength:=32
	Username string `json:"username,omitEmpty"`
	// +kubebuilder:validation:Optional
	// +nullable
	DatabaseList []DatabasePermission `json:"databasePermissions,omitEmpty"`
	// Identifies the current permissions of the user as indicated by SHOW GRANTS
	// +kubebuilder:validation:Optional
	// +nullable
	Grants []string `json:"grants,omitEmpty"`
	// +kubebuilder:validation:Optional
	// +nullable
	Identification *Identification `json:"identification,omitEmpty"`
	// +kubebuilder:validation:Optional
	// +nullable
	IdentificationResourceVersion string `json:"identificationResourceVersion,omitEmpty"`
	// +kubebuilder:validation:Optional
	// +nullable
	TlsOptions TlsOptions `json:"tlsOptions,omitEmpty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// DatabaseUser is the Schema for the databaseusers API
type DatabaseUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatabaseUserSpec   `json:"spec,omitempty"`
	Status DatabaseUserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DatabaseUserList contains a list of DatabaseUser
type DatabaseUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DatabaseUser `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DatabaseUser{}, &DatabaseUserList{})
}

func (r *DatabaseUser) PermissionListEqual() bool {
	// Always has GRANT USAGE as the first one. Only when we have something more complicated than
	if len(r.Status.Grants) != len(r.Spec.DatabaseList) {
		return false
	}
	return reflect.DeepEqual(r.Spec.DatabaseList, r.Status.DatabaseList)
}
