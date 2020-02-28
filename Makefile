SHELL := /bin/bash

# output directory, where all artifacts will be created and managed
OUTPUT_DIR ?= build/_output
# relative path to operator binary
OPERATOR = $(OUTPUT_DIR)/bin/operator
# golang cache directory path
GOCACHE ?= "$(shell echo ${PWD})/$(OUTPUT_DIR)/gocache"
# golang global flags
GO_FLAGS ?= -v -mod=vendor
# golang test floags
GO_TEST_FLAGS ?= -failfast
# configure zap based logr
ZAP_ENCODER_FLAG = --zap-level=debug --zap-encoder=console

default: build

env:
	export GOCACHE=$(GOCACHE)
	export GOARCH=amd64

.PHONY: vendor
vendor: env go.mod go.sum
	go mod vendor

.PHONY: build
build: env $(OPERATOR)

$(OPERATOR): vendor
	go build $(GO_FLAGS) -o $(OPERATOR) cmd/manager/main.go

.PHONY: test
test: env
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
