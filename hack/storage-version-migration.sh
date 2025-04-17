#!/bin/bash

# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

if ! hash jq >/dev/null 2>&1 ; then
  echo "[ERROR] jq is not installed"
  exit 1
fi

# Delete old job for storage version migration
kubectl -n shipwright-build delete job --selector app=storage-version-migration-shipwright --wait=true

# create new job for storage version migration
cat <<EOF | kubectl create -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: storage-version-migration-role
rules:
  - apiGroups: ['apiextensions.k8s.io']
    resources: ['customresourcedefinitions', 'customresourcedefinitions/status']
    verbs:     ['get', 'list', 'watch', 'patch']
  - apiGroups: ['shipwright.io']
    resources: ['builds','buildruns', 'buildstrategies', 'clusterbuildstrategies']
    verbs: ['get', 'list', 'watch' ,'patch']
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: storage-version-migration
  namespace: shipwright-build
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: storage-version-migration-rolebinding
subjects:
  - kind: ServiceAccount
    name: storage-version-migration
    namespace: shipwright-build
roleRef:
  kind: ClusterRole
  name: storage-version-migration-role
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: batch/v1
kind: Job
metadata:
  generateName: storage-version-migration-shipwright-
  labels:
    app: storage-version-migration-shipwright
    app.kubernetes.io/component: storage-version-migration-job
    app.kubernetes.io/name: shipwright-build
  namespace: shipwright-build
spec:
  backoffLimit: 10
  template:
    metadata:
      annotations:
        sidecar.istio.io/inject: "false"
      labels:
        app: storage-version-migration-shipwright
        app.kubernetes.io/component: storage-version-migration-job
        app.kubernetes.io/name: shipwright-build
    spec:
      serviceAccountName: storage-version-migration
      containers:
        - args:
            - buildruns.shipwright.io
            - builds.shipwright.io
            - buildstrategies.shipwright.io
            - clusterbuildstrategies.shipwright.io
          image: gcr.io/knative-releases/knative.dev/pkg/apiextensions/storageversion/cmd/migrate
          name: migrate
          resources:
            limits:
              cpu: 1000m
              memory: 1000Mi
            requests:
              cpu: 100m
              memory: 100Mi
          securityContext:
            allowPrivilegeEscalation: false
            capabilities:
              drop:
                - ALL
            readOnlyRootFilesystem: true
            runAsNonRoot: true
            seccompProfile:
              type: RuntimeDefault
      restartPolicy: OnFailure
  ttlSecondsAfterFinished: 600
EOF

JOB_NAME=$(kubectl -n shipwright-build get job --selector app=storage-version-migration-shipwright -o jsonpath='{.items[0].metadata.name}')

while [ "$(kubectl -n shipwright-build get job "${JOB_NAME}" -o json | jq -r '.status.completionTime // ""')" == "" ]; do
    echo "[INFO] Storage version migraton job is still running"
    sleep 10
done

isFailed="$(kubectl -n shipwright-build get job "${JOB_NAME}" -o json | jq -r '.status.conditions[] | select(.type == "Failed") | .status')"

# Delete the ClusterRole, ServiceAccount, and ClusterRoleBinding after the job finishes
kubectl delete clusterrole storage-version-migration-role || true
kubectl delete serviceaccount storage-version-migration -n shipwright-build || true
kubectl delete clusterrolebinding storage-version-migration-rolebinding || true

if [ "${isFailed}" == "True" ]; then
    echo "[ERROR] Storage version migration failed"
    kubectl -n shipwright-build logs "job/${JOB_NAME}"
    exit 1
fi

echo "[DONE]"