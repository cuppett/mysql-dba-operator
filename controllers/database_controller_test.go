package controllers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"time"

	. "github.com/cuppett/mysql-dba-operator/api/v1alpha1"
)

var _ = Describe("Database", func() {

	Describe("Creation Scenario", func() {

		var database *Database

		It("Happy Path", func(ctx SpecContext) {

			database = &Database{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-database",
					Namespace: ServerAdminConnection.Namespace,
				},
				Spec: DatabaseSpec{
					AdminConnection: AdminConnectionRef{
						Name: ServerAdminConnection.Name,
					},
					Name: "test-database",
				},
			}

			err := k8sClient.Create(ctx, database)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() string {
				databaseObject := &Database{}
				databaseNamespacedName := types.NamespacedName{
					Namespace: database.Namespace,
					Name:      database.Name,
				}
				err := k8sClient.Get(ctx, databaseNamespacedName, databaseObject)
				Expect(err).ToNot(HaveOccurred())
				return databaseObject.Status.Message
			}).WithContext(ctx).Should(Equal("Database in sync"))
		}, NodeTimeout(time.Second*30))
	})

})
