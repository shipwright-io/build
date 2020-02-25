# It's necessary to set this because some environments don't link sh -> bash.
SHELL := /bin/bash

#-----------------------------------------------------------------------------
# VERBOSE target
#-----------------------------------------------------------------------------

# When you run make VERBOSE=1, executed commands will be printed
# before executed.
VERBOSE ?= 0
Q = @
ifeq ($(VERBOSE),1)
	Q =
endif

# output directory, where all artifacts will be created and managed
OUTPUT_DIR ?= build/_output
# relative path to operator binary
OPERATOR = $(OUTPUT_DIR)/bin/operator
# golang cache directory path
GOCACHE ?= "$(shell echo ${PWD})/$(OUTPUT_DIR)/gocache"

default: build

.PHONY: vendor
vendor: go.mod go.sum
	$(Q)GOCACHE=$(GOCACHE) go mod vendor ${V_FLAG}

.PHONY: build
build: $(OPERATOR)

$(OPERATOR): vendor
	$(Q)GOCACHE=$(GOCACHE) GOARCH=amd64 GOOS=linux go build -o $(OPERATOR) cmd/manager/main.go

local:
	- hack/crd.sh uninstall
	@hack/crd.sh install
	operator-sdk run --local

clean:
	rm -rfv $(OUTPUT_DIR)

.PHONY: test
test: build
	$(Q)GOCACHE=$(GOCACHE) go test ./pkg/apis/... ./pkg/controller/...
