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
	"github.com/cuppett/mysql-dba-operator/orm"
	"github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"log"
	"os"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
	"time"
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
	// Indicates current database is set and ready
	// +kubebuilder:validation:Optional
	ControlDatabase string `json:"controlDatabase,omitEmpty"`
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

func (in *AdminConnection) getDbConfig(ctx context.Context, client client.Client) (mysql.Config, error) {
	var err error
	var dbConfig mysql.Config

	// Reading the admin connection details
	dbConfig.Net = "tcp"
	dbConfig.DBName = "mysql"
	dbConfig.ParseTime = true
	dbConfig.AllowNativePasswords = true
	dbConfig.TLSConfig = "preferred"
	dbConfig.Addr = in.Spec.Host + ":" + strconv.Itoa(int(in.Spec.Port))
	// Default the admin user to root if it was not specified by the definition
	dbConfig.User = "root"
	if in.Spec.AdminUser != nil {
		dbConfig.User, err = GetSecretRefValue(ctx, client, in.Namespace, &in.Spec.AdminUser.SecretKeyRef)
	}

	if err == nil {
		// Default the admin password to empty if it was not specified by the definition
		dbConfig.Passwd = ""
		if in.Spec.AdminPassword != nil {
			dbConfig.Passwd, err = GetSecretRefValue(ctx, client, in.Namespace, &in.Spec.AdminPassword.SecretKeyRef)
		}
	}
	return dbConfig, err
}

// GetDatabaseConnection This function handles setting up all the database stuff, so we're ready to talk to it.
func (in *AdminConnection) GetDatabaseConnection(ctx context.Context, client client.Client, cache map[types.UID]*orm.ConnectionDefinition) (*gorm.DB, error) {

	// Generate the current config we'd use for fresh connections.
	// This includes protocols, usernames, passwords, etc.
	dbConfig, err := in.getDbConfig(ctx, client)
	if err != nil {
		return nil, err
	}

	conn, ok := cache[in.UID]

	var rawDatabase *sql.DB
	newConnection := true

	// Need to DeepEquals the dbConfig against the existing connection.
	// Do a ping/close depending on if there's a match or a difference and an existing entry
	if ok && reflect.DeepEqual(conn.Config, dbConfig) {
		// Do a ping if there is a match
		rawDatabase, err = conn.DB.DB()
		if err == nil {
			err = rawDatabase.Ping()
			if err == nil {
				newConnection = false
			}
		}
	} else if ok {
		// Exists, but the configuration is no longer equal.
		// Do a close to get out of that pool.
		rawDatabase, err = conn.DB.DB()
		if err == nil {
			err = rawDatabase.Close()
		}
	}

	//TODO: Remove all other connection.Close() operations throughout the codebase
	if newConnection {
		delete(cache, in.UID)
		gormDB, err := in.createFreshConnection(ctx, dbConfig)
		if err != nil {
			return nil, err
		}
		newEntry := &orm.ConnectionDefinition{
			DB:     gormDB,
			Config: dbConfig,
		}
		cache[in.UID] = newEntry

		return gormDB, nil
	} else {
		return conn.DB, nil
	}

}

func (in *AdminConnection) createFreshConnection(ctx context.Context, dbConfig mysql.Config) (*gorm.DB, error) {

	db, err := sql.Open("mysql", dbConfig.FormatDSN())
	if err != nil {
		return nil, err
	}

	// Open doesn't open a connection. Validate DSN data and our connection
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,  // Slow SQL threshold
			LogLevel:                  logger.Error, // Log level
			IgnoreRecordNotFoundError: false,        // Ignore ErrRecordNotFound error for logger
			Colorful:                  false,        // Disable color
		},
	)
	gormDB, err := gorm.Open(gormmysql.New(gormmysql.Config{Conn: db}), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		defer func(db *sql.DB) {
			err := db.Close()
			if err != nil {
				// TODO: Increment a fail counter here
			}
		}(db)
		return nil, err
	}

	// Creating and switching to the control database.
	in.switchDatabase(ctx, gormDB)
	err = gormDB.AutoMigrate(&orm.ManagedDatabase{}, &orm.ManagedUser{})
	if err != nil {
		newLogger.Error(ctx, "Failed to migrate content for AdminConnection")
		defer func(db *sql.DB) {
			err := db.Close()
			if err != nil {
				// TODO: Increment a fail counter here
			}
		}(db)
		return nil, err
	}

	return gormDB, nil
}

// switchDatabase Ensuring the control database exists and also that we are using it on this connection going forward.
func (in *AdminConnection) switchDatabase(ctx context.Context, gormDB *gorm.DB) {

	var createQuery string

	// No specific rules about the collation or character sets here yet, just taking the server defaults.
	// TODO: Could allow setting these (and the name) in a generic Config class.
	createQuery = "CREATE DATABASE IF NOT EXISTS " + orm.DatabaseName
	gormDB.Exec(createQuery)

	createQuery = "USE " + orm.DatabaseName
	gormDB.Exec(createQuery)
}

// AllowedNamespace This function handles setting up all the database stuff so we're ready to talk to it.
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

func (in *AdminConnection) DatabaseMine(gormDB *gorm.DB, database *Database) bool {

	var managedDatabase orm.ManagedDatabase

	// If it doesn't exist, go ahead and take it!
	if orm.DatabaseExists(gormDB, database.Spec.Name) == nil {
		return true
	}

	// If it does exist, let's check the triple after fetching by UID
	gormDB.Limit(1).Find(&managedDatabase, "uuid = ?", string(database.UID))
	if managedDatabase.DatabaseName == database.Spec.Name &&
		managedDatabase.Name == database.Name &&
		managedDatabase.Namespace == database.Namespace {
		return true
	}
	return false
}

func (in *AdminConnection) UserMine(gormDB *gorm.DB, user *DatabaseUser) bool {

	var managedUser orm.ManagedUser

	// If it doesn't exist, go ahead and take it!
	if orm.UserExists(gormDB, user.Spec.Username) == nil {
		return true
	}

	// If it does exist, let's check the triple after fetching by UID
	gormDB.Limit(1).Find(&managedUser, "uuid = ?", string(user.UID))
	if managedUser.Username == user.Spec.Username &&
		managedUser.Name == user.Name &&
		managedUser.Namespace == user.Namespace {
		return true
	}
	return false
}
