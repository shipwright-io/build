module github.com/shipwright-io/build

go 1.14

require (
	github.com/go-git/go-git/v5 v5.2.0
	github.com/go-logr/logr v0.3.0
	github.com/go-openapi/spec v0.19.6
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/operator-framework/operator-sdk v0.19.4
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.8.0
	github.com/prometheus/client_model v0.2.0
	github.com/spf13/pflag v1.0.5
	github.com/tektoncd/pipeline v0.20.1
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.19.2
	k8s.io/kube-openapi v0.0.0-20200805222855-6aeccd4b50c6
	k8s.io/kubectl v0.19.2
	k8s.io/utils v0.0.0-20200912215256-4140de9c8800
	knative.dev/pkg v0.0.0-20210107022335-51c72e24c179
	sigs.k8s.io/controller-runtime v0.7.2
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.2.0+incompatible // Required by OLM
	// github.com/operator-framework/operator-registry requires Sirupsen/logrus@v1.7.0
	github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.7.0
	// Pin docker/* to versions used to spinning up local test registries by operator-sdk
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/docker/docker v1.4.2-0.20200203170920-46ec8731fbce
	github.com/go-logr/logr => github.com/go-logr/logr v0.1.0 // v0.2.0 release doesn't have logr.InfoLogger
	k8s.io/client-go => k8s.io/client-go v0.19.2 // Required by prometheus-operator
	k8s.io/code-generator => k8s.io/code-generator v0.19.2
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200410145947-bcb3869e6f29 // resolve `case-insensitive import collision` for gnostic/openapiv2 package
)
