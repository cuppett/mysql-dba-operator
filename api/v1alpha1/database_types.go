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

// Important: Run "make" to regenerate code after modifying this file

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DatabaseSpec defines the desired state of Database
type DatabaseSpec struct {
	AdminConnection AdminConnectionRef `json:"adminConnection"`
	// +kubebuilder:validation:MaxLength:=64
	// +kubebuilder:validation:MinLength:=1
	Name string `json:"name"`
	// +kubebuilder:validation:MaxLength:=64
	// +kubebuilder:validation:Optional
	// +nullable
	CharacterSet string `json:"characterSet,omitEmpty"`
	// +kubebuilder:validation:MaxLength:=64
	// +kubebuilder:validation:Optional
	Collate string `json:"collate,omitEmpty"`
}

// DatabaseStatus defines the observed state of Database
type DatabaseStatus struct {
	// Timestamp identifying when the database was successfully created
	// +kubebuilder:validation:Optional
	// +nullable
	CreationTime metav1.Time `json:"creationTime,omitEmpty"`
	// +kubebuilder:validation:Optional
	// +nullable
	SyncTime metav1.Time `json:"syncTime,omitEmpty"`
	// +kubebuilder:validation:Optional
	CharacterSet string `json:"defaultCharacterSet,omitEmpty"`
	// +kubebuilder:validation:Optional
	Collate string `json:"defaultCollation,omitEmpty"`
	// Indicates current state, phase or issue
	// +kubebuilder:validation:Optional
	Message string `json:"message,omitEmpty"`
	// +kubebuilder:validation:Optional
	// +nullable
	Name string `json:"name,omitempty"`
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
