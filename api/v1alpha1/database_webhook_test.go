package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Database Webhook", func() {
	var database *Database

	DescribeTable("Name rules",
		func(name string) {
			database = &Database{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "namerules",
					Namespace: "default",
				},
				Spec: DatabaseSpec{
					Name: name,
				},
			}
			err := k8sClient.Create(ctx, database)
			Expect(err).To(HaveOccurred())
		},
		Entry("should not allow names with slash", `test\test`),
		Entry("should not allow names with question mark", "test?test"),
		Entry("should not allow names with asterisks", "test*test"),
		Entry("should not allow long names", "test0123893498391389193874adkljflkasjdflkajdf197194797149714897349734979test"),
	)

	Describe("Changing names", func() {
		BeforeEach(func() {
			database = &Database{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: DatabaseSpec{
					Name: "test",
				},
			}
		})

		It("should not allow changing names", func() {
			err := k8sClient.Create(ctx, database)
			Expect(err).NotTo(HaveOccurred())

			database.Spec.Name = "test2"
			err = k8sClient.Update(ctx, database)
			Expect(err).To(HaveOccurred())
		})

		It("should allow changing something else", func() {
			err := k8sClient.Create(ctx, database)
			Expect(err).NotTo(HaveOccurred())

			database.Spec.CharacterSet = "utf8mb4"
			err = k8sClient.Update(ctx, database)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err := k8sClient.Delete(ctx, database)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
