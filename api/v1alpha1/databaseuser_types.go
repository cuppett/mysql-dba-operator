/*
Copyright 2021.

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

// DatabaseUserSpec defines the desired state of DatabaseUser
type DatabaseUserSpec struct {
	AdminConnection AdminConnectionRef `json:"adminConnection"`
	// TODO: Block or allow the rename of a user (currently would just CREATE new one if changed)
	// +kubebuilder:validation:MaxLength:=32
	// +kubebuilder:validation:MinLength:=1
	Username string `json:"username"`
	// +kubebuilder:validation:Optional
	// +nullable
	Identification *Identification `json:"identification,omitEmpty"`
	// Currently set to allow all via GRANT ALL PRIVILEGES for the databases listed here
	// +kubebuilder:validation:Optional
	// +nullable
	DatabaseList []DatabasePermission `json:"databasePermissions,omitEmpty"`
}

type DatabasePermission struct {
	Name string `json:"databaseName"`
}

type Identification struct {
	// Relates to auth_plugin, See: MySQL CREATE USER
	// +kubebuilder:validation:Optional
	AuthPlugin string `json:"authPlugin"`
	// Relates to auth_string, See: MySQL CREATE USER
	// TODO: We should watch this object, if it changes we can flush through a new password/token.
	// +kubebuilder:validation:Optional
	// +nullable
	AuthString *SecretKeySource `json:"authString,omitEmpty"`
	// Indicates stored authString is not already hashed by the auth_plugin
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=true
	ClearText bool `json:"clearText"`
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
	// +kubebuilder:validation:MinLength:=1
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
	if len(r.Status.Grants)-1 != len(r.Spec.DatabaseList) {
		return false
	}
	if len(r.Spec.DatabaseList) != len(r.Status.DatabaseList) {
		return false
	}
	for i, elem := range r.Spec.DatabaseList {
		if elem.Name != r.Status.DatabaseList[i].Name {
			return false
		}
	}
	return true
}
