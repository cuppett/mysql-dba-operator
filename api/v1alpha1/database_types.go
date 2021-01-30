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

// Important: Run "make" to regenerate code after modifying this file

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DatabaseSpec defines the desired state of Database
type DatabaseSpec struct {

	// +kubebuilder:validation:Format:=hostname
	Host string `json:"host"`
	// +kubebuilder:default:=3306
	// +kubebuilder:validation:Minimum=1024
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:validation:Optional
	Port int32 `json:"port"`
	// +kubebuilder:validation:Optional
	// +nullable
	AdminUser *SecretKeySource `json:"adminUser,omitEmpty"`
	// +kubebuilder:validation:Optional
	// +nullable
	AdminPassword *SecretKeySource `json:"adminPassword,omitEmpty"`
	Name          string           `json:"name"`
	// +kubebuilder:default:=utf8mb4
	// +kubebuilder:validation:Optional
	CharacterSet string `json:"characterSet"`
	// +kubebuilder:default:=utf8mb4_general_ci
	// +kubebuilder:validation:Optional
	Collate string `json:"collate"`
}

type SecretKeySource struct {
	SecretKeyRef v1.SecretKeySelector `json:"secretKeyRef"`
}

// DatabaseStatus defines the observed state of Database
type DatabaseStatus struct {
	// Timestamp identifying when the database was successfully created
	CreationTime metav1.Time `json:"creationTime,omitEmpty"`
	// Indicates current state, phase or issue
	Message string `json:"message"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Database is the Schema for the databases API
type Database struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatabaseSpec   `json:"spec,omitempty"`
	Status DatabaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DatabaseList contains a list of Database
type DatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Database `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Database{}, &DatabaseList{})
}
