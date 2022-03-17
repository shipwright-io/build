SHELL := /bin/bash

# output directory, where all artifacts will be created and managed
OUTPUT_DIR ?= build/_output
# relative path to controller binary
CONTROLLER = $(OUTPUT_DIR)/bin/shipwright-build-controller

# golang cache directory path
GOCACHE ?= $(shell echo ${PWD})/$(OUTPUT_DIR)/gocache

# golang target architecture
# Check if GO_OS is defined, if not set it to `linux` which is the only supported OS
# this provides flexibility to users to set GO_OS in the future if require
ifeq ($(origin GO_OS), undefined)
GO_OS = "linux"
endif
GO_ARCH ?= $(shell uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/')

# golang global flags
GO_FLAGS ?= -v -mod=vendor -ldflags=-w

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# GINKGO is the path to the ginkgo cli
GINKGO ?= $(shell which ginkgo)

# configure zap based logr
ZAP_FLAGS ?= --zap-log-level=debug --zap-encoder=console

# test namespace name
TEST_NAMESPACE ?= default

# CI: tekton pipelines controller version
TEKTON_VERSION ?= v0.30.0

# E2E test flags
TEST_E2E_FLAGS ?= --fail-fast -p --randomize-all --slow-spec-threshold=5m -timeout=1h -progress -trace -v

# E2E test service account name to be used for the build runs, can be set to generated to use the generated service account feature
TEST_E2E_SERVICEACCOUNT_NAME ?= pipeline

# E2E test verify Tekton objects
TEST_E2E_VERIFY_TEKTONOBJECTS ?= true

# E2E test timeout multiplier
TEST_E2E_TIMEOUT_MULTIPLIER ?= 1

# test repository to store images build during end-to-end tests
TEST_IMAGE_REPO ?= ghcr.io/shipwright-io/build/build-e2e
# test container registyr secret name
TEST_IMAGE_REPO_SECRET ?=
# test container registry secret, must be defined during runtime
TEST_IMAGE_REPO_DOCKERCONFIGJSON ?=

# enable private git repository tests
TEST_PRIVATE_REPO ?= false
# github private repository url
TEST_PRIVATE_GITHUB ?=
# gitlab private repository url
TEST_PRIVATE_GITLAB ?=
# private repository authentication secret
TEST_SOURCE_SECRET ?=

# Image settings for building and pushing images
IMAGE_HOST ?= ghcr.io
IMAGE_NAMESPACE ?= shipwright-io/build
TAG ?= latest

# options for generating crds with controller-gen
CONTROLLER_GEN="${GOBIN}/controller-gen"
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"

.EXPORT_ALL_VARIABLES:

default: build

build: $(CONTROLLER)

.PHONY: $(CONTROLLER)
$(CONTROLLER):
	go build -trimpath $(GO_FLAGS) -o $(CONTROLLER) cmd/shipwright-build-controller/main.go

.PHONY: build-plain
build-plain:
	go build -trimpath $(GO_FLAGS) -o $(CONTROLLER) cmd/shipwright-build-controller/main.go

.PHONY: build-image
build-image:
	KO_DOCKER_REPO="$(IMAGE_HOST)/$(IMAGE_NAMESPACE)" GOFLAGS="$(GO_FLAGS)" ko publish --base-import-paths ./cmd/shipwright-build-controller

.PHONY: build-image-with-pprof
build-image-with-pprof:
	KO_DOCKER_REPO="$(IMAGE_HOST)/$(IMAGE_NAMESPACE)" GOFLAGS="$(GO_FLAGS) -tags=pprof_enabled" ko publish --base-import-paths --tags=pprof ./cmd/shipwright-build-controller

.PHONY: release
release:
	hack/release.sh

.PHONY: generate
generate:
	hack/update-codegen.sh
	hack/generate-fakes.sh
	hack/generate-copyright.sh
	hack/install-controller-gen.sh
	"$(CONTROLLER_GEN)" "$(CRD_OPTIONS)" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=deploy/crds

.PHONY: verify-generate
verify-generate: generate
	@hack/verify-generate.sh

ginkgo:
ifeq (, $(GINKGO))
  ifeq (, $(shell which ginkgo))
	@{ \
	set -e ;\
	go install github.com/onsi/ginkgo/v2/ginkgo@latest ;\
	}
  override GINKGO=$(GOBIN)/ginkgo
  else
  override GINKGO=$(shell which ginkgo)
  endif
endif

gocov:
ifeq (, $(shell which gocov))
	@{ \
	set -e ;\
	GOCOV_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$GOCOV_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go install github.com/axw/gocov/gocov@v1.0.0 ;\
	rm -rf $$GOCOV_GEN_TMP_DIR ;\
	}
GOCOV=$(GOBIN)/gocov
else
GOCOV=$(shell which gocov)
endif

install-counterfeiter:
	hack/install-counterfeiter.sh

# Install golangci-lint via: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
.PHONY: sanity-check
sanity-check:
	golangci-lint run

# https://github.com/shipwright-io/build/issues/123
test: test-unit

.PHONY: test-unit
test-unit:
	rm -rf build/coverage
	mkdir build/coverage
	go test \
		./cmd/... \
		./pkg/... \
		-coverprofile=unit.coverprofile \
		-outputdir=build/coverage \
		-race \
		-v

test-unit-coverage: test-unit gocov
	echo "Combining coverage profiles"
	cat build/coverage/*.coverprofile | sed -E 's/([0-9])github.com/\1\ngithub.com/g' | sed -E 's/([0-9])mode: atomic/\1/g' > build/coverage/coverprofile
	$(GOCOV) convert build/coverage/coverprofile > build/coverage/coverprofile.json
	$(GOCOV) report build/coverage/coverprofile.json

.PHONY: test-unit-ginkgo
test-unit-ginkgo: ginkgo
	GO111MODULE=on $(GINKGO) \
		--randomize-all \
		--randomize-suites \
		--fail-on-pending \
		--slow-spec-threshold=4m \
		-compilers=2 \
		-race \
		-trace \
		cmd/... \
		pkg/...

# Based on https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/integration-tests.md
.PHONY: test-integration
test-integration: install-apis ginkgo
	GO111MODULE=on $(GINKGO) \
		--randomize-all \
		--randomize-suites \
		--fail-on-pending \
		--slow-spec-threshold=4m \
		-trace \
		test/integration/...


.PHONY: test-e2e
test-e2e: install-strategies test-e2e-plain

.PHONY: test-e2e-plain
test-e2e-plain: ginkgo
	GO111MODULE=on \
	TEST_CONTROLLER_NAMESPACE=${TEST_NAMESPACE} \
	TEST_WATCH_NAMESPACE=${TEST_NAMESPACE} \
	TEST_E2E_SERVICEACCOUNT_NAME=${TEST_E2E_SERVICEACCOUNT_NAME} \
	TEST_E2E_TIMEOUT_MULTIPLIER=${TEST_E2E_TIMEOUT_MULTIPLIER} \
	TEST_E2E_VERIFY_TEKTONOBJECTS=${TEST_E2E_VERIFY_TEKTONOBJECTS} \
	$(GINKGO) ${TEST_E2E_FLAGS} test/e2e

.PHONY: test-e2e-kind-with-prereq-install
test-e2e-kind-with-prereq-install: ginkgo install-controller-kind install-strategies test-e2e-plain

.PHONY: install install-apis install-controller install-strategies

install:
	@echo "Building Shipwright Build controller for platform ${GO_OS}/${GO_ARCH}"
	GOOS=$(GO_OS) GOARCH=$(GO_ARCH) KO_DOCKER_REPO="$(IMAGE_HOST)/$(IMAGE_NAMESPACE)" GOFLAGS="$(GO_FLAGS)" ko apply --base-import-paths -R -f deploy/

install-with-pprof:
	GOOS=$(GO_OS) GOARCH=$(GO_ARCH) GOFLAGS="$(GO_FLAGS) -tags=pprof_enabled" ko apply -R -f deploy/

install-apis:
	kubectl apply -f deploy/crds/
	# Wait for the CRD type to be established; this can take a second or two.
	kubectl wait --timeout=10s --for condition=established crd/clusterbuildstrategies.shipwright.io

install-controller: install-apis
	@echo "Building Shipwright Build controller for platform ${GO_OS}/${GO_ARCH}"
	GOOS=$(GO_OS) GOARCH=$(GO_ARCH) KO_DOCKER_REPO="$(IMAGE_HOST)/$(IMAGE_NAMESPACE)" GOFLAGS="$(GO_FLAGS)" ko apply --base-import-paths -f deploy/

install-controller-kind: install-apis
	KO_DOCKER_REPO=kind.local GOFLAGS="$(GO_FLAGS)" ko apply -f deploy/

install-strategies: install-apis
	kubectl apply -R -f samples/buildstrategy/

local: install-strategies
	CONTROLLER_NAME=shipwright-build-controller \
	go run cmd/shipwright-build-controller/main.go $(ZAP_FLAGS)

local-plain:
	CONTROLLER_NAME=shipwright-build-controller \
	go run cmd/shipwright-build-controller/main.go $(ZAP_FLAGS)

clean:
	rm -rf $(OUTPUT_DIR)

gen-fakes:
	./hack/generate-fakes.sh

kubectl:
	./hack/install-kubectl.sh

kind-registry:
	./hack/install-registry.sh

kind-tekton:
	./hack/install-tekton.sh

kind:
	./hack/install-kind.sh
	./hack/install-registry.sh
