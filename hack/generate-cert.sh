# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0

#!/bin/bash

set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"

echo "[INFO] Generating key for Shipwright Build Webhook"

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
openssl req -new -days 365  -key /tmp/server-key.pem -subj "/O=system:nodes/CN=system:node:shp-build-webhook.shipwright-build.svc.cluster.local" -out /tmp/server.csr -config /tmp/csr.conf
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
  request: $(cat /tmp/server.csr | base64 | tr -d '\n')
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


echo "[INFO] Creating shipwright-build namespace"
kubectl apply -f $DIR/deploy/100-namespace.yaml

echo "[INFO] Creating Opaque Secret with generated certificates"
kubectl create secret tls shipwright-build-webhook-cert -n shipwright-build --cert /tmp/server-cert.pem --key /tmp/server-key.pem

rm -rf /tmp/csr.conf /tmp/server-cert.pem /tmp/server-key.pem

echo "[INFO] Retrieving CABundle"
CA=$(kubectl get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' | base64 | tr -d '\n')

echo "[INFO] Applying CABundle into customization/conversion_webhook_block.yaml"
sed -i "s/CA_BUNDLE/${CA}/g" $DIR/hack/customization/conversion_webhook_block.yaml