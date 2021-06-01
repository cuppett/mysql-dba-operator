module github.com/brightframe/mysql-database-operator

go 1.15

require (
	github.com/go-logr/logr v0.4.0
	github.com/go-sql-driver/mysql v1.6.0
	github.com/onsi/ginkgo v1.16.3
	github.com/onsi/gomega v1.12.0
	github.com/prometheus/common v0.23.0
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	sigs.k8s.io/controller-runtime v0.8.3
)
