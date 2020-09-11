#!/bin/bash

# Copyright The Shipwright Contributors
# 
# SPDX-License-Identifier: Apache-2.0

#
# Installs Tekton Pipelines operator.
#

set -eu

TEKTON_VERSION="${TEKTON_VERSION:-v0.14.2}"

TEKTON_HOST="github.com"
TEKTON_HOST_PATH="tektoncd/pipeline/releases/download"
PLATFORM="${1:-k8s}"
echo "# Deploying Tekton Pipelines Operator '${TEKTON_VERSION}' on ${PLATFORM}"

kubectl apply \
    --filename="https://${TEKTON_HOST}/${TEKTON_HOST_PATH}/${TEKTON_VERSION}/release.yaml" \
    --output="yaml"

if [[ ${PLATFORM} == "openshift" ]]; then
    echo "Granting additional privileges to the tekton-pipelines-controller"
    kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: privileged-scc-role
  labels:
    app.kubernetes.io/instance: default
    app.kubernetes.io/part-of: tekton-pipelines
rules:
- apiGroups: [security.openshift.io]
  resourceNames: [privileged]
  resources: [securitycontextconstraints]
  verbs: [use]
EOF
    kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: tekton-pipelines-privileged
  labels:
    app.kubernetes.io/component: controller
    app.kubernetes.io/instance: default
    app.kubernetes.io/part-of: tekton-pipelines
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: privileged-scc-role
subjects:
- kind: ServiceAccount
  name: tekton-pipelines-controller
  namespace: tekton-pipelines
EOF
fi