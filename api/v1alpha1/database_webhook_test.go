package v1alpha1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
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

	var _ = Describe("Charset and Collation", func() {
		var database *Database

		DescribeTable("Charset and Collation rules on create",
			func(name string, charset string, collation string, expectError bool, expectedError string, expectedWarnings admission.Warnings) {
				database = &Database{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "comborules",
						Namespace: "default",
					},
					Spec: DatabaseSpec{
						Name: name,
						AdminConnection: AdminConnectionRef{
							Name: ServerAdminConnection.Name,
						},
						CharacterSet: charset,
						Collate:      collation,
					},
				}
				err := k8sClient.Create(ctx, database)
				if expectError {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(expectedError))
				} else {
					Expect(err).NotTo(HaveOccurred())
				}

				err = k8sClient.Delete(ctx, database)
			},
			Entry("Defaults", "default-charset-collate", "", "", false, nil, nil),
			Entry("Good combination", "good-charset-collate", "utf8mb4", "utf8mb4_general_ci", false, nil, nil),
			Entry("Bad charset", "bad-charset", "foo", "", true, "Charset not valid for this server", nil),
			Entry("Bad collation for charset", "bad-collate", "utf8mb4", "foo", true, "Charset and collation combination not valid for this server", nil),
			Entry("Non default collation for charset", "non-default-collate", "utf8mb4", "utf8mb4_unicode_ci", false, nil, admission.Warnings{"Collation not the default for this charset"}),
		)
	})
})
