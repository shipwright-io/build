# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0
FROM ubuntu:21.04
RUN apt update && apt install -y golang make git && \
    go get -u github.com/onsi/ginkgo/ginkgo && \
    go get github.com/axw/gocov/gocov && \
    go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.5.0 && \
    mkdir /build && chmod a+rwx /build
WORKDIR /src
ENV GOPATH=/root/go
