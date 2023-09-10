package controllers

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

var _ = Describe("ServerAdminConnection", func() {

	Describe("Testing ServerAdminConnection for database and user happy paths", func() {

		It("should have good status", func(ctx SpecContext) {
			Eventually(func() string {
				adminConnectionNamespacedName := types.NamespacedName{
					Namespace: ServerAdminConnection.Namespace,
					Name:      ServerAdminConnection.Name,
				}
				err := k8sClient.Get(ctx, adminConnectionNamespacedName, ServerAdminConnection)
				Expect(err).ToNot(HaveOccurred())
				return ServerAdminConnection.Status.Message
			}).WithContext(ctx).Should(Equal("Successfully pinged database"))
		}, NodeTimeout(time.Second*30))

		It("should have character set and collation data", func(ctx SpecContext) {
			Expect(ServerAdminConnection.Status.CharacterSet).ShouldNot(BeEmpty())
			Expect(ServerAdminConnection.Status.Collation).ShouldNot(BeEmpty())
			Expect(ServerAdminConnection.Status.AvailableCharsets).ShouldNot(BeEmpty())
		})

	})
})
