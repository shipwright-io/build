#!/bin/bash

set -euo pipefail

[ ! -d "vendor" ] && echo "$0 requires vendor/ folder, run 'go mod vendor'"

counterfeiter -o pkg/controller/fakes/manager.go vendor/sigs.k8s.io/controller-runtime/pkg/manager Manager
counterfeiter -o pkg/controller/fakes/client.go vendor/sigs.k8s.io/controller-runtime/pkg/client Client
counterfeiter -o pkg/controller/fakes/status_writer.go vendor/sigs.k8s.io/controller-runtime/pkg/client StatusWriter