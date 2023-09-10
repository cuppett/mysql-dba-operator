package controllers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"time"

	. "github.com/cuppett/mysql-dba-operator/api/v1alpha1"
)

var _ = Describe("DatabaseUser", func() {

	Describe("Creation Scenario", func() {

		var databaseUser *DatabaseUser

		It("Happy Path", func(ctx SpecContext) {

			databaseUser = &DatabaseUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: ServerAdminConnection.Namespace,
				},
				Spec: DatabaseUserSpec{
					AdminConnection: AdminConnectionRef{
						Name: ServerAdminConnection.Name,
					},
					Username: "test-user",
				},
			}

			err := k8sClient.Create(ctx, databaseUser)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() string {
				userObject := &DatabaseUser{}
				databaseNamespacedName := types.NamespacedName{
					Namespace: databaseUser.Namespace,
					Name:      databaseUser.Name,
				}
				err := k8sClient.Get(ctx, databaseNamespacedName, userObject)
				Expect(err).ToNot(HaveOccurred())
				return userObject.Status.Message
			}).WithContext(ctx).Should(Equal("Created user"))
		}, NodeTimeout(time.Second*30))
	})

})
