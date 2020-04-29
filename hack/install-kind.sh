#!/bin/bash
#
# Installs KinD (Kubernetes in Docker) via "go get" and configure it as current context.
#

set -eu

if [ ! -f "${GOPATH}/bin/kind" ] ; then
    echo "# Installing KinD..."
    go get sigs.k8s.io/kind
fi

# kind cluster name
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-kind}"

echo "# Creating a new Kubernetes cluster..."
kind create cluster --quiet --name="${KIND_CLUSTER_NAME}" --wait=120s

echo "# Using KinD context..."
kubectl config use-context "kind-kind"

echo "# KinD nodes:"
kubectl get nodes
