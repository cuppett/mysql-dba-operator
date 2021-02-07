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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mysqlv1alpha1 "github.com/brightframe/mysql-database-operator/api/v1alpha1"
)

const (
	userFinalizer = "finalizer.user.mysql.brightframe.com"
)

// DatabaseUserReconciler reconciles a DatabaseUser object
type DatabaseUserReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// Custom variables used for the reconciliation loops
type UserLoopContext struct {
	instance        *mysqlv1alpha1.DatabaseUser
	adminConnection *mysqlv1alpha1.AdminConnection
	db              *sql.DB
}

// +kubebuilder:rbac:groups=mysql.brightframe.com,resources=databaseusers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mysql.brightframe.com,resources=databaseusers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=mysql.brightframe.com,resources=databaseusers/finalizers,verbs=update
// +kubebuilder:rbac:groups=*,resources=secrets,verbs=list;get;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile

func (r *DatabaseUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("databaseuser", req.NamespacedName)

	loop := UserLoopContext{}

	// Fetch the Database instance
	loop.instance = &mysqlv1alpha1.DatabaseUser{}
	err := r.Client.Get(ctx, req.NamespacedName, loop.instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			r.Log.Info("DatabaseUser resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		r.Log.Error(err, "Failed to get Database")
		return ctrl.Result{}, err
	}

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

	// Check if the user instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isUserMarkedToBeDeleted := loop.instance.GetDeletionTimestamp() != nil
	if isUserMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(loop.instance, userFinalizer) {
			// Run finalization logic for the database. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeUser(&loop); err != nil {
				return ctrl.Result{}, err
			}

			// Remove stacksFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(loop.instance, userFinalizer)
			err := r.Update(ctx, loop.instance)
			if err != nil {
				r.Log.Error(err, "Failure removing the finalizer.")
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	if loop.instance.Status.Username == "" {
		// Ensuring old/new username is always set.
		loop.instance.Status.Username = loop.instance.Spec.Username
		err = r.Status().Update(ctx, loop.instance)
	} else if !controllerutil.ContainsFinalizer(loop.instance, userFinalizer) {
		// Add finalizer for this CR
		controllerutil.AddFinalizer(loop.instance, userFinalizer)
		err = r.Update(ctx, loop.instance)
		if err != nil {
			r.Log.Error(err, "Failure adding the finalizer.")
		}
	} else {
		exists, err := r.userExists(&loop)

		if !exists {
			err = r.userCreate(ctx, &loop)
			if err != nil {
				return ctrl.Result{}, err
			}
			loop.instance.Status.CreationTime = metav1.NewTime(time.Now())
			loop.instance.Status.Message = "Created user"
			err = r.Status().Update(ctx, loop.instance)
		} else {
			updated, err := r.userUpdate(ctx, &loop)
			if err != nil {
				return ctrl.Result{}, err
			}
			if updated {
				loop.instance.Status.SyncTime = metav1.NewTime(time.Now())
				err = r.Status().Update(ctx, loop.instance)
			}
		}
		if err != nil {
			r.Log.Error(err, "Failure to reconcile user.")
		}
	}
	return ctrl.Result{}, err
}

func (r *DatabaseUserReconciler) userExists(loop *UserLoopContext) (bool, error) {

	var exists bool

	findStmt, err := loop.db.Prepare("SELECT EXISTS(SELECT 1 FROM mysql.user WHERE user = ?)")
	if err != nil {
		r.Log.Error(err, "Failed to prepare information schema query.", "Host",
			loop.adminConnection.Spec.Host, "User", loop.instance.Status.Username)
		return false, err
	}
	defer findStmt.Close()

	result := findStmt.QueryRow(loop.instance.Status.Username)
	err = result.Scan(&exists)
	if err != nil {
		r.Log.Error(err, "Failed retrieving user", "Host", loop.adminConnection.Spec.Host, "Name", loop.instance.Status.Username)
		return false, nil
	}

	if exists {
		r.Log.Info("Successfully retrieved user", "Host", loop.adminConnection.Spec.Host,
			"Name", loop.instance.Status.Username)
	} else {
		r.Log.Info("User does not exist", "Host", loop.adminConnection.Spec.Host, "Name", loop.instance.Status.Username)
	}
	return exists, nil
}

func (r *DatabaseUserReconciler) userCreate(ctx context.Context, loop *UserLoopContext) error {

	createQuery := "CREATE USER '" + mysqlv1alpha1.Escape(loop.instance.Status.Username) + "'"
	params := make([]interface{}, 0)

	userDetails, err := r.userDetailString(ctx, loop, false)
	if err != nil {
		return err
	}
	if userDetails != "" {
		createQuery += userDetails
	}

	err = r.runStmt(loop, createQuery, params...)
	if err != nil {
		return err
	}
	r.Log.Info("Successfully created user", "Host", loop.adminConnection.Spec.Host,
		"Name", loop.instance.Status.Username)

	_, err = r.grant(loop)
	return err
}

func (r *DatabaseUserReconciler) userUpdate(ctx context.Context, loop *UserLoopContext) (bool, error) {

	// Tolerate a user rename
	if loop.instance.Spec.Username != loop.instance.Status.Username {
		renameQuery := "RENAME USER '" + mysqlv1alpha1.Escape(loop.instance.Status.Username) + "' TO '" +
			mysqlv1alpha1.Escape(loop.instance.Spec.Username) + "'"
		err := r.runStmt(loop, renameQuery)
		if err != nil {
			return false, err
		}
		r.Log.Info("Successfully renamed user", "Host", loop.adminConnection.Spec.Host,
			"Old", loop.instance.Status.Username, "New", loop.instance.Spec.Username)
		loop.instance.Status.Username = loop.instance.Spec.Username
		loop.instance.Status.Message = "User renamed"
		return true, nil
	}

	// Determining if we need to update the user wrt their authentication
	userDetails, err := r.userDetailString(ctx, loop, true)
	if err != nil {
		return false, err
	}
	if userDetails != "" {
		alterQuery := "ALTER USER '" + mysqlv1alpha1.Escape(loop.instance.Status.Username) + "'" + userDetails
		params := make([]interface{}, 0)
		err = r.runStmt(loop, alterQuery, params...)
		if err != nil {
			return false, err
		}
		r.Log.Info("Successfully updated user", "Host", loop.adminConnection.Spec.Host,
			"Name", loop.instance.Status.Username)
		loop.instance.Status.Message = "User altered"
		return true, nil
	}

	// Determining if we have a permissions thing and need to do something there.
	permsDiff, err := r.grantStatusUpdate(loop, false)
	// Always has GRANT USAGE as the first one. Only when we have something more complicated than
	if err == nil && (permsDiff || !loop.instance.PermissionListEqual()) {
		permsDiff = true
		r.Log.Info("Permissions difference.", "Host", loop.adminConnection.Spec.Host,
			"Name", loop.instance.Status.Username)
		// Always has GRANT USAGE as the first one. Only when we have something more complicated than
		// that do we need to revoke it.
		if len(loop.instance.Status.Grants) > 1 {
			_, err = r.revoke(loop)
		}
		if err == nil {
			_, err = r.grant(loop)
		}
	}
	return permsDiff, err
}

func (r *DatabaseUserReconciler) userDetailString(ctx context.Context, loop *UserLoopContext, update bool) (string, error) {

	var err error
	queryFragment := ""
	authString := ""

	if loop.instance.Spec.Identification != nil {
		if loop.instance.Spec.Identification.AuthString != nil {
			authString, err = mysqlv1alpha1.GetSecretRefValue(ctx, r.Client, loop.instance.Namespace,
				&loop.instance.Spec.Identification.AuthString.SecretKeyRef)
			if err != nil {
				r.Log.Error(err, "Failed to read user auth string", "Host",
					loop.adminConnection.Spec.Host, "Name", loop.instance.Status.Username, "Secret",
					loop.instance.Spec.Identification.AuthString.SecretKeyRef.Name)
				return queryFragment, err
			}
			authString = mysqlv1alpha1.Escape(authString)
		}

		// TODO: We should validate the authplugin against those actually installed in the database (webhook?)
		if loop.instance.Spec.Identification.AuthPlugin != "" {
			queryFragment += " IDENTIFIED WITH " + loop.instance.Spec.Identification.AuthPlugin
			if authString != "" {
				// TODO: We could actually check the token here by seeing if the secret was updated since SyncTime
				// (update && loop.instance.Spec.Identification.ClearText) would require the time check.
				if loop.instance.Spec.Identification.ClearText {
					queryFragment += " BY '" + authString + "'"
				} else {
					queryFragment += " AS '" + authString + "'"
				}
			}
		} else {
			// TODO: We could actually check the token here by seeing if the secret was updated since SyncTime
			// (update && loop.instance.Spec.Identification.ClearText) would require the time check.
			if authString != "" && !update {
				queryFragment += " IDENTIFIED BY"
				if !loop.instance.Spec.Identification.ClearText {
					queryFragment += " PASSWORD"
				}
				queryFragment += " '" + authString + "'"
			}
		}
	}

	return queryFragment, nil
}

func (r *DatabaseUserReconciler) revoke(loop *UserLoopContext) (bool, error) {

	var err error
	revokeQuery := "REVOKE ALL PRIVILEGES, GRANT OPTION FROM '" + mysqlv1alpha1.Escape(loop.instance.Status.Username) + "'"

	err = r.runStmt(loop, revokeQuery)
	if err != nil {
		r.Log.Error(err, "Failed to revoke user permissions", "Host",
			loop.adminConnection.Spec.Host, "Name", loop.instance.Status.Username, "Query",
			revokeQuery)
		return false, err
	}
	loop.instance.Status.DatabaseList = make([]mysqlv1alpha1.DatabasePermission, 0)
	return r.grantStatusUpdate(loop, true)
}

func (r *DatabaseUserReconciler) grant(loop *UserLoopContext) (bool, error) {

	var err error
	var grantQuery string

	for _, database := range loop.instance.Spec.DatabaseList {
		grantQuery = "GRANT ALL ON " + database.Name + ".* TO '" + mysqlv1alpha1.Escape(loop.instance.Status.Username) + "'"
		err = r.runStmt(loop, grantQuery)
		if err != nil {
			r.Log.Error(err, "Failed to grant user permissions", "Host",
				loop.adminConnection.Spec.Host, "Name", loop.instance.Status.Username, "Query",
				grantQuery)
			return false, err
		}
	}

	loop.instance.Status.DatabaseList = loop.instance.Spec.DatabaseList
	return r.grantStatusUpdate(loop, false)
}

/**
 * Checking whether we need to reflect a newer set of grants back to the model
 * Compares the two sets of grants (status + database). The lists should be relatively small,
 * so we'll bruteforce it.
 * @return bool If the grants have been changed.
 * @return error In the event of a failure.
 */
func (r *DatabaseUserReconciler) grantStatusUpdate(loop *UserLoopContext, empty bool) (bool, error) {

	var grant string
	var update bool

	showQuery := "SHOW GRANTS FOR '" + mysqlv1alpha1.Escape(loop.instance.Status.Username) + "'"

	if empty {
		// Empty out the list. We're loading it fresh.
		loop.instance.Status.Grants = make([]string, 0)
	}

	rows, err := loop.db.Query(showQuery)
	if err != nil {
		r.Log.Error(err, "Failed to get user grants", "Host",
			loop.adminConnection.Spec.Host, "Name", loop.instance.Status.Username, "Query",
			showQuery)
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&grant); err != nil {
			r.Log.Error(err, "Failure retrieving grant row from SHOW GRANT", "Host",
				loop.adminConnection.Spec.Host, "Name", loop.instance.Status.Username, "Query",
				showQuery)
			return false, err
		}
		if !contains(loop.instance.Status.Grants, grant) {
			r.Log.Info("Existing grants do not contain this one.", "Grant", grant, "Host",
				loop.adminConnection.Spec.Host, "Name", loop.instance.Status.Username)
			update = true
			loop.instance.Status.Grants = append(loop.instance.Status.Grants, grant)
		}

	}
	return update, nil
}

func (r *DatabaseUserReconciler) runStmt(loop *UserLoopContext, query string, args ...interface{}) error {
	_, err := loop.db.Exec(query, args...)
	if err != nil {
		r.Log.Error(err, "Failed to execute query.", "Host", loop.adminConnection.Spec.Host,
			"Name", loop.instance.Status.Username, "Query", query)
		return err
	}
	return nil
}

// This is the finalizer which will DROP the database from the server losing all data.
func (r *DatabaseUserReconciler) finalizeUser(loop *UserLoopContext) error {

	_, err := loop.db.Exec("DROP USER IF EXISTS " + loop.instance.Status.Username)
	if err != nil {
		r.Log.Error(err, "Failed to delete the user")
		return err
	}
	r.Log.Info("Successfully, deleted user", "Host", loop.adminConnection.Spec.Host,
		"Name", loop.instance.Status.Username)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatabaseUserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mysqlv1alpha1.DatabaseUser{}).
		Complete(r)
}

// Contains tells whether a contains x.
func contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}
