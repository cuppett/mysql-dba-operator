package controllers

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"time"

	. "github.com/cuppett/mysql-dba-operator/api/v1alpha1"
)

var _ = Describe("AdminConnection", func() {

	Describe("Testing AdminConnection for database and user happy paths", func() {

		serverAdminConnection := &AdminConnection{}

		It("should have good status", func(ctx SpecContext) {
			Eventually(func() string {
				adminConnectionNamespacedName := types.NamespacedName{
					Namespace: adminConnection.Namespace,
					Name:      adminConnection.Name,
				}
				err := k8sClient.Get(ctx, adminConnectionNamespacedName, serverAdminConnection)
				Expect(err).ToNot(HaveOccurred())
				return serverAdminConnection.Status.Message
			}).WithContext(ctx).Should(Equal("Successfully pinged database"))
		}, NodeTimeout(time.Second*30))

		It("should have character set and collation data", func(ctx SpecContext) {
			Expect(serverAdminConnection.Status.CharacterSet).ShouldNot(BeEmpty())
			Expect(serverAdminConnection.Status.Collation).ShouldNot(BeEmpty())
			Expect(serverAdminConnection.Status.AvailableCharsets).ShouldNot(BeEmpty())
		})

	})
})
