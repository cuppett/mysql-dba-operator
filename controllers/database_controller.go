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
	mysqlv1alpha1 "github.com/brightframe/mysql-database-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/common/log"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)

const (
	dbFinalizer = "finalizer.db.mysql.brightframe.com"
)

// DatabaseReconciler reconciles a Database object
type DatabaseReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	adminConnection *mysqlv1alpha1.AdminConnection
}

// +kubebuilder:rbac:groups=mysql.brightframe.com,resources=databases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mysql.brightframe.com,resources=databases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=mysql.brightframe.com,resources=databases/finalizers,verbs=update
// +kubebuilder:rbac:groups=*,resources=secrets,verbs=list;get;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	_ = r.Log.WithValues("database", req.NamespacedName)

	// Fetch the Database instance
	instance := &mysqlv1alpha1.Database{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
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

	r.adminConnection = &mysqlv1alpha1.AdminConnection{}
	adminConnectionNamespacedName := types.NamespacedName{
		Namespace: instance.Spec.AdminConnection.Namespace,
		Name:      instance.Spec.AdminConnection.Name,
	}
	err = r.Client.Get(ctx, adminConnectionNamespacedName, r.adminConnection)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("AdminConnection resource not found. Object must be deleted")
			return ctrl.Result{}, err
		}
		// Error reading the object - requeue the request.
		r.Log.Error(err, "Failed to get AdminConnection")
		return ctrl.Result{}, err
	}

	// Establish the database connection
	db, err := r.adminConnection.GetDatabaseConnection(ctx, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	} else {
		defer db.Close()
	}

	// Check if the database instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isDatabaseMarkedToBeDeleted := instance.GetDeletionTimestamp() != nil
	if isDatabaseMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(instance, dbFinalizer) {
			// Run finalization logic for the database. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeDatabase(instance, db); err != nil {
				return ctrl.Result{}, err
			}

			// Remove stacksFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(instance, dbFinalizer)
			err := r.Update(ctx, instance)
			if err != nil {
				r.Log.Error(err, "Failure removing the finalizer.")
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(instance, dbFinalizer) {
		controllerutil.AddFinalizer(instance, dbFinalizer)
		err = r.Update(ctx, instance)
		if err != nil {
			r.Log.Error(err, "Failure adding the finalizer.")
		}
	} else {
		exists, err := r.databaseExists(ctx, instance, db)

		if !exists {
			_, err = r.databaseCreate(ctx, instance, db)
			if err != nil {
				return ctrl.Result{}, err
			}
		} else {
			_, err = r.databaseUpdate(instance, db)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *DatabaseReconciler) databaseExists(ctx context.Context, m *mysqlv1alpha1.Database, db *sql.DB) (bool, error) {

	var collate string
	var characterSet string

	findStmt, err := db.Prepare("SELECT DEFAULT_CHARACTER_SET_NAME, DEFAULT_COLLATION_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME=?")
	if err != nil {
		r.Log.Error(err, "Failed to prepare information schema query.", "Host", r.adminConnection.Spec.Host, "Database", m.Spec.Name)
		return false, err
	}
	defer findStmt.Close()

	result := findStmt.QueryRow(m.Spec.Name)
	err = result.Scan(&characterSet, &collate)
	if err != nil {
		r.Log.Info("Database does not exist", "Host", r.adminConnection.Spec.Host, "Name", m.Spec.Name)
		return false, nil
	}

	r.Log.Info("Successfully retrieved database", "Host", r.adminConnection.Spec.Host, "Name", m.Spec.Name)
	difference := false
	if m.Status.Collate == "" || m.Status.Collate != collate {
		difference = true
		m.Status.Collate = collate
	}
	if m.Status.CharacterSet == "" || m.Status.CharacterSet != characterSet {
		difference = true
		m.Status.CharacterSet = characterSet
	}
	if difference {
		m.Status.SyncTime = metav1.NewTime(time.Now())
		err = r.Status().Update(ctx, m)
		if err != nil {
			r.Log.Error(err, "Failure recording difference.")
		}
	}

	return true, nil
}

func (r *DatabaseReconciler) databaseUpdate(m *mysqlv1alpha1.Database, db *sql.DB) (bool, error) {

	var createQuery string
	requireAlter := false

	createQuery = "ALTER DATABASE " + m.Spec.Name
	if m.Spec.CharacterSet != "" && m.Spec.CharacterSet != m.Status.CharacterSet {
		requireAlter = true
		createQuery += " CHARACTER SET " + m.Spec.CharacterSet
		m.Status.CharacterSet = m.Spec.CharacterSet
	}
	if m.Spec.Collate != "" && m.Spec.Collate != m.Status.Collate {
		requireAlter = true
		createQuery += " COLLATE " + m.Spec.Collate
		m.Status.Collate = m.Spec.Collate
	}

	if requireAlter {
		r.Log.Info("Required to alter database", "Host", r.adminConnection.Spec.Host, "Name", m.Spec.Name, "Query", createQuery)
		_, err := db.Exec(createQuery)
		if err != nil {
			r.Log.Error(err, "Failed to alter database.", "Host", r.adminConnection.Spec.Host, "Name", m.Spec.Name)
			return false, err
		}

		r.Log.Info("Successfully altered database", "Host", r.adminConnection.Spec.Host, "Name", m.Spec.Name)
		m.Status.SyncTime = metav1.NewTime(time.Now())
		m.Status.Message = "Altered database"
		err = r.Status().Update(context.TODO(), m)
		if err != nil {
			log.Error(err, "Failure recording altered state.")
		}

	}

	return true, nil
}

func (r *DatabaseReconciler) databaseCreate(ctx context.Context, m *mysqlv1alpha1.Database, db *sql.DB) (bool, error) {

	var createQuery string

	createQuery = "CREATE DATABASE " + m.Spec.Name
	if m.Spec.CharacterSet != "" {
		createQuery += " CHARACTER SET " + m.Spec.CharacterSet
		m.Status.CharacterSet = m.Spec.CharacterSet
	}
	if m.Spec.Collate != "" {
		createQuery += " COLLATE " + m.Spec.Collate
		m.Status.Collate = m.Spec.Collate
	}

	_, err := db.Exec(createQuery)
	if err != nil {
		r.Log.Error(err, "Failed to create database.", "Host", r.adminConnection.Spec.Host, "Name", m.Spec.Name, "Query", createQuery)
		return false, err
	}

	exists, err := r.databaseExists(ctx, m, db)
	if exists {
		r.Log.Info("Successfully created database", "Host", r.adminConnection.Spec.Host, "Name", m.Spec.Name)
		m.Status.CreationTime = metav1.NewTime(time.Now())
		m.Status.Message = "Created database"
		err = r.Status().Update(ctx, m)
		if err != nil {
			r.Log.Error(err, "Failure recording created database.")
		}

	} else if err != nil {
		return false, err
	}

	return true, nil
}

// This is the finalizer which will DROP the database from the server losing all data.
func (r *DatabaseReconciler) finalizeDatabase(m *mysqlv1alpha1.Database, db *sql.DB) error {

	_, err := db.Exec("DROP DATABASE IF EXISTS " + m.Spec.Name)
	if err != nil {
		log.Error(err, "Failed to delete the database")
	}
	log.Info("Successfully, deleted database", "Host", r.adminConnection.Spec.Host, "Name", m.Spec.Name)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mysqlv1alpha1.Database{}).
		Complete(r)
}
