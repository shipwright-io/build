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
GO_FLAGS ?= -v -mod=vendor
# configure zap based logr
ZAP_ENCODER_FLAG = --zap-level=debug --zap-encoder=console
# CI: tekton pipelines operator version
TEKTON_VERSION ?= v0.10.1
# CI: operator-sdk version
SDK_VERSION ?= v0.15.2

.EXPORT_ALL_VARIABLES:

default: build

.PHONY: vendor
vendor: go.mod go.sum
	go mod vendor

.PHONY: build
build: $(OPERATOR)

$(OPERATOR): vendor
	go build $(GO_FLAGS) -o $(OPERATOR) cmd/manager/main.go

.PHONY: test
test:
	GO111MODULE=on ginkgo \
	  -randomizeAllSpecs \
	  -randomizeSuites \
	  -failOnPending \
	  -nodes=4 \
	  -compilers=2 \
	  -slowSpecThreshold=240 \
	  -race \
	  -cover \
	  -trace \
	  internal/... \
	  pkg/...

local:
	-hack/crd.sh uninstall
	@hack/crd.sh install
	operator-sdk run --local --operator-flags="$(ZAP_ENCODER_FLAG)"

clean:
	rm -rf $(OUTPUT_DIR)

gen-fakes:
	./hack/generate-fakes.sh

travis:
	./hack/install-operator-sdk.sh
	./hack/install-kind.sh
	./hack/install-tekton.sh
