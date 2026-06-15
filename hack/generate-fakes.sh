#!/bin/bash

# Copyright The Shipwright Contributors
# 
# SPDX-License-Identifier: Apache-2.0


set -euo pipefail

[ ! -d "vendor" ] && echo "$0 requires vendor/ folder, run 'go mod vendor'"

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..

counterfeiter -header "${SCRIPT_ROOT}/hack/boilerplate.go.txt" -o pkg/controller/fakes/manager.go sigs.k8s.io/controller-runtime/pkg/manager.Manager
counterfeiter -header "${SCRIPT_ROOT}/hack/boilerplate.go.txt" -o pkg/controller/fakes/client.go sigs.k8s.io/controller-runtime/pkg/client.Client
counterfeiter -header "${SCRIPT_ROOT}/hack/boilerplate.go.txt" -o pkg/controller/fakes/status_writer.go sigs.k8s.io/controller-runtime/pkg/client.StatusWriter
