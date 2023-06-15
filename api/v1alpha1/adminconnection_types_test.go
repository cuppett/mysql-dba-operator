package v1alpha1

import (
	"github.com/cuppett/mysql-dba-operator/orm"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("AdminConnection_Types", func() {

	Describe("AllowedNamespace", func() {
		DescribeTable("Namespace rules",
			func(namespaceList []string, name string, good bool) {
				adminConnection.Spec.AllowedNamespaces = namespaceList
				condition := adminConnection.AllowedNamespace(name)

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

	Describe("DatabaseMine", func() {
		var gormDB *gorm.DB
		var err error
		var managedDatabase *orm.ManagedDatabase
		var database *Database
		cache := make(map[types.UID]*orm.ConnectionDefinition)
		var saveGormDb, createDb bool

		BeforeEach(func() {
			gormDB, err = adminConnection.GetDatabaseConnection(ctx, k8sClient, cache)
			Expect(err).NotTo(HaveOccurred())
			Expect(gormDB).NotTo(BeNil())

			// Wipe the database table here.
			gormDB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&orm.ManagedDatabase{})

			newUID := uuid.New()
			database = &Database{
				ObjectMeta: metav1.ObjectMeta{
					UID:       types.UID(newUID.String()),
					Namespace: "default",
					Name:      "test",
				},
				Spec: DatabaseSpec{
					Name: "test",
				},
			}

			managedDatabase = &orm.ManagedDatabase{
				Uuid:         newUID.String(),
				Namespace:    database.Namespace,
				Name:         database.Name,
				DatabaseName: database.Spec.Name,
			}
			saveGormDb = true
			createDb = true
		})

		JustBeforeEach(func() {
			if saveGormDb {
				tx := gormDB.Create(managedDatabase)
				Expect(tx.Error).To(BeNil())
			}

			if createDb {
				createQuery := "CREATE DATABASE `" + database.Spec.Name + "`"
				tx := gormDB.Exec(createQuery)
				Expect(tx.Error).To(BeNil())
			}
		})

		JustAfterEach(func() {
			if createDb {
				dropQuery := "DROP DATABASE IF EXISTS `" + database.Spec.Name + "`"
				tx := gormDB.Exec(dropQuery)
				Expect(tx.Error).To(BeNil())
			}
		})

		Context("Database does exist in the table, and in the server, and it is a match.", func() {
			It("returns true for mine.", func() {
				isDatabaseMine := adminConnection.DatabaseMine(gormDB, database)
				Expect(isDatabaseMine).To(BeTrue())
			})
		})

		Context("Database doesn't exist in the table or server", func() {
			BeforeEach(func() {
				createDb = false
				saveGormDb = false
			})
			It("returns true for mine.", func() {
				isDatabaseMine := adminConnection.DatabaseMine(gormDB, database)
				Expect(isDatabaseMine).To(BeTrue())
			})
		})

		Context("Database does exist in the table, but not in the server", func() {
			BeforeEach(func() {
				createDb = false
			})
			It("returns true for mine.", func() {
				isDatabaseMine := adminConnection.DatabaseMine(gormDB, database)
				Expect(isDatabaseMine).To(BeTrue())
			})
		})

		Context("Database does exist in the table, and in the server, but not my UID", func() {
			BeforeEach(func() {
				managedDatabase.Uuid = uuid.New().String()
			})
			It("returns false for mine.", func() {
				isDatabaseMine := adminConnection.DatabaseMine(gormDB, database)
				Expect(isDatabaseMine).To(BeFalse())
			})
		})

		Context("Database does exist in the table, and in the server, but not my k8s name", func() {
			BeforeEach(func() {
				managedDatabase.Name = "wrongk8sname"
			})
			It("returns false for mine.", func() {
				isDatabaseMine := adminConnection.DatabaseMine(gormDB, database)
				Expect(isDatabaseMine).To(BeFalse())
			})
		})

		Context("Database does exist in the table, and in the server, but not my k8s namespace", func() {
			BeforeEach(func() {
				managedDatabase.Namespace = "wrongnamespace"
			})
			It("returns false for mine.", func() {
				isDatabaseMine := adminConnection.DatabaseMine(gormDB, database)
				Expect(isDatabaseMine).To(BeFalse())
			})
		})

		Context("Database does exist in the table, and in the server, but not my database name", func() {
			BeforeEach(func() {
				managedDatabase.DatabaseName = "wrongdatabasename"
			})
			It("returns false for mine.", func() {
				isDatabaseMine := adminConnection.DatabaseMine(gormDB, database)
				Expect(isDatabaseMine).To(BeFalse())
			})
		})
	})

	Describe("UserMine", func() {
		var gormDB *gorm.DB
		var err error
		var managedUser *orm.ManagedUser
		var user *DatabaseUser
		cache := make(map[types.UID]*orm.ConnectionDefinition)
		var saveGormDb, createUser bool

		BeforeEach(func() {
			gormDB, err = adminConnection.GetDatabaseConnection(ctx, k8sClient, cache)
			Expect(err).NotTo(HaveOccurred())
			Expect(gormDB).NotTo(BeNil())

			// Wipe the user table here.
			gormDB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&orm.ManagedUser{})

			newUID := uuid.New()
			user = &DatabaseUser{
				ObjectMeta: metav1.ObjectMeta{
					UID:       types.UID(newUID.String()),
					Namespace: "default",
					Name:      "test",
				},
				Spec: DatabaseUserSpec{
					Username: "test",
				},
			}

			managedUser = &orm.ManagedUser{
				Uuid:      newUID.String(),
				Namespace: user.Namespace,
				Name:      user.Name,
				Username:  user.Spec.Username,
			}
			saveGormDb = true
			createUser = true
		})

		JustBeforeEach(func() {
			if saveGormDb {
				tx := gormDB.Create(managedUser)
				Expect(tx.Error).To(BeNil())
			}

			if createUser {
				createQuery := "CREATE USER '" + Escape(user.Spec.Username) + "'"
				tx := gormDB.Exec(createQuery)
				Expect(tx.Error).To(BeNil())
			}
		})

		JustAfterEach(func() {
			if createUser {
				dropQuery := "DROP USER IF EXISTS `" + Escape(user.Spec.Username) + "`"
				tx := gormDB.Exec(dropQuery)
				Expect(tx.Error).To(BeNil())
			}
		})

		Context("User does exist in the table, and in the server, and it is a match.", func() {
			It("returns true for mine.", func() {
				isUserMine := adminConnection.UserMine(gormDB, user)
				Expect(isUserMine).To(BeTrue())
			})
		})

		Context("User doesn't exist in the table or server", func() {
			BeforeEach(func() {
				createUser = false
				saveGormDb = false
			})
			It("returns true for mine.", func() {
				isUserMine := adminConnection.UserMine(gormDB, user)
				Expect(isUserMine).To(BeTrue())
			})
		})

		Context("User does exist in the table, but not in the server", func() {
			BeforeEach(func() {
				createUser = false
			})
			It("returns true for mine.", func() {
				isUserMine := adminConnection.UserMine(gormDB, user)
				Expect(isUserMine).To(BeTrue())
			})
		})

		Context("User does exist in the table, and in the server, but not my UID", func() {
			BeforeEach(func() {
				managedUser.Uuid = uuid.New().String()
			})
			It("returns false for mine.", func() {
				isUserMine := adminConnection.UserMine(gormDB, user)
				Expect(isUserMine).To(BeFalse())
			})
		})

		Context("User does exist in the table, and in the server, but not my k8s name", func() {
			BeforeEach(func() {
				managedUser.Name = "wrongk8sname"
			})
			It("returns false for mine.", func() {
				isUserMine := adminConnection.UserMine(gormDB, user)
				Expect(isUserMine).To(BeFalse())
			})
		})

		Context("User does exist in the table, and in the server, but not my k8s namespace", func() {
			BeforeEach(func() {
				managedUser.Namespace = "wrongnamespace"
			})
			It("returns false for mine.", func() {
				isUserMine := adminConnection.UserMine(gormDB, user)
				Expect(isUserMine).To(BeFalse())
			})
		})

		Context("User does exist in the table, and in the server, but not my user name", func() {
			BeforeEach(func() {
				managedUser.Username = "wrongusername"
			})
			It("returns false for mine.", func() {
				isUserMine := adminConnection.UserMine(gormDB, user)
				Expect(isUserMine).To(BeFalse())
			})
		})
	})
})
