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

package controllers

import (
	"context"
	"database/sql"
	"fmt"
	mysqlv1alpha1 "github.com/brightframe/mysql-database-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strconv"
	"strings"
	"time"
)

const (
	dbFinalizer = "finalizer.db.mysql.brightframe.com"
)

// DatabaseReconciler reconciles a Database object
type DatabaseReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=mysql.brightframe.com,resources=databases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mysql.brightframe.com,resources=databases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=mysql.brightframe.com,resources=databases/finalizers,verbs=update
// +kubebuilder:rbac:groups=*,resources=secrets,verbs=list;get;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Database object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	_ = r.Log.WithValues("database", req.NamespacedName)

	// your logic here
	// Fetch the Stack instance
	instance := &mysqlv1alpha1.Database{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			r.Log.Info("Database resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		r.Log.Error(err, "Failed to get Database")
		return ctrl.Result{}, err
	}

	// Establish the database connection
	db, err := r.getDatabaseConnection(instance)
	if err != nil {
		return ctrl.Result{}, err
	} else {
		defer db.Close()
	}

	// Check if the database instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isDatabaseMarkedToBeDeleted := instance.GetDeletionTimestamp() != nil
	if isDatabaseMarkedToBeDeleted {
		if contains(instance.GetFinalizers(), dbFinalizer) {
			// Run finalization logic for the database. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := finalizeDatabase(r.Log, instance, db); err != nil {
				return ctrl.Result{}, err
			}

			// Remove stacksFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(instance, dbFinalizer)
			err := r.Update(context.TODO(), instance)
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	if !contains(instance.GetFinalizers(), dbFinalizer) {
		controllerutil.AddFinalizer(instance, dbFinalizer)
		r.Update(context.TODO(), instance)
	}

	_, err = r.databaseCreate(instance, db)
	if err != nil {
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *DatabaseReconciler) databaseCreate(m *mysqlv1alpha1.Database, db *sql.DB) (bool, error) {

	result, err := db.Exec("CREATE DATABASE IF NOT EXISTS " + m.Spec.Name + " CHARACTER SET " + m.Spec.CharacterSet + " COLLATE " + m.Spec.Collate)
	if err != nil {
		r.Log.Error(err, fmt.Sprintf("Failed to create database %s", m.Spec.Name))
		return false, err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 1 {
		r.Log.Info("Successfully, created database", "Host", m.Spec.Host, "Name", m.Spec.Name)
		m.Status.CreationTime = metav1.NewTime(time.Now())
		m.Status.Message = "Created database"
		r.Status().Update(context.TODO(), m)
		if err != nil {
			r.Log.Error(err, fmt.Sprintf("Failed to update status on database %s", m.Spec.Name))
		}
		return true, nil
	}

	if m.Status.Message == "" {
		m.Status.Message = "Database already exists"
		r.Status().Update(context.TODO(), m)
	}

	return false, nil
}

// This is the finalizer which will DROP the database from the server losing all data.
func finalizeDatabase(log logr.Logger, m *mysqlv1alpha1.Database, db *sql.DB) error {

	_, err := db.Exec("DROP DATABASE " + m.Spec.Name)
	if err != nil {
		// Exclude the "database doesn't exist" error
		if !strings.Contains(err.Error(), "Error 1008:") {
			log.Error(err, "Failed to delete the database")
			return err
		} else {
			log.Info("Database doesn't exist, no need for delete", "Host", m.Spec.Host, "Name", m.Spec.Name)
		}
	}
	log.Info("Successfully, deleted database", "Host", m.Spec.Host, "Name", m.Spec.Name)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mysqlv1alpha1.Database{}).
		Complete(r)
}

// This function handles setting up all the database stuff so we're ready to talk to it.
func (r *DatabaseReconciler) getDatabaseConnection(instance *mysqlv1alpha1.Database) (*sql.DB, error) {
	var err error
	var dbConfig mysql.Config

	// Reading the admin connection details
	dbConfig.Net = "tcp"
	dbConfig.DBName = "mysql"
	dbConfig.AllowNativePasswords = true
	dbConfig.Addr = instance.Spec.Host + ":" + strconv.Itoa(int(instance.Spec.Port))
	// Default the admin user to root if it was not specified by the definition
	dbConfig.User = "root"
	if instance.Spec.AdminUser != nil {
		dbConfig.User, err = getSecretRefValue(r.Client, instance.Namespace, &instance.Spec.AdminUser.SecretKeyRef)
		if err != nil {
			return nil, err
		}
	}
	// Default the admin password to empty if it was not specified by the definition
	dbConfig.Passwd = ""
	if instance.Spec.AdminPassword != nil {
		dbConfig.Passwd, err = getSecretRefValue(r.Client, instance.Namespace, &instance.Spec.AdminPassword.SecretKeyRef)
		if err != nil {
			return nil, err
		}
	}

	db, err := sql.Open("mysql", dbConfig.FormatDSN())
	if err != nil {
		r.Log.Error(err, "Failed to connect to database.")
		return nil, err
	}

	// Open doesn't open a connection. Validate DSN data and our connection
	err = db.Ping()
	if err != nil {
		r.Log.Error(err, "Failed to ping database.")
		defer db.Close()
		return nil, err
	}

	return db, nil
}

// getSecretRefValue returns the value of a secret in the supplied namespace
func getSecretRefValue(client client.Client, namespace string, secretSelector *v1.SecretKeySelector) (string, error) {

	var namespacedName types.NamespacedName

	namespacedName.Name = secretSelector.Name
	namespacedName.Namespace = namespace

	// Fetch the Stack instance
	secret := &v1.Secret{}
	err := client.Get(context.TODO(), namespacedName, secret)
	if err != nil {
		return "", err
	}
	if data, ok := secret.Data[secretSelector.Key]; ok {
		return string(data), nil
	}
	return "", fmt.Errorf("key %s not found in secret %s", secretSelector.Key, secretSelector.Name)

}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
