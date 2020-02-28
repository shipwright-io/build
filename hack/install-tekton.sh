#!/bin/bash
#
# Intall Tekton in the "kind" cluster
#

set -eu

TEKTON_VERSION="v0.10.1"

kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/previous/${TEKTON_VERSION}/release.yaml
