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
	"context"
	"database/sql"
	"github.com/go-sql-driver/mysql"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
)

// AdminConnectionSpec defines the desired state of AdminConnection
type AdminConnectionSpec struct {
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
	// +kubebuilder:validation:Optional
	// +nullable
	AllowedNamespaces []string `json:"allowedNamespaces,omitEmpty"`
}

// AdminConnectionStatus defines the observed state of AdminConnection
type AdminConnectionStatus struct {
	// +kubebuilder:validation:Optional
	// +nullable
	SyncTime metav1.Time `json:"syncTime,omitEmpty"`
	// Indicates current state, phase or issue
	// +kubebuilder:validation:Optional
	Message string `json:"message,omitEmpty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// AdminConnection is the Schema for the adminconnections API
type AdminConnection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AdminConnectionSpec   `json:"spec,omitempty"`
	Status AdminConnectionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AdminConnectionList contains a list of AdminConnection
type AdminConnectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AdminConnection `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AdminConnection{}, &AdminConnectionList{})
}

// This function handles setting up all the database stuff so we're ready to talk to it.
func (in *AdminConnection) GetDatabaseConnection(ctx context.Context, client client.Client) (*sql.DB, error) {
	var err error
	var dbConfig mysql.Config

	// Reading the admin connection details
	dbConfig.Net = "tcp"
	dbConfig.DBName = "mysql"
	dbConfig.AllowNativePasswords = true
	dbConfig.Addr = in.Spec.Host + ":" + strconv.Itoa(int(in.Spec.Port))
	// Default the admin user to root if it was not specified by the definition
	dbConfig.User = "root"
	if in.Spec.AdminUser != nil {
		dbConfig.User, err = GetSecretRefValue(ctx, client, in.Namespace, &in.Spec.AdminUser.SecretKeyRef)
		if err != nil {
			return nil, err
		}
	}
	// Default the admin password to empty if it was not specified by the definition
	dbConfig.Passwd = ""
	if in.Spec.AdminPassword != nil {
		dbConfig.Passwd, err = GetSecretRefValue(ctx, client, in.Namespace, &in.Spec.AdminPassword.SecretKeyRef)
		if err != nil {
			return nil, err
		}
	}

	db, err := sql.Open("mysql", dbConfig.FormatDSN())
	if err != nil {
		return nil, err
	}

	// Open doesn't open a connection. Validate DSN data and our connection
	err = db.Ping()
	if err != nil {
		defer db.Close()
		return nil, err
	}

	return db, nil
}

// This function handles setting up all the database stuff so we're ready to talk to it.
func (in *AdminConnection) AllowedNamespace(namespace string) bool {
	if namespace == in.Namespace {
		return true
	}

	for _, allowedNamespace := range in.Spec.AllowedNamespaces {
		if allowedNamespace == namespace {
			return true
		}
		if strings.HasSuffix(allowedNamespace, "*") {
			if strings.HasPrefix(namespace, strings.TrimSuffix(allowedNamespace, "*")) {
				return true
			}
		}
	}

	return false
}
