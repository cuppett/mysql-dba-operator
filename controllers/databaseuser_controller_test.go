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

		It("User Removed Path", func(ctx SpecContext) {

			// Pre-reqs
			username := "missing-user"
			databaseNamespacedName := types.NamespacedName{
				Name:      username,
				Namespace: ServerAdminConnection.Namespace,
			}

			// Getting database connection
			cache := make(map[types.UID]*orm.ConnectionDefinition)
			gormDB, err := ServerAdminConnection.GetDatabaseConnection(ctx, k8sClient, cache)
			Expect(err).ToNot(HaveOccurred())

			databaseUser := &DatabaseUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      username,
					Namespace: ServerAdminConnection.Namespace,
				},
				Spec: DatabaseUserSpec{
					AdminConnection: AdminConnectionRef{
						Name: ServerAdminConnection.Name,
					},
					Username: username,
				},
			}

			err = k8sClient.Create(ctx, databaseUser)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() string {
				userObject := &DatabaseUser{}
				err := k8sClient.Get(ctx, databaseNamespacedName, userObject)
				Expect(err).ToNot(HaveOccurred())
				return userObject.Status.Message
			}).WithContext(ctx).Should(Equal("Created user"))

			// Check the actual database if the user exists again
			var count int64
			gormDB.Model(&orm.MySqlUser{}).Where("User = ?", username).Count(&count)
			Expect(count).To(Equal(int64(1)))

			// Remove database user underneath
			dropQuery := "DROP USER '" + Escape(username) + "'"
			tx := gormDB.Exec(dropQuery)
			Expect(tx.Error).To(BeNil())

			// Tickle so it gets put back via reconcile.
			// Continue to do this until we stop getting 409.
			Eventually(func() error {
				// Fetch a fresh DatabaseUser
				err = k8sClient.Get(ctx, databaseNamespacedName, databaseUser)
				Expect(err).ToNot(HaveOccurred())
				databaseUser.Spec.TlsOptions.Required = true
				err = k8sClient.Update(ctx, databaseUser)
				return err
			}).WithContext(ctx).Should(BeNil())

			Eventually(func() int64 {
				// Check the actual database if the user exists again
				gormDB.Model(&orm.MySqlUser{}).Where("User = ?", username).Count(&count)
				return count
			}).WithContext(ctx).Should(Equal(int64(1)))

			err = k8sClient.Get(ctx, databaseNamespacedName, databaseUser)
			Expect(err).ToNot(HaveOccurred())
			Expect(databaseUser.Status.Message).To(Equal("Created user"))

		}, NodeTimeout(time.Second*30))

		It("User Removed With Database", func(ctx SpecContext) {

			// Pre-reqs
			username := "missing-user-with-db"
			databaseNamespacedName := types.NamespacedName{
				Name:      username,
				Namespace: ServerAdminConnection.Namespace,
			}

			// Getting database connection
			cache := make(map[types.UID]*orm.ConnectionDefinition)
			gormDB, err := ServerAdminConnection.GetDatabaseConnection(ctx, k8sClient, cache)
			Expect(err).ToNot(HaveOccurred())

			// Creating database to use.
			database := &Database{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-database-grant",
					Namespace: ServerAdminConnection.Namespace,
				},
				Spec: DatabaseSpec{
					AdminConnection: AdminConnectionRef{
						Name: ServerAdminConnection.Name,
					},
					Name: "test-database-grant",
				},
			}
			err = k8sClient.Create(ctx, database)
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

			databaseUser := &DatabaseUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      username,
					Namespace: ServerAdminConnection.Namespace,
				},
				Spec: DatabaseUserSpec{
					AdminConnection: AdminConnectionRef{
						Name: ServerAdminConnection.Name,
					},
					Username: username,
					DatabaseList: []DatabasePermission{
						{
							Name: "test-database-grant",
							Grants: []string{
								"ALL",
							},
						},
					},
				},
			}

			err = k8sClient.Create(ctx, databaseUser)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() string {
				userObject := &DatabaseUser{}
				err := k8sClient.Get(ctx, databaseNamespacedName, userObject)
				Expect(err).ToNot(HaveOccurred())
				return userObject.Status.Message
			}).WithContext(ctx).Should(Equal("Created user"))

			// Check the actual database if the user exists again
			var count int64
			gormDB.Model(&orm.MySqlUser{}).Where("User = ?", username).Count(&count)
			Expect(count).To(Equal(int64(1)))

			// Check the actual database if the user grant exists
			gormDB.Model(&orm.MySqlDb{}).Where("User = ?", username).Count(&count)
			Expect(count).To(Equal(int64(1)))

			// Remove database user underneath
			dropQuery := "DROP USER '" + Escape(username) + "'"
			tx := gormDB.Exec(dropQuery)
			Expect(tx.Error).To(BeNil())

			// Tickle so it gets put back via reconcile.
			// Continue to do this until we stop getting 409.
			Eventually(func() error {
				// Fetch a fresh DatabaseUser
				err = k8sClient.Get(ctx, databaseNamespacedName, databaseUser)
				Expect(err).ToNot(HaveOccurred())
				databaseUser.Spec.TlsOptions.Required = true
				err = k8sClient.Update(ctx, databaseUser)
				return err
			}).WithContext(ctx).Should(BeNil())

			// Check the actual database if the user exists again
			Eventually(func() int64 {
				gormDB.Model(&orm.MySqlUser{}).Where("User = ?", username).Count(&count)
				return count
			}).WithContext(ctx).Should(Equal(int64(1)))

			err = k8sClient.Get(ctx, databaseNamespacedName, databaseUser)
			Expect(err).ToNot(HaveOccurred())
			Expect(databaseUser.Status.Message).To(Equal("Created user"))

			// Check the actual database if the user grant exists again
			Eventually(func() int64 {
				gormDB.Model(&orm.MySqlDb{}).Where("User = ?", username).Count(&count)
				return count
			}).WithContext(ctx).Should(Equal(int64(1)))

			Eventually(func() int {
				err = k8sClient.Get(ctx, databaseNamespacedName, databaseUser)
				Expect(err).ToNot(HaveOccurred())
				return len(databaseUser.Status.Grants)
			}).WithContext(ctx).Should(Equal(1))

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

			// Continue to do this until we stop getting 409.
			Eventually(func() error {
				// Fetch a fresh DatabaseUser
				err = k8sClient.Get(ctx, databaseNamespacedName, databaseUser)
				Expect(err).ToNot(HaveOccurred())
				databaseUser.Spec.Username = "test-user-renamed"
				err = k8sClient.Update(ctx, databaseUser)
				return err
			}).WithContext(ctx).Should(BeNil())

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

			// Continue to do this until we stop getting 409.
			Eventually(func() error {
				// Fetch a fresh DatabaseUser
				err = k8sClient.Get(ctx, databaseNamespacedName, databaseUser)
				Expect(err).ToNot(HaveOccurred())
				databaseUser.Spec.Username = "existing_user"
				err = k8sClient.Update(ctx, databaseUser)
				return err
			}).WithContext(ctx).Should(BeNil())

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

			// Continue to do this until we stop getting 409.
			Eventually(func() error {
				// Fetch a fresh DatabaseUser
				err = k8sClient.Get(ctx, databaseNamespacedName, databaseUser)
				Expect(err).ToNot(HaveOccurred())
				databaseUser.Spec.Username = "switch-good-user"
				err = k8sClient.Update(ctx, databaseUser)
				return err
			}).WithContext(ctx).Should(BeNil())

			Eventually(func() string {
				userObject := &DatabaseUser{}
				err := k8sClient.Get(ctx, databaseNamespacedName, userObject)
				Expect(err).NotTo(HaveOccurred())
				return userObject.Status.Message
			}).WithContext(ctx).Should(Equal("Created user"))

			// Fetching again and checking the actual username.
			err = k8sClient.Get(ctx, databaseNamespacedName, databaseUser)
			Expect(err).ToNot(HaveOccurred())
			Expect(databaseUser.Status.Username).To(Equal("switch-good-user"))

		}, NodeTimeout(time.Second*30))
	})
})
