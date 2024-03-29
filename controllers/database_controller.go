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

package controllers

import (
	"context"
	"fmt"
	mysqlv1alpha1 "github.com/cuppett/mysql-dba-operator/api/v1alpha1"
	"github.com/cuppett/mysql-dba-operator/orm"
	"github.com/go-logr/logr"
	_ "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

const (
	dbFinalizer = "mysql.apps.cuppett.dev/db-finalizer"
)

// DatabaseReconciler reconciles a Database object
type DatabaseReconciler struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	Connections map[types.UID]*orm.ConnectionDefinition
}

// DatabaseLoopContext Custom variables used for the reconciliation loops
type DatabaseLoopContext struct {
	instance        *mysqlv1alpha1.Database
	adminConnection *mysqlv1alpha1.AdminConnection
	db              *gorm.DB
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
	_ = r.Log.WithValues("Database", req.NamespacedName)

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

	// Getting admin connection
	var adminErr error
	loop.adminConnection, adminErr = mysqlv1alpha1.GetAdminConnection(ctx, r.Client, loop.instance.Namespace, loop.instance.Spec.AdminConnection)
	if adminErr == nil && loop.adminConnection != nil {
		// Grabbing actual database connection here.
		loop.db, adminErr = loop.adminConnection.GetDatabaseConnection(ctx, r.Client, r.Connections)
		if adminErr != nil {
			// This could be temporary, we have no way to really know.
			loop.adminConnection = nil
		}
	}

	// Check if the database instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isDatabaseMarkedToBeDeleted := loop.instance.GetDeletionTimestamp() != nil
	if isDatabaseMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(loop.instance, dbFinalizer) {
			// Run finalization logic for the database. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if loop.adminConnection != nil && loop.db != nil && loop.adminConnection.DatabaseMine(loop.db, loop.instance) {
				if err := r.finalizeDatabase(&loop); err != nil {
					return ctrl.Result{}, err
				}
			} else {
				r.Log.Info("Unable or not permitted to delete database, finalizing without dropping",
					"Host", loop.adminConnection.Spec.Host, "Name", loop.instance.Status.Name)
			}

			// Remove dbFinalizer. Once all finalizers have been
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

	if loop.adminConnection == nil {
		r.Log.Error(adminErr, "Failed to obtain AdminConnection or connection to database")
		loop.instance.Status.Message = "Failed to further reconcile against current admin connection."
		err = r.Status().Update(ctx, loop.instance)
		if err != nil {
			return ctrl.Result{}, err
		} else {
			return ctrl.Result{}, adminErr
		}
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

		loop.instance.Status.Host = loop.adminConnection.Spec.Host
		loop.instance.Status.Port = loop.adminConnection.Spec.Port
		loop.instance.Status.Name = loop.instance.Spec.Name
		loop.instance.Status.SyncTime = metav1.NewTime(time.Now())

		if !exists {
			created, err := r.databaseCreate(&loop)
			if err == nil && created {
				loop.instance.Status.CreationTime = metav1.NewTime(time.Now())
				loop.instance.Status.Message = "Created database"
			}
		} else if loop.adminConnection.DatabaseMine(loop.db, loop.instance) {
			updated, err := r.databaseUpdate(&loop)
			if err == nil && updated {
				loop.instance.Status.Message = "Altered database"
			} else if err == nil {
				loop.instance.Status.Message = "Database in sync"
			} else {
				loop.instance.Status.Message = "Failed to update database"
			}
		} else {
			loop.instance.Status.Message = "No permission to this database."
		}

		err = r.Status().Update(ctx, loop.instance)
		if err != nil {
			r.Log.Error(err, "Failure recording status.")
		}
	}

	return ctrl.Result{}, err
}

func (r *DatabaseReconciler) databaseExists(loop *DatabaseLoopContext) (bool, error) {

	schema := orm.DatabaseExists(loop.db, loop.instance.Spec.Name)
	if schema == nil {
		r.Log.Info("Database does not exist", "Host", loop.adminConnection.Spec.Host,
			"Name", loop.instance.Spec.Name)
		return false, nil
	}

	if loop.instance.Status.Collate == "" || loop.instance.Status.Collate != schema.DefaultCollation {
		loop.instance.Status.Collate = schema.DefaultCollation
	}
	if loop.instance.Status.CharacterSet == "" || loop.instance.Status.CharacterSet != schema.DefaultCharacterSet {
		loop.instance.Status.CharacterSet = schema.DefaultCharacterSet
	}
	return true, nil
}

func (r *DatabaseReconciler) databaseUpdate(loop *DatabaseLoopContext) (bool, error) {

	var alterQuery string
	requireAlter := false

	alterQuery = "ALTER DATABASE `" + loop.instance.Spec.Name + "`"
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
		tx := loop.db.Exec(alterQuery)
		if tx.Error != nil {
			r.Log.Error(tx.Error, "Failed to alter database.", "Host", loop.adminConnection.Spec.Host,
				"Name", loop.instance.Spec.Name)
			return false, tx.Error
		}
		r.Log.Info("Successfully altered database", "Host", loop.adminConnection.Spec.Host,
			"Name", loop.instance.Spec.Name)
	}
	_, err := r.databaseExists(loop)
	return requireAlter, err
}

func (r *DatabaseReconciler) databaseCreate(loop *DatabaseLoopContext) (bool, error) {

	var createQuery string

	createQuery = "CREATE DATABASE `" + loop.instance.Spec.Name + "`"
	if loop.instance.Spec.CharacterSet != "" {
		createQuery += " CHARACTER SET " + loop.instance.Spec.CharacterSet
		loop.instance.Status.CharacterSet = loop.instance.Spec.CharacterSet
	}
	if loop.instance.Spec.Collate != "" {
		createQuery += " COLLATE " + loop.instance.Spec.Collate
		loop.instance.Status.Collate = loop.instance.Spec.Collate
	}

	tx := loop.db.Exec(createQuery)
	if tx.Error != nil {
		r.Log.Error(tx.Error, "Failed to create database.", "Host", loop.adminConnection.Spec.Host, "Name",
			loop.instance.Spec.Name, "Query", createQuery)
		return false, tx.Error
	}

	managedDatabase := orm.ManagedDatabase{
		Uuid:         string(loop.instance.UID),
		Namespace:    loop.instance.Namespace,
		Name:         loop.instance.Name,
		DatabaseName: loop.instance.Spec.Name,
	}

	tx = tx.Create(&managedDatabase)
	if tx.Error != nil {
		r.Log.Error(tx.Error, "Failed to insert managed record.", "Host", loop.adminConnection.Spec.Host, "Name",
			loop.instance.Spec.Name)
	}
	tx.Commit()

	exists, err := r.databaseExists(loop)
	return exists, err
}

// This is the finalizer which will DROP the database from the server losing all data.
func (r *DatabaseReconciler) finalizeDatabase(loop *DatabaseLoopContext) error {

	tx := loop.db.Exec("DROP DATABASE IF EXISTS `" + loop.instance.Spec.Name + "`")
	if tx.Error != nil {
		r.Log.Error(tx.Error, "Failed to delete the database")
	}
	r.Log.Info("Successfully, deleted database", "Host", loop.adminConnection.Spec.Host, "Name", loop.instance.Spec.Name)

	loop.db.Delete(&orm.ManagedDatabase{}, "uuid = ?", fmt.Sprintf("%v", loop.instance.UID))

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mysqlv1alpha1.Database{}).
		Watches(&mysqlv1alpha1.AdminConnection{}, handler.EnqueueRequestsFromMapFunc(
			func(ctx context.Context, a client.Object) []reconcile.Request {
				return r.findObjectsForAdminConnection(ctx, a.(*mysqlv1alpha1.AdminConnection))
			},
		)).
		Complete(r)
}

func (r *DatabaseReconciler) findObjectsForAdminConnection(ctx context.Context, adminConnection *mysqlv1alpha1.AdminConnection) []reconcile.Request {

	// List all DatabaseUser objects in the same namespace
	databaseList := &mysqlv1alpha1.DatabaseList{}
	err := r.Client.List(ctx, databaseList, &client.ListOptions{})
	if err != nil {
		// handle error, perhaps log it
		return nil
	}

	// Prepare a list of reconcile requests
	var requests []reconcile.Request
	for _, db := range databaseList.Items {
		if db.Spec.AdminConnection.Name == adminConnection.Name &&
			db.Spec.AdminConnection.Namespace == adminConnection.Namespace {
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKeyFromObject(&db),
			})
		}
	}

	return requests
}
