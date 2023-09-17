#!/bin/bash

# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

if ! hash jq >/dev/null 2>&1 ; then
  echo "[ERROR] jq is not installed"
  exit 1
fi

if ! hash openssl >/dev/null 2>&1 ; then
  echo "[ERROR] openssl is not installed"
  exit 1
fi

echo "[INFO] Generating key and signing request for Shipwright Build Webhook"

cat <<EOF >/tmp/csr.conf
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names
[alt_names]
DNS.1 = shp-build-webhook
DNS.2 = shp-build-webhook.shipwright-build
DNS.3 = shp-build-webhook.shipwright-build.svc
DNS.4 = shp-build-webhook.shipwright-build.svc.cluster.local
EOF

openssl genrsa -out /tmp/server-key.pem 2048
openssl req -new -days 365 -key /tmp/server-key.pem -subj "/O=system:nodes/CN=system:node:shp-build-webhook.shipwright-build.svc.cluster.local" -out /tmp/server.csr -config /tmp/csr.conf

echo "[INFO] Deleting previous CertificateSigningRequest"
kubectl delete csr shipwright-build-webhook-csr --ignore-not-found

echo "[INFO] Create a CertificateSigningRequest"
cat <<EOF | kubectl create -f -
apiVersion: certificates.k8s.io/v1
kind: CertificateSigningRequest
metadata:
  name: shipwright-build-webhook-csr
spec:
  groups:
  - system:authenticated
  request: $(base64 </tmp/server.csr | tr -d '\n')
  signerName: kubernetes.io/kubelet-serving
  usages:
  - digital signature
  - key encipherment
  - server auth
EOF

echo "[INFO] Approve the CertificateSigningRequest"
kubectl certificate approve shipwright-build-webhook-csr

certificate=$(kubectl get csr shipwright-build-webhook-csr -o json | jq -r '.status.certificate')
while [ "${certificate}" == "null" ]; do
  echo "[INFO] Waiting for certificate to be ready"
  sleep 1
  certificate=$(kubectl get csr shipwright-build-webhook-csr -o json | jq -r '.status.certificate')
done

openssl base64 -d -A -out /tmp/server-cert.pem <<<"${certificate}"

echo "[INFO] Deleting the CertificateSigningRequest"
kubectl delete csr shipwright-build-webhook-csr --ignore-not-found

echo "[INFO] Creating TLS secret shipwright-build-webhook-cert"
kubectl -n shipwright-build delete secret shipwright-build-webhook-cert --ignore-not-found
kubectl -n shipwright-build create secret tls shipwright-build-webhook-cert --cert /tmp/server-cert.pem --key /tmp/server-key.pem
rm -rf /tmp/csr.conf /tmp/server-cert.pem /tmp/server-key.pem

echo "[INFO] Retrieving CABundle"
CA="$(kubectl get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' | base64 | tr -d '\n')"

echo "[INFO] Patching caBundle into CustomResourceDefinitions"
kubectl patch crd clusterbuildstrategies.shipwright.io -p "{\"spec\":{\"conversion\":{\"webhook\":{\"clientConfig\":{\"caBundle\":\"${CA}\"}}}}}"
kubectl patch crd buildstrategies.shipwright.io -p "{\"spec\":{\"conversion\":{\"webhook\":{\"clientConfig\":{\"caBundle\":\"${CA}\"}}}}}"
kubectl patch crd builds.shipwright.io -p "{\"spec\":{\"conversion\":{\"webhook\":{\"clientConfig\":{\"caBundle\":\"${CA}\"}}}}}"
kubectl patch crd buildruns.shipwright.io -p "{\"spec\":{\"conversion\":{\"webhook\":{\"clientConfig\":{\"caBundle\":\"${CA}\"}}}}}"

echo "[INFO] Restarting shipwright-build-webhook"
kubectl -n shipwright-build rollout restart deployment shipwright-build-webhook
kubectl -n shipwright-build rollout status deployment shipwright-build-webhook