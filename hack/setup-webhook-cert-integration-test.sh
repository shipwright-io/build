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
DNS.1 = host.docker.internal
EOF

openssl genrsa -out /tmp/server-key.pem 2048
openssl req -new -days 365 -key /tmp/server-key.pem -subj "/O=system:nodes/CN=system:node:host.docker.internal" -out /tmp/server.csr -config /tmp/csr.conf

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

certificate="$(kubectl get csr shipwright-build-webhook-csr -o json | jq -r '.status.certificate')"
while [ "${certificate}" == "null" ]; do
  echo "[INFO] Waiting for certificate to be ready"
  sleep 1
  certificate="$(kubectl get csr shipwright-build-webhook-csr -o json | jq -r '.status.certificate')"
done

openssl base64 -d -A -out /tmp/server-cert.pem <<<"${certificate}"

echo "[INFO] Deleting the CertificateSigningRequest"
kubectl delete csr shipwright-build-webhook-csr --ignore-not-found
rm -rf /tmp/csr.conf

echo "[INFO] Retrieving CABundle"
CA="$(kubectl get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' | base64 | tr -d '\n')"

echo "[INFO] Patching caBundle into CustomResourceDefinitions"
kubectl patch crd clusterbuildstrategies.shipwright.io --type=json -p "[{\"op\":\"replace\",\"path\":\"/spec/conversion/webhook/clientConfig\",\"value\":{\"caBundle\":\"${CA}\",\"url\":\"https://host.docker.internal:30443/convert\"}}]"
kubectl patch crd buildstrategies.shipwright.io --type=json -p "[{\"op\":\"replace\",\"path\":\"/spec/conversion/webhook/clientConfig\",\"value\":{\"caBundle\":\"${CA}\",\"url\":\"https://host.docker.internal:30443/convert\"}}]"
kubectl patch crd builds.shipwright.io --type=json -p "[{\"op\":\"replace\",\"path\":\"/spec/conversion/webhook/clientConfig\",\"value\":{\"caBundle\":\"${CA}\",\"url\":\"https://host.docker.internal:30443/convert\"}}]"
kubectl patch crd buildruns.shipwright.io --type=json -p "[{\"op\":\"replace\",\"path\":\"/spec/conversion/webhook/clientConfig\",\"value\":{\"caBundle\":\"${CA}\",\"url\":\"https://host.docker.internal:30443/convert\"}}]"
