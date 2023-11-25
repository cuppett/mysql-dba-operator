package controllers

import (
	"github.com/cuppett/mysql-dba-operator/orm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"time"

	. "github.com/cuppett/mysql-dba-operator/api/v1alpha1"
)

var _ = Describe("DatabaseUser", func() {

	Describe("Creation Scenario", func() {

		It("Happy Path", func(ctx SpecContext) {

			databaseUser := &DatabaseUser{
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

	Describe("Rename Scenario", func() {

		It("Should have an existing user to rename", func(ctx SpecContext) {
			// Pre-reqs
			cache := make(map[types.UID]*orm.ConnectionDefinition)
			gormDB, err := ServerAdminConnection.GetDatabaseConnection(ctx, k8sClient, cache)
			Expect(err).ToNot(HaveOccurred())

			// Create database user to be renamed
			createQuery := "CREATE USER '" + Escape("existing_user") + "'"
			tx := gormDB.Exec(createQuery)
			Expect(tx.Error).To(BeNil())
		})

		It("Happy Path", func(ctx SpecContext) {

			databaseNamespacedName := types.NamespacedName{
				Name:      "test-user-name",
				Namespace: ServerAdminConnection.Namespace,
			}

			databaseUser := &DatabaseUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user-name",
					Namespace: ServerAdminConnection.Namespace,
				},
				Spec: DatabaseUserSpec{
					AdminConnection: AdminConnectionRef{
						Name: ServerAdminConnection.Name,
					},
					Username: "test-user-name",
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

			// Fetch a fresh DatabaseUser
			databaseUser = &DatabaseUser{}
			err = k8sClient.Get(ctx, databaseNamespacedName, databaseUser)
			Expect(err).ToNot(HaveOccurred())

			databaseUser.Spec.Username = "test-user-renamed"
			err = k8sClient.Update(ctx, databaseUser)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() string {
				userObject := &DatabaseUser{}
				err := k8sClient.Get(ctx, databaseNamespacedName, userObject)
				Expect(err).ToNot(HaveOccurred())
				return userObject.Status.Message
			}).WithContext(ctx).Should(Equal("User renamed"))

			// Fetching again and checking the actual username.
			databaseUser = &DatabaseUser{}
			err = k8sClient.Get(ctx, databaseNamespacedName, databaseUser)
			Expect(err).ToNot(HaveOccurred())
			Expect(databaseUser.Status.Username).To(Equal("test-user-renamed"))

		}, NodeTimeout(time.Second*30))

		It("Fail Rename To Existing User", func(ctx SpecContext) {
			databaseNamespacedName := types.NamespacedName{
				Name:      "orig-user-name",
				Namespace: ServerAdminConnection.Namespace,
			}

			databaseUser := &DatabaseUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "orig-user-name",
					Namespace: ServerAdminConnection.Namespace,
				},
				Spec: DatabaseUserSpec{
					AdminConnection: AdminConnectionRef{
						Name: ServerAdminConnection.Name,
					},
					Username: "orig-user-name",
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

			// Fetch a fresh DatabaseUser
			databaseUser = &DatabaseUser{}
			err = k8sClient.Get(ctx, databaseNamespacedName, databaseUser)
			Expect(err).ToNot(HaveOccurred())

			// Update the username field
			databaseUser.Spec.Username = "existing_user"
			err = k8sClient.Update(ctx, databaseUser)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() string {
				userObject := &DatabaseUser{}
				err := k8sClient.Get(ctx, databaseNamespacedName, userObject)
				Expect(err).NotTo(HaveOccurred())
				return userObject.Status.Message
			}).WithContext(ctx).Should(Equal("Ownership failed. Unable to update user."))

			// Fetching again and checking the actual username.
			databaseUser = &DatabaseUser{}
			err = k8sClient.Get(ctx, databaseNamespacedName, databaseUser)
			Expect(err).ToNot(HaveOccurred())
			Expect(databaseUser.Status.Username).To(Equal("orig-user-name"))

		}, NodeTimeout(time.Second*30))

		It("Allow Rename To Non-Existent User", func(ctx SpecContext) {

			databaseNamespacedName := types.NamespacedName{
				Name:      "switch-good-user",
				Namespace: ServerAdminConnection.Namespace,
			}

			databaseUser := &DatabaseUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "switch-good-user",
					Namespace: ServerAdminConnection.Namespace,
				},
				Spec: DatabaseUserSpec{
					AdminConnection: AdminConnectionRef{
						Name: ServerAdminConnection.Name,
					},
					Username: "existing_user",
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
			}).WithContext(ctx).Should(Equal("Invalid username specified."))

			// Fetch a fresh DatabaseUser
			databaseUser = &DatabaseUser{}
			err = k8sClient.Get(ctx, databaseNamespacedName, databaseUser)
			Expect(err).ToNot(HaveOccurred())

			// Update the username field
			databaseUser.Spec.Username = "switch-good-user"
			err = k8sClient.Update(ctx, databaseUser)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() string {
				userObject := &DatabaseUser{}
				err := k8sClient.Get(ctx, databaseNamespacedName, userObject)
				Expect(err).NotTo(HaveOccurred())
				return userObject.Status.Message
			}).WithContext(ctx).Should(Equal("Created user"))

			// Fetching again and checking the actual username.
			databaseUser = &DatabaseUser{}
			err = k8sClient.Get(ctx, databaseNamespacedName, databaseUser)
			Expect(err).ToNot(HaveOccurred())
			Expect(databaseUser.Status.Username).To(Equal("switch-good-user"))

		}, NodeTimeout(time.Second*30))
	})
})
