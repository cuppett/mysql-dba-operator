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
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	mysqlv1alpha1 "github.com/brightframe/mysql-database-operator/api/v1alpha1"
)

// AdminConnectionReconciler reconciles a AdminConnection object
type AdminConnectionReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=mysql.brightframe.com,resources=adminconnections,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mysql.brightframe.com,resources=adminconnections/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=mysql.brightframe.com,resources=adminconnections/finalizers,verbs=update
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
	db, err := instance.GetDatabaseConnection(ctx, r.Client)
	if err != nil {
		instance.Status.Message = "Failed to connect or ping database"
		return ctrl.Result{}, err
	} else {
		defer db.Close()
	}

	instance.Status.Message = "Successfully pinged database"
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AdminConnectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mysqlv1alpha1.AdminConnection{}).
		Complete(r)
}
