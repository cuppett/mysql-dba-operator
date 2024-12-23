/*
Copyright 2021.

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

package controllers

import (
	"context"
	"crypto/tls"
	"github.com/cuppett/mysql-dba-operator/orm"
	"github.com/docker/docker/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	. "github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/wait"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsServer "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	mysqlv1alpha1 "github.com/cuppett/mysql-dba-operator/api/v1alpha1"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc

var ServerAdminConnection *mysqlv1alpha1.AdminConnection
var mysqlContainer *MySQLContainer

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithCancel(context.TODO())
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = mysqlv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = mysqlv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = mysqlv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	webhookInstallOptions := webhook.Options{
		Host:    testEnv.WebhookInstallOptions.LocalServingHost,
		Port:    testEnv.WebhookInstallOptions.LocalServingPort,
		CertDir: testEnv.WebhookInstallOptions.LocalServingCertDir,
	}
	webhookServer := webhook.NewServer(webhookInstallOptions)

	metricsOptions := metricsServer.Options{
		BindAddress:   "0",
		SecureServing: false,
		TLSOpts:       []func(*tls.Config){},
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:         scheme.Scheme,
		WebhookServer:  webhookServer,
		LeaderElection: false,
		Metrics:        metricsOptions,
	})
	Expect(err).ToNot(HaveOccurred())

	var connectionCache = make(map[types.UID]*orm.ConnectionDefinition)

	err = (&mysqlv1alpha1.Database{}).SetupWebhookWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	err = (&AdminConnectionReconciler{
		Client:      mgr.GetClient(),
		Scheme:      mgr.GetScheme(),
		Connections: connectionCache,
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	err = (&DatabaseReconciler{
		Client:      mgr.GetClient(),
		Scheme:      mgr.GetScheme(),
		Connections: connectionCache,
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	err = (&DatabaseUserReconciler{
		Client:      mgr.GetClient(),
		Scheme:      mgr.GetScheme(),
		Connections: connectionCache,
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

	image, ok := os.LookupEnv("MYSQL_IMAGE")
	if !ok {
		image = "ghcr.io/cuppett/mariadb:11.0"
	}

	mysqlContainer, err = Run(ctx, image,
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

	ServerAdminConnection = &mysqlv1alpha1.AdminConnection{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: mysqlv1alpha1.AdminConnectionSpec{
			Host: hostname,
			Port: int32(port.Int()),
		},
	}
	err = k8sClient.Create(ctx, ServerAdminConnection)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	err := mysqlContainer.Terminate(ctx)
	Expect(err).NotTo(HaveOccurred())
	err = k8sClient.Delete(ctx, ServerAdminConnection)
	Expect(err).NotTo(HaveOccurred())
	ServerAdminConnection = nil

	cancel()
	By("tearing down the test environment")
	err = testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
