SHELL := /bin/bash

# output directory, where all artifacts will be created and managed
OUTPUT_DIR ?= build/_output
# relative path to operator binary
OPERATOR = $(OUTPUT_DIR)/bin/operator
# golang cache directory path
GOCACHE ?= $(shell echo ${PWD})/$(OUTPUT_DIR)/gocache
# golang target architecture
GOARCH ?= amd64
# golang global flags
GO_FLAGS ?= -v -mod=vendor
# golang test floags
GO_TEST_FLAGS ?= -failfast
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
	go test $(GO_FLAGS) $(GO_TEST_FLAGS) ./pkg/apis/... ./pkg/controller/...

local:
	-hack/crd.sh uninstall
	@hack/crd.sh install
	operator-sdk run --local --operator-flags="$(ZAP_ENCODER_FLAG)"

clean:
	rm -rf $(OUTPUT_DIR)

travis:
	./hack/install-operator-sdk.sh
	./hack/install-kind.sh
	./hack/install-tekton.sh
