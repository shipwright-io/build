#!/bin/bash

# Copyright The Shipwright Contributors
# 
# SPDX-License-Identifier: Apache-2.0


set -euo pipefail

[ ! -d "vendor" ] && echo "$0 requires vendor/ folder, run 'go mod vendor'"

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
BIN=$(go env GOPATH)/bin

GO111MODULE=off "${BIN}/counterfeiter" -header "${SCRIPT_ROOT}/hack/boilerplate.go.txt" -o pkg/controller/fakes/manager.go vendor/sigs.k8s.io/controller-runtime/pkg/manager Manager
GO111MODULE=off "${BIN}/counterfeiter" -header "${SCRIPT_ROOT}/hack/boilerplate.go.txt" -o pkg/controller/fakes/client.go vendor/sigs.k8s.io/controller-runtime/pkg/client Client
GO111MODULE=off "${BIN}/counterfeiter" -header "${SCRIPT_ROOT}/hack/boilerplate.go.txt" -o pkg/controller/fakes/status_writer.go vendor/sigs.k8s.io/controller-runtime/pkg/client StatusWriter
