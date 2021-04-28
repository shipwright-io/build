FROM registry.access.redhat.com/ubi8-minimal:latest

RUN \
  microdnf update && \
  microdnf install git git-lfs && \
  microdnf clean all && \
  rm -rf /var/cache/yum && \
  echo 'nonroot:x:1000:1000:nonroot:/:/sbin/nologin' > /etc/passwd && \
  echo 'nonroot:x:1000:' > /etc/group

USER 1000:1000
