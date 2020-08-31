#!/bin/bash

# Copyright The Shipwright Contributors
# 
# SPDX-License-Identifier: Apache-2.0

#
# Installs "kubectl" on Travis-CI Ubuntu.
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
