package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Database User", func() {
	Describe("Testing Database and Grant Lengths", func() {
		var database_user *DatabaseUser

		BeforeEach(func() {
			database_user = &DatabaseUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: DatabaseUserSpec{
					Username: "test",
				},
			}
		})

		It("has equal empty lists", func() {
			database_user.Spec.DatabaseList = []DatabasePermission{}
			database_user.Status.DatabaseList = []DatabasePermission{}
			isEqual := database_user.PermissionListEqual()
			Expect(isEqual).To(BeTrue())
		})

		It("has equal lists", func() {
			database_user.Spec.DatabaseList = []DatabasePermission{
				{
					Name:   "test1",
					Grants: []string{"SELECT", "INSERT"},
				}, {
					Name:   "test2",
					Grants: []string{"SELECT", "INSERT"},
				}}

			database_user.Status.Grants = []string{
				"GRANT SELECT, INSERT ON test1",
				"GRANT SELECT, INSERT ON test2",
			}
			database_user.Status.DatabaseList = []DatabasePermission{
				{
					Name:   "test1",
					Grants: []string{"SELECT", "INSERT"},
				}, {
					Name:   "test2",
					Grants: []string{"SELECT", "INSERT"},
				}}

			isEqual := database_user.PermissionListEqual()
			Expect(isEqual).To(BeTrue())
		})

		It("has missing grants", func() {
			database_user.Spec.DatabaseList = []DatabasePermission{
				{
					Name:   "test1",
					Grants: []string{"SELECT", "INSERT"},
				}, {
					Name:   "test2",
					Grants: []string{"SELECT", "INSERT"},
				}}

			database_user.Status.Grants = []string{}
			database_user.Status.DatabaseList = []DatabasePermission{
				{
					Name:   "test1",
					Grants: []string{"SELECT", "INSERT"},
				}, {
					Name:   "test2",
					Grants: []string{"SELECT", "INSERT"},
				}}

			isEqual := database_user.PermissionListEqual()
			Expect(isEqual).To(BeFalse())
		})

		It("has missing permissions", func() {
			database_user.Spec.DatabaseList = []DatabasePermission{
				{
					Name:   "test1",
					Grants: []string{"SELECT", "INSERT"},
				}, {
					Name:   "test2",
					Grants: []string{"SELECT", "INSERT"},
				}}

			database_user.Status.Grants = []string{
				"GRANT SELECT ON test1",
				"GRANT SELECT ON test2",
			}
			database_user.Status.DatabaseList = []DatabasePermission{
				{
					Name:   "test1",
					Grants: []string{"SELECT"},
				}, {
					Name:   "test2",
					Grants: []string{"SELECT"},
				}}

			isEqual := database_user.PermissionListEqual()
			Expect(isEqual).To(BeFalse())
		})

		It("has missing grant", func() {
			database_user.Spec.DatabaseList = []DatabasePermission{
				{
					Name:   "test1",
					Grants: []string{"SELECT", "INSERT"},
				}, {
					Name:   "test2",
					Grants: []string{"SELECT", "INSERT"},
				}}

			database_user.Status.Grants = []string{
				"GRANT SELECT ON test1",
			}
			database_user.Status.DatabaseList = []DatabasePermission{
				{
					Name:   "test1",
					Grants: []string{"SELECT"},
				}}

			isEqual := database_user.PermissionListEqual()
			Expect(isEqual).To(BeFalse())
		})

		AfterEach(func() {
			database_user = nil
		})
	})
})
