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
	"k8s.io/apimachinery/pkg/runtime"
	"regexp"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	nameRegEx = regexp.MustCompile(`^[^\\/?%*:|"<>.]{1,64}$`)
)

// log is for logging in this package.
var databaselog = logf.Log.WithName("database-resource")

func (r *Database) SetupWebhookWithManager(mgr ctrl.Manager) error {
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
	databaselog.Info("validate create", "name", r.Name)

	// See also: https://stackoverflow.com/questions/9537771/mysql-database-name-restrictions
	if !nameRegEx.MatchString(r.Spec.Name) {
		return nil, &validationError{"Invalid database name."}
	}

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Database) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	databaselog.Info("validate update", "name", r.Name)

	// Converting to Database type
	oldDatabase := old.(*Database)

	if r.Spec.Name != oldDatabase.Spec.Name {
		return nil, &validationError{"Name not allowed to be changed"}
	}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Database) ValidateDelete() (admission.Warnings, error) {
	databaselog.Info("validate delete", "name", r.Name)
	// Not implemented
	return nil, nil
}
