/*
Copyright 2021, 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/docker/docker/api/types/container"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"net"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	// +kubebuilder:scaffold:imports
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsServer "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/testcontainers/testcontainers-go"
	. "github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/wait"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc

var ServerAdminConnection *AdminConnection
var mysqlContainer *MySQLContainer

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")},
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join("..", "..", "config", "webhook")},
		},
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	scheme := runtime.NewScheme()
	err = AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = clientgoscheme.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = admissionv1beta1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// start webhook server using Manager
	disableHTTP2 := func(c *tls.Config) {
		c.NextProtos = []string{"http/1.1"}
	}

	webhookInstallOptions := webhook.Options{
		Host:    testEnv.WebhookInstallOptions.LocalServingHost,
		Port:    testEnv.WebhookInstallOptions.LocalServingPort,
		CertDir: testEnv.WebhookInstallOptions.LocalServingCertDir,
		TLSOpts: []func(config *tls.Config){disableHTTP2},
	}
	webhookServer := webhook.NewServer(webhookInstallOptions)

	metricsOptions := metricsServer.Options{
		BindAddress:   "0",
		SecureServing: false,
		TLSOpts:       []func(*tls.Config){disableHTTP2},
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:         scheme,
		WebhookServer:  webhookServer,
		LeaderElection: false,
		Metrics:        metricsOptions,
	})
	Expect(err).NotTo(HaveOccurred())

	err = (&Database{}).SetupWebhookWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:webhook

	go func() {
		err = mgr.Start(ctx)
		if err != nil {
			Expect(err).NotTo(HaveOccurred())
		}
	}()

	// wait for the webhook server to get ready
	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%d", webhookInstallOptions.Host, webhookInstallOptions.Port)
	Eventually(func() error {
		conn, err := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			return err
		}
		_ = conn.Close()
		return nil
	}).Should(Succeed())

	// start mysql container
	image, ok := os.LookupEnv("MYSQL_IMAGE")
	if !ok {
		image = "ghcr.io/cuppett/mariadb:11.0"
	}

	mysqlContainer, err = RunContainer(ctx, testcontainers.WithImage(image),
		WithUsername("root"), WithPassword(""),
		testcontainers.WithConfigModifier(func(config *container.Config) {
			config.Env = []string{"MYSQL_ALLOW_EMPTY_PASSWORD=true"}
		}),
		testcontainers.WithWaitStrategy(wait.ForLog(": ready for connections.").WithOccurrence(2).WithStartupTimeout(time.Minute*5)),
	)
	Expect(err).NotTo(HaveOccurred())

	hostname, err := mysqlContainer.Host(ctx)
	Expect(err).NotTo(HaveOccurred())
	port, err := mysqlContainer.MappedPort(ctx, "3306")
	Expect(err).NotTo(HaveOccurred())

	// create admin connection
	ServerAdminConnection = &AdminConnection{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: AdminConnectionSpec{
			Host: hostname,
			Port: int32(port.Int()),
		},
	}
	err = k8sClient.Create(ctx, ServerAdminConnection)
	Expect(err).NotTo(HaveOccurred())

	// Fetch the actual instance back
	adminConnectionNamespacedName := types.NamespacedName{
		Namespace: ServerAdminConnection.Namespace,
		Name:      ServerAdminConnection.Name,
	}
	ServerAdminConnection = &AdminConnection{}
	err = k8sClient.Get(ctx, adminConnectionNamespacedName, ServerAdminConnection)
	Expect(err).NotTo(HaveOccurred())

	// Setting/saving status as if the controllers were running
	ServerAdminConnection.Status.Message = "Successfully pinged database"
	ServerAdminConnection.Status.SyncTime = metav1.Now()
	ServerAdminConnection.Status.ControlDatabase = "zz_mysql_dba_operator_control"
	ServerAdminConnection.Status.CharacterSet = "utf8mb4"
	ServerAdminConnection.Status.Collation = "utf8mb4_general_ci"
	ServerAdminConnection.Status.AvailableCharsets = []Charset{
		{
			Name: "utf8mb4",
			Collations: []Collation{
				{
					Name:    "utf8mb4_general_ci",
					Default: true,
				},
				{
					Name:    "utf8mb4_unicode_ci",
					Default: false,
				},
			},
		},
	}
	err = k8sClient.Status().Update(ctx, ServerAdminConnection)
	Expect(err).NotTo(HaveOccurred())

	// Adding a basic secret to the cluster
	secret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"key": []byte("value"),
		},
	}
	err = k8sClient.Create(ctx, &secret)
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	var err error

	err = mysqlContainer.Terminate(ctx)
	Expect(err).NotTo(HaveOccurred())
	err = k8sClient.Delete(ctx, ServerAdminConnection)
	Expect(err).NotTo(HaveOccurred())
	ServerAdminConnection = nil

	cancel()
	By("tearing down the test environment")
	err = testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
