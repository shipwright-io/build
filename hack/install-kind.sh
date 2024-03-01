#!/bin/bash

# Copyright The Shipwright Contributors
# 
# SPDX-License-Identifier: Apache-2.0

#
# Installs KinD (Kubernetes in Docker) via "go get" and configure it as current context.
#

set -eu

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"

# kind version
KIND_VERSION="${KIND_VERSION:-v0.22.0}"

if ! hash kind > /dev/null 2>&1 ; then
    echo "# Installing KinD..."
    go install "sigs.k8s.io/kind@${KIND_VERSION}"
fi

# print kind version
kind --version

# kind cluster name
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-kind}"

# kind cluster version
KIND_CLUSTER_VERSION="${KIND_CLUSTER_VERSION:-v1.27.11}"

echo "# Creating a new Kubernetes cluster..."
kind delete cluster --name="${KIND_CLUSTER_NAME}"
kind create cluster --name="${KIND_CLUSTER_NAME}" --image="kindest/node:${KIND_CLUSTER_VERSION}" --wait=120s --config="${DIR}/../test/kind/config.yaml"

echo "# Using KinD context..."
kubectl config use-context "kind-kind"

echo "# KinD nodes:"
kubectl get nodes
