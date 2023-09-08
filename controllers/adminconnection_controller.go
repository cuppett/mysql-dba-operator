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
	"fmt"
	"github.com/cuppett/mysql-dba-operator/orm"
	"github.com/go-logr/logr"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	mysqlv1alpha1 "github.com/cuppett/mysql-dba-operator/api/v1alpha1"
)

// AdminConnectionReconciler reconciles a AdminConnection object
type AdminConnectionReconciler struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	Connections map[types.UID]*orm.ConnectionDefinition
}

// +kubebuilder:rbac:groups=mysql.apps.cuppett.dev,resources=adminconnections,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mysql.apps.cuppett.dev,resources=adminconnections/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=mysql.apps.cuppett.dev,resources=adminconnections/finalizers,verbs=update
// +kubebuilder:rbac:groups=*,resources=secrets,verbs=list;get;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *AdminConnectionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("AdminConnection", req.NamespacedName)

	// Fetch the AdminConnection instance
	instance := &mysqlv1alpha1.AdminConnection{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			r.Log.Info("AdminConnection resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		r.Log.Error(err, "Failed to get AdminConnection")
		return ctrl.Result{}, err
	}

	// No matter what, we're saving out the timestamp and the loop.
	instance.Status.SyncTime = metav1.NewTime(time.Now())
	defer r.Status().Update(ctx, instance)

	// Establish the database connection
	db, err := instance.GetDatabaseConnection(ctx, r.Client, r.Connections)
	if err != nil {
		instance.Status.Message = "Failed to connect or ping database"
		return ctrl.Result{}, err
	}

	instance.Status.CharacterSet, err = r.getVariable("character_set_server", db)
	if err != nil {
		instance.Status.Message = "Failed to retrieve default server character set"
		return ctrl.Result{}, err
	}

	instance.Status.Collation, err = r.getVariable("collation_server", db)
	if err != nil {
		instance.Status.Message = "Failed to retrieve default server collation"
		return ctrl.Result{}, err
	}

	instance.Status.AvailableCharsets, err = r.getCharSets(db)
	if err != nil {
		instance.Status.Message = "Failed to retrieve available character sets"
		return ctrl.Result{}, err
	}

	instance.Status.Message = "Successfully pinged database"
	instance.Status.ControlDatabase = orm.DatabaseName
	return ctrl.Result{}, nil
}

func (r *AdminConnectionReconciler) getVariable(name string, db *gorm.DB) (string, error) {

	var results []map[string]interface{}
	query := "SHOW VARIABLES LIKE '" + mysqlv1alpha1.Escape(name) + "'"
	tx := db.Raw(query).Scan(&results)
	if tx.Error != nil {
		return "", tx.Error
	}

	if len(results) != 1 {
		return "", fmt.Errorf("expected 1 row, got %v", len(results))
	}

	return results[0]["Value"].(string), nil
}

func (r *AdminConnectionReconciler) getCharSets(db *gorm.DB) ([]mysqlv1alpha1.Charset, error) {

	var toReturn []mysqlv1alpha1.Charset

	query := "SHOW COLLATION WHERE Charset IS NOT NULL"
	var results []map[string]interface{}
	tx := db.Raw(query).Scan(&results)
	if tx.Error != nil {
		return toReturn, tx.Error
	}

	var collation, charset string
	var isDefault bool
	var sortedResults map[string][]mysqlv1alpha1.Collation
	sortedResults = make(map[string][]mysqlv1alpha1.Collation)
	for _, row := range results {
		collation = row["Collation"].(string)
		charset = row["Charset"].(string)
		isDefault = row["Default"].(string) == "Yes"

		if _, ok := sortedResults[charset]; !ok {
			sortedResults[charset] = make([]mysqlv1alpha1.Collation, 0)
		}
		sortedResults[charset] = append(sortedResults[charset], mysqlv1alpha1.Collation{
			Name:    collation,
			Default: isDefault,
		})
	}

	for charset, collations := range sortedResults {
		toReturn = append(toReturn, mysqlv1alpha1.Charset{
			Name:       charset,
			Collations: collations,
		})
	}

	return toReturn, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AdminConnectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mysqlv1alpha1.AdminConnection{}).
		Complete(r)
}
