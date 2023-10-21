/*
Copyright 2021, 2023.

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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"regexp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	nameRegEx = regexp.MustCompile(`^[^\\/?%*:|"<>.]{1,64}$`)
)

// log is for logging in this package.
var databaseLog = logf.Log.WithName("database-resource")
var k8sClient client.Client

func (r *Database) SetupWebhookWithManager(mgr ctrl.Manager) error {
	k8sClient = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

type validationError struct {
	s string
}

func (e *validationError) Error() string {
	return e.s
}

// +kubebuilder:webhook:path=/validate-mysql-apps-cuppett-dev-v1alpha1-database,mutating=false,failurePolicy=fail,sideEffects=None,groups=mysql.apps.cuppett.dev,resources=databases,verbs=create;update;delete,versions=v1alpha1,name=vdatabase.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Database{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Database) ValidateCreate() (admission.Warnings, error) {
	databaseLog.Info("validate create", "namespace", r.Namespace, "name", r.Name)

	// See also: https://stackoverflow.com/questions/9537771/mysql-database-name-restrictions
	if !nameRegEx.MatchString(r.Spec.Name) {
		return nil, &validationError{"Invalid database name."}
	}

	return r.ValidateCharsetCollationCombo()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Database) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	databaseLog.Info("validate update", "namespace", r.Namespace, "name", r.Name)

	// Converting to Database type
	oldDatabase := old.(*Database)

	if r.Spec.Name != oldDatabase.Spec.Name {
		return nil, &validationError{"Name not allowed to be changed"}
	}

	return r.ValidateCharsetCollationCombo()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Database) ValidateDelete() (admission.Warnings, error) {
	databaseLog.Info("validate delete", "namespace", r.Namespace, "name", r.Name)
	// Not implemented
	return nil, nil
}

func (r *Database) ValidateCharsetCollationCombo() (admission.Warnings, error) {

	// If there is no admin connection, we can skip this validation for now.
	if r.Spec.AdminConnection.Name == "" {
		return nil, nil
	}

	databaseLog.Info("validate charset collation combo", "namespace", r.Namespace, "name", r.Name)

	// Getting admin connection
	adminNamespace := r.Namespace
	if r.Spec.AdminConnection.Namespace != "" {
		adminNamespace = r.Spec.AdminConnection.Namespace
	}
	adminConnectionNamespacedName := types.NamespacedName{
		Namespace: adminNamespace,
		Name:      r.Spec.AdminConnection.Name,
	}

	adminConnection := &AdminConnection{}
	err := k8sClient.Get(context.TODO(), adminConnectionNamespacedName, adminConnection)
	if err != nil {
		return nil, err
	}
	logger := databaseLog.WithValues("AdminConnection", types.NamespacedName{Name: adminConnection.Name, Namespace: adminConnection.Namespace})

	// Getting charset and collation
	charset := r.Spec.CharacterSet
	if charset == "" {
		charset = adminConnection.Status.CharacterSet
	}
	collation := r.Spec.Collate
	if collation == "" {
		collation = adminConnection.Status.Collation
	}
	logger.Info("Validating charset and collation", "charset", charset, "collation", collation, "adminConnection", adminConnection.Status)

	for _, charsetCollationCombo := range adminConnection.Status.AvailableCharsets {
		logger.Info("Checking", "current charset", charsetCollationCombo.Name, "charset", charset)
		if charsetCollationCombo.Name == charset {
			for _, collationEntry := range charsetCollationCombo.Collations {
				if collationEntry.Name == collation {
					if !collationEntry.Default {
						logger.Info("Collation not default for charset")
						warnings := admission.Warnings{
							"Collation not the default for this charset",
						}
						return warnings, nil
					} else {
						logger.Info("Charset and collation combination valid for this server")
						return nil, nil
					}
				}
			}
			return nil, &validationError{"Charset and collation combination not valid for this server"}
		}
	}
	return nil, &validationError{"Charset not valid for this server"}
}
