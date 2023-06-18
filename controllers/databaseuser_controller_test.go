package controllers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"time"

	. "github.com/cuppett/mysql-dba-operator/api/v1alpha1"
)

var _ = Describe("DatabaseUser", func() {

	It("Has good connection", func(ctx SpecContext) {
		serverAdminConnection := &AdminConnection{}
		Eventually(func() string {
			adminConnectionNamespacedName := types.NamespacedName{
				Namespace: adminConnection.Namespace,
				Name:      adminConnection.Name,
			}
			err := k8sClient.Get(ctx, adminConnectionNamespacedName, serverAdminConnection)
			if err != nil {
				if errors.IsNotFound(err) {
					return "AdminConnection resource not found. Object must be deleted"
				}
				// Error reading the object - requeue the request.
				return err.Error()
			}
			return serverAdminConnection.Status.Message
		}).WithContext(ctx).Should(Equal("Successfully pinged database"))
		adminConnection = serverAdminConnection
	}, NodeTimeout(time.Second*30))

	Describe("Creation Scenario", func() {

		var databaseUser *DatabaseUser

		It("Happy Path", func(ctx SpecContext) {

			databaseUser = &DatabaseUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: adminConnection.Namespace,
				},
				Spec: DatabaseUserSpec{
					AdminConnection: AdminConnectionRef{
						Name: adminConnection.Name,
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
