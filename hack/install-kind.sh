#!/bin/bash
#
# Installs kubectl and KinD for Travis-CI (Ubuntu). Docker is a dependency for this script, in order
# to run KinD, and it needs to use the actual "docker" client in command-line.
#

set -eu

sudo apt-get update > /dev/null && \
    sudo apt-get install -y \
        apt-transport-https \
        curl \
        git

#
# Kubectl
#

echo "# Configuring kubectl APT repository..."

curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg \
    |sudo apt-key add -

if [ ! -f "/etc/apt/sources.list.d/kubernetes.list" ] ; then
    echo "deb https://apt.kubernetes.io/ kubernetes-xenial main" \
        |sudo tee -a /etc/apt/sources.list.d/kubernetes.list
fi

echo "# Installing kubectl..."
sudo apt-get update && \
    sudo apt-get install -y kubectl

echo "# Kubectl version: `kubectl version`"

#
# KinD
#

echo "# Installing KinD..."
go get sigs.k8s.io/kind

echo "# Creating a new Kubernetes cluster..."
kind --quiet create cluster

echo "# Using KinD context..."
kubectl config use-context "kind-kind"

echo "# KinD nodes:"
kubectl get nodes
