module github.com/shipwright-io/build

go 1.13

require (
	github.com/go-logr/logr v0.2.0
	github.com/go-openapi/spec v0.19.6
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/operator-framework/operator-sdk v0.17.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.6.0
	github.com/prometheus/client_model v0.2.0
	github.com/spf13/pflag v1.0.5
	github.com/tektoncd/pipeline v0.17.1
	k8s.io/api v0.18.7-rc.0
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.18.6
	k8s.io/kube-openapi v0.0.0-20200805222855-6aeccd4b50c6
	k8s.io/kubectl v0.17.4
	k8s.io/utils v0.0.0-20200603063816-c1c6865ac451
	knative.dev/pkg v0.0.0-20200831162708-14fb2347fb77
	sigs.k8s.io/controller-runtime v0.6.1
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	github.com/go-logr/logr => github.com/go-logr/logr v0.1.0 // v0.2.0 release doesn't have logr.InfoLogger
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.6
	k8s.io/client-go => k8s.io/client-go v0.17.4 // Required by prometheus-operator
	k8s.io/code-generator => k8s.io/code-generator v0.17.4
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200410145947-bcb3869e6f29 // resolve `case-insensitive import collision` for gnostic/openapiv2 package
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.5.2
)
