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
	mysqlv1alpha1 "github.com/cuppett/mysql-dba-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	_ "github.com/go-sql-driver/mysql"
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
	dbFinalizer = "mysql.brightframe.com/db-finalizer"
)

// DatabaseReconciler reconciles a Database object
type DatabaseReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// Custom variables used for the reconciliation loops
type DatabaseLoopContext struct {
	instance        *mysqlv1alpha1.Database
	adminConnection *mysqlv1alpha1.AdminConnection
	db              *sql.DB
}

// +kubebuilder:rbac:groups=mysql.apps.cuppett.dev,resources=databases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mysql.apps.cuppett.dev,resources=databases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=mysql.apps.cuppett.dev,resources=databases/finalizers,verbs=update
// +kubebuilder:rbac:groups=*,resources=secrets,verbs=list;get;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("database", req.NamespacedName)

	loop := DatabaseLoopContext{}

	// Fetch the Database instance
	loop.instance = &mysqlv1alpha1.Database{}
	err := r.Client.Get(ctx, req.NamespacedName, loop.instance)
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

	// Acquiring the database information
	loop.adminConnection = &mysqlv1alpha1.AdminConnection{}
	adminConnectionNamespacedName := types.NamespacedName{
		Namespace: loop.instance.Spec.AdminConnection.Namespace,
		Name:      loop.instance.Spec.AdminConnection.Name,
	}
	err = r.Client.Get(ctx, adminConnectionNamespacedName, loop.adminConnection)
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
	loop.db, err = loop.adminConnection.GetDatabaseConnection(ctx, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	} else {
		defer loop.db.Close()
	}

	// Check if the database instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isDatabaseMarkedToBeDeleted := loop.instance.GetDeletionTimestamp() != nil
	if isDatabaseMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(loop.instance, dbFinalizer) {
			// Run finalization logic for the database. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeDatabase(&loop); err != nil {
				return ctrl.Result{}, err
			}

			// Remove stacksFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(loop.instance, dbFinalizer)
			err := r.Update(ctx, loop.instance)
			if err != nil {
				r.Log.Error(err, "Failure removing the finalizer.")
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(loop.instance, dbFinalizer) {
		controllerutil.AddFinalizer(loop.instance, dbFinalizer)
		err = r.Update(ctx, loop.instance)
		if err != nil {
			r.Log.Error(err, "Failure adding the finalizer.")
		}
	} else {
		exists, err := r.databaseExists(&loop)
		if err != nil {
			return ctrl.Result{}, err
		}

		if !exists {
			created, err := r.databaseCreate(&loop)
			if err == nil && created {
				loop.instance.Status.CreationTime = metav1.NewTime(time.Now())
				loop.instance.Status.Message = "Created database"
				err = r.Status().Update(ctx, loop.instance)
				if err != nil {
					r.Log.Error(err, "Failure creating database.", "Name", loop.instance.Spec.Name)
				}
			}
		} else {
			updated, err := r.databaseUpdate(&loop)
			if err == nil && updated {
				loop.instance.Status.SyncTime = metav1.NewTime(time.Now())
				loop.instance.Status.Message = "Altered database"
				err = r.Status().Update(ctx, loop.instance)
				if err != nil {
					r.Log.Error(err, "Failure recording difference.")
				}
			}
		}
	}

	return ctrl.Result{}, err
}

func (r *DatabaseReconciler) databaseExists(loop *DatabaseLoopContext) (bool, error) {

	var collate string
	var characterSet string

	findStmt, err := loop.db.Prepare("SELECT DEFAULT_CHARACTER_SET_NAME, DEFAULT_COLLATION_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME=?")
	if err != nil {
		r.Log.Error(err, "Failed to prepare information schema query.", "Host",
			loop.adminConnection.Spec.Host, "Database", loop.instance.Spec.Name)
		return false, err
	}
	defer findStmt.Close()

	result := findStmt.QueryRow(loop.instance.Spec.Name)
	err = result.Scan(&characterSet, &collate)
	if err != nil {
		r.Log.Info("Database does not exist", "Host", loop.adminConnection.Spec.Host,
			"Name", loop.instance.Spec.Name)
		return false, nil
	}

	r.Log.Info("Successfully retrieved database", "Host", loop.adminConnection.Spec.Host,
		"Name", loop.instance.Spec.Name)
	if loop.instance.Status.Collate == "" || loop.instance.Status.Collate != collate {
		loop.instance.Status.Collate = collate
	}
	if loop.instance.Status.CharacterSet == "" || loop.instance.Status.CharacterSet != characterSet {
		loop.instance.Status.CharacterSet = characterSet
	}
	return true, nil
}

func (r *DatabaseReconciler) databaseUpdate(loop *DatabaseLoopContext) (bool, error) {

	var alterQuery string
	requireAlter := false

	alterQuery = "ALTER DATABASE " + loop.instance.Spec.Name
	if loop.instance.Spec.CharacterSet != "" && loop.instance.Spec.CharacterSet != loop.instance.Status.CharacterSet {
		requireAlter = true
		alterQuery += " CHARACTER SET " + loop.instance.Spec.CharacterSet
		loop.instance.Status.CharacterSet = loop.instance.Spec.CharacterSet
	}
	if loop.instance.Spec.Collate != "" && loop.instance.Spec.Collate != loop.instance.Status.Collate {
		requireAlter = true
		alterQuery += " COLLATE " + loop.instance.Spec.Collate
		loop.instance.Status.Collate = loop.instance.Spec.Collate
	}

	if requireAlter {
		r.Log.Info("Required to alter database", "Host", loop.adminConnection.Spec.Host, "Name",
			loop.instance.Spec.Name, "Query", alterQuery)
		_, err := loop.db.Exec(alterQuery)
		if err != nil {
			r.Log.Error(err, "Failed to alter database.", "Host", loop.adminConnection.Spec.Host,
				"Name", loop.instance.Spec.Name)
			return false, err
		}

		r.Log.Info("Successfully altered database", "Host", loop.adminConnection.Spec.Host,
			"Name", loop.instance.Spec.Name)
		if err != nil {
			r.Log.Error(err, "Failure recording altered state.")
			return false, err
		}
	}
	_, err := r.databaseExists(loop)
	return requireAlter, err
}

func (r *DatabaseReconciler) databaseCreate(loop *DatabaseLoopContext) (bool, error) {

	var createQuery string

	createQuery = "CREATE DATABASE " + loop.instance.Spec.Name
	if loop.instance.Spec.CharacterSet != "" {
		createQuery += " CHARACTER SET " + loop.instance.Spec.CharacterSet
		loop.instance.Status.CharacterSet = loop.instance.Spec.CharacterSet
	}
	if loop.instance.Spec.Collate != "" {
		createQuery += " COLLATE " + loop.instance.Spec.Collate
		loop.instance.Status.Collate = loop.instance.Spec.Collate
	}

	_, err := loop.db.Exec(createQuery)
	if err != nil {
		r.Log.Error(err, "Failed to create database.", "Host", loop.adminConnection.Spec.Host, "Name",
			loop.instance.Spec.Name, "Query", createQuery)
		return false, err
	}

	exists, err := r.databaseExists(loop)
	return exists, err
}

// This is the finalizer which will DROP the database from the server losing all data.
func (r *DatabaseReconciler) finalizeDatabase(loop *DatabaseLoopContext) error {

	_, err := loop.db.Exec("DROP DATABASE IF EXISTS " + loop.instance.Spec.Name)
	if err != nil {
		r.Log.Error(err, "Failed to delete the database")
	}
	r.Log.Info("Successfully, deleted database", "Host", loop.adminConnection.Spec.Host, "Name", loop.instance.Spec.Name)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mysqlv1alpha1.Database{}).
		Complete(r)
}
