package controllers

import (
	"github.com/docker/docker/api/types/container"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	. "github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/wait"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"time"

	. "github.com/cuppett/mysql-dba-operator/api/v1alpha1"
)

var _ = Describe("Admin Connection", func() {

	var adminConnection *AdminConnection
	var mysqlContainer *MySQLContainer
	var err error

	BeforeEach(func() {

		image, ok := os.LookupEnv("MYSQL_IMAGE")
		if !ok {
			image = "ghcr.io/cuppett/mariadb:10.11"
		}

		mysqlContainer, err = RunContainer(ctx, testcontainers.WithImage(image),
			WithUsername("root"), WithPassword(""),
			testcontainers.WithConfigModifier(func(config *container.Config) {
				config.Env = []string{"MYSQL_ALLOW_EMPTY_PASSWORD=true"}
			}),
			testcontainers.WithWaitStrategyAndDeadline(time.Second*60, wait.ForListeningPort("3306/tcp")),
		)
		Expect(err).NotTo(HaveOccurred())

		hostname, err := mysqlContainer.Host(ctx)
		port, err := mysqlContainer.MappedPort(ctx, "3306")

		Expect(err).NotTo(HaveOccurred())

		adminConnection = &AdminConnection{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
			},
			Spec: AdminConnectionSpec{
				Host: hostname,
				Port: int32(port.Int()),
			},
		}
		err = k8sClient.Create(ctx, adminConnection)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Testing AdminConnection for database and user happy paths", func() {

		It("should have good status", func(ctx SpecContext) {
			Eventually(func() string {
				serverAdminConnection := &AdminConnection{}
				//time.Sleep(time.Second * 90)
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
		}, NodeTimeout(time.Second*10))
	})

	AfterEach(func() {
		err = mysqlContainer.Terminate(ctx)
		Expect(err).NotTo(HaveOccurred())
		err = k8sClient.Delete(ctx, adminConnection)
		Expect(err).NotTo(HaveOccurred())
		adminConnection = nil
	})

})
