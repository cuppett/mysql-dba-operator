package v1alpha1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Admin Connection", func() {

	var adminconnection *AdminConnection

	BeforeEach(func() {
		adminconnection = &AdminConnection{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
			},
			Spec: AdminConnectionSpec{},
		}
	})

	Describe("Testing AdminConnection for database and user happy paths", func() {
		DescribeTable("Namespace rules",
			func(namespace_list []string, name string, good bool) {
				adminconnection.Spec.AllowedNamespaces = namespace_list
				condition := adminconnection.AllowedNamespace(name)

				if good {
					Expect(condition).To(BeTrue())
				} else {
					Expect(condition).To(BeFalse())
				}
			},
			Entry("Allow itself when no list", []string{}, "default", true),
			Entry("Disallow explicit thing", []string{}, "kube-system", false),
			Entry("Allow itself when any list", []string{"test1", "test2"}, "default", true),
			Entry("Disallow explicit thing not in list", []string{"test1"}, "kube-system", false),
			Entry("Allow explicit thing in list", []string{"test1"}, "test1", true),
			Entry("Allow all tests", []string{"test*"}, "test1", true),
			Entry("Allow all tests (2)", []string{"test*"}, "test-fun", true),
			Entry("Allow self with prefix", []string{"test*"}, "default", true),
			Entry("Disallow non-match with prefix", []string{"test*"}, "kube-system", false),
		)
	})

	AfterEach(func() {
		adminconnection = nil
	})

})
