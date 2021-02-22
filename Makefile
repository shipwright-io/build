SHELL := /bin/bash

# output directory, where all artifacts will be created and managed
OUTPUT_DIR ?= build/_output
# relative path to operator binary
OPERATOR = $(OUTPUT_DIR)/bin/build-operator

# golang cache directory path
GOCACHE ?= $(shell echo ${PWD})/$(OUTPUT_DIR)/gocache
# golang target architecture
GOARCH ?= amd64
# golang global flags
GO_FLAGS ?= -v -mod=vendor -ldflags=-w

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# configure zap based logr
ZAP_FLAGS ?= --zap-log-level=debug --zap-encoder=console
# extra flags passed to operator-sdk
OPERATOR_SDK_EXTRA_ARGS ?= --debug

# test namespace name
TEST_NAMESPACE ?= default

# CI: tekton pipelines operator version
TEKTON_VERSION ?= v0.20.1
# CI: operator-sdk version
SDK_VERSION ?= v0.18.2

# E2E test flags
TEST_E2E_FLAGS ?= -failFast -p -randomizeAllSpecs -slowSpecThreshold=300 -timeout=30m -progress -stream -trace -v

# E2E test operator behavior, can be start_local or managed_outside
TEST_E2E_OPERATOR ?= start_local

# E2E test service account name to be used for the build runs, can be set to generated to use the generated service account feature
TEST_E2E_SERVICEACCOUNT_NAME ?= pipeline

# E2E test build global object creation (custom resource definitions and build strategies)
TEST_E2E_CREATE_GLOBALOBJECTS ?= true

# E2E test verify Tekton objects
TEST_E2E_VERIFY_TEKTONOBJECTS ?= true

# E2E test timeout multiplier
TEST_E2E_TIMEOUT_MULTIPLIER ?= 1

# test repository to store images build during end-to-end tests
TEST_IMAGE_REPO ?= quay.io/shipwright-io/build-e2e
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
IMAGE_HOST ?= quay.io
IMAGE ?= shipwright/shipwright-operator
TAG ?= latest

.EXPORT_ALL_VARIABLES:

default: build

.PHONY: vendor
vendor: go.mod go.sum
	go mod vendor

.PHONY: build
build: $(OPERATOR)

$(OPERATOR): vendor
	go build -trimpath $(GO_FLAGS) -o $(OPERATOR) cmd/manager/main.go

.PHONY: build-plain
build-plain: 
	go build -trimpath $(GO_FLAGS) -o $(OPERATOR) cmd/manager/main.go

.PHONY: build-image
build-image:
	KO_DOCKER_REPO="$(IMAGE_HOST)/$(IMAGE)" GOFLAGS="$(GO_FLAGS)" ko publish --bare ./cmd/manager

.PHONY: build-image-with-pprof
build-image-with-pprof:
	KO_DOCKER_REPO="$(IMAGE_HOST)/$(IMAGE)" GOFLAGS="$(GO_FLAGS) -tags=pprof_enabled" ko publish --bare --tags=pprof ./cmd/manager

.PHONY: release
release:
	hack/release.sh

.PHONY: generate
generate:
	hack/generate-client.sh
	hack/generate-fakes.sh
	hack/generate-copyright.sh

.PHONY: verify-codegen
verify-codegen: generate
	# TODO: Fix travis issue with ginkgo install updating go.mod and go.sum
	# TODO: Verify vendor tree is accurate
	git diff --quiet -- ':(exclude)go.mod' ':(exclude)go.sum' ':(exclude)vendor/*'

ginkgo:
ifeq (, $(shell which ginkgo))
	@{ \
	set -e ;\
	GINKGO_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$GINKGO_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get -u github.com/onsi/ginkgo/ginkgo ;\
	go get -u github.com/onsi/gomega/... ;\
	rm -rf $$GINKGO_GEN_TMP_DIR ;\
	}
GINKGO=$(GOBIN)/ginkgo
else
GINKGO=$(shell which ginkgo)
endif

gocov:
ifeq (, $(shell which gocov))
	@{ \
	set -e ;\
	GOCOV_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$GOCOV_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get github.com/axw/gocov/gocov@v1.0.0 ;\
	rm -rf $$GOCOV_GEN_TMP_DIR ;\
	}
GOCOV=$(GOBIN)/gocov
else
GOCOV=$(shell which gocov)
endif

install-counterfeiter:
	hack/install-counterfeiter.sh

install-operator-sdk:
	hack/install-operator-sdk.sh

.PHONY: govet
govet:
	@echo "Checking go vet"
	@go vet ./...

# Install it via: go get -u github.com/gordonklaus/ineffassign
.PHONY: ineffassign
ineffassign:
	@echo "Checking ineffassign"
	@ineffassign ./...

# Install it via: go get -u golang.org/x/lint/golint
# See https://github.com/golang/lint/issues/320 for details regarding the grep
.PHONY: golint
golint:
	@echo "Checking golint"
	@go list ./... | grep -v -e /vendor -e /test | xargs -L1 golint -set_exit_status

# Install it via: go get -u github.com/client9/misspell/cmd/misspell
.PHONY: misspell
misspell:
	@echo "Checking misspell"
	@find . -type f -not -path './vendor/*' -not -path './.git/*' -not -path './build/*' -print0 | xargs -0 misspell -source=text -error

# Install it via: go get -u honnef.co/go/tools/cmd/staticcheck
.PHONY: staticcheck
staticcheck:
	@echo "Checking staticcheck"
	@go list ./... | grep -v /test | xargs staticcheck

.PHONY: sanity-check
sanity-check: ineffassign golint govet misspell staticcheck

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

# Based on https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/integration-tests.md
.PHONY: test-integration
test-integration: install-apis ginkgo
	GO111MODULE=on $(GINKGO) \
		-randomizeAllSpecs \
		-randomizeSuites \
		-failOnPending \
		-slowSpecThreshold=240 \
		-trace \
		test/integration/...


.PHONY: test-e2e
test-e2e: install-strategies test-e2e-plain

.PHONY: test-e2e-plain
test-e2e-plain: ginkgo
	GO111MODULE=on \
	TEST_OPERATOR_NAMESPACE=${TEST_NAMESPACE} \
	TEST_WATCH_NAMESPACE=${TEST_NAMESPACE} \
	TEST_E2E_OPERATOR=${TEST_E2E_OPERATOR} \
	TEST_E2E_CREATE_GLOBALOBJECTS=${TEST_E2E_CREATE_GLOBALOBJECTS} \
	TEST_E2E_SERVICEACCOUNT_NAME=${TEST_E2E_SERVICEACCOUNT_NAME} \
	TEST_E2E_TIMEOUT_MULTIPLIER=${TEST_E2E_TIMEOUT_MULTIPLIER} \
	TEST_E2E_VERIFY_TEKTONOBJECTS=${TEST_E2E_VERIFY_TEKTONOBJECTS} \
	$(GINKGO) ${TEST_E2E_FLAGS} test/e2e

.PHONY: install install-apis install-operator install-strategies

install:
	KO_DOCKER_REPO="$(IMAGE_HOST)/$(IMAGE)" GOFLAGS="$(GO_FLAGS)" ko apply --bare -R -f deploy/

install-with-pprof:
	GOFLAGS="$(GO_FLAGS) -tags=pprof_enabled" ko apply -R -f deploy/

install-apis:
	kubectl apply -f deploy/crds/
	# Wait for the CRD type to be established; this can take a second or two.
	kubectl wait --timeout=10s --for condition=established crd/clusterbuildstrategies.build.dev

install-operator: install-apis
	KO_DOCKER_REPO="$(IMAGE_HOST)/$(IMAGE)" GOFLAGS="$(GO_FLAGS)" ko apply --bare -f deploy/

install-operator-kind: install-apis
	KO_DOCKER_REPO=kind.local GOFLAGS="$(GO_FLAGS)" ko apply -f deploy/

install-strategies: install-apis
	kubectl apply -R -f samples/buildstrategy/

local: vendor install-strategies
	OPERATOR_NAME=build-operator \
	operator-sdk run local --operator-flags="$(ZAP_FLAGS)"

local-plain: vendor
	OPERATOR_NAME=build-operator \
	operator-sdk run local --operator-flags="$(ZAP_FLAGS)"

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
