module github.com/shipwright-io/build

go 1.16

require (
	github.com/docker/cli v20.10.9+incompatible
	github.com/go-git/go-git/v5 v5.4.2
	github.com/go-logr/logr v0.4.0
	github.com/go-logr/zapr v0.4.0 // indirect
	github.com/go-openapi/spec v0.20.3
	github.com/google/go-containerregistry v0.6.0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.16.0
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/client_model v0.2.0
	github.com/spf13/pflag v1.0.5
	github.com/tektoncd/pipeline v0.27.3
	go.uber.org/zap v1.19.1
	golang.org/x/mod v0.5.1 // indirect
	golang.org/x/sys v0.0.0-20211031064116-611d5d643895 // indirect
	golang.org/x/tools v0.1.7 // indirect
	k8s.io/api v0.20.11
	k8s.io/apimachinery v0.20.11
	k8s.io/client-go v0.20.11
	k8s.io/code-generator v0.20.11
	k8s.io/kube-openapi v0.0.0-20210113233702-8566a335510f
	k8s.io/kubectl v0.20.11
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009
	knative.dev/pkg v0.0.0-20210730172132-bb4aaf09c430
	sigs.k8s.io/controller-runtime v0.6.1
	sigs.k8s.io/yaml v1.3.0
)
