# It's necessary to set this because some environments don't link sh -> bash.
SHELL := /bin/bash

#-----------------------------------------------------------------------------
# VERBOSE target
#-----------------------------------------------------------------------------

# When you run make VERBOSE=1 (the default), executed commands will be printed
# before executed. If you run make VERBOSE=2 verbose flags are turned on and
# quiet flags are turned off for various commands. Use V_FLAG in places where
# you can toggle on/off verbosity using -v. Use Q_FLAG in places where you can
# toggle on/off quiet mode using -q. Use S_FLAG where you want to toggle on/off
# silence mode using -s...
VERBOSE ?= 1
Q = @
Q_FLAG = -q
QUIET_FLAG = --quiet
V_FLAG =
S_FLAG = -s
X_FLAG =
ZAP_ENCODER_FLAG = --zap-level=debug --zap-encoder=console
ZAP_LEVEL_FLAG =
ifeq ($(VERBOSE),1)
	Q =
endif
ifeq ($(VERBOSE),2)
	Q =
	Q_FLAG =
	QUIET_FLAG =
	S_FLAG =
	V_FLAG = -v
	X_FLAG = -x
	ZAP_LEVEL_FLAG = --zap-level 1
endif
ifeq ($(VERBOSE),3)
	Q_FLAG =
	QUIET_FLAG =
	S_FLAG =
	V_FLAG = -v
	X_FLAG = -x
	ZAP_LEVEL_FLAG = --zap-level 2
endif

ZAP_FLAGS = $(ZAP_ENCODER_FLAG) $(ZAP_LEVEL_FLAG)


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
