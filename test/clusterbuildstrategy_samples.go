// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package test

// MinimalBuildahClusterBuildStrategy defines a
// BuildStrategy for Buildah with two steps
// each of them with different container resources
const MinimalBuildahClusterBuildStrategy = `
apiVersion: build.dev/v1alpha1
kind: BuildStrategy
metadata:
  name: buildah
spec:
  buildSteps:
    - name: buildah-bud
      image: quay.io/buildah/stable:latest
      workingDir: /workspace/source
      securityContext:
        privileged: true
      command:
        - /usr/bin/buildah
      args:
        - bud
        - --tag=$(build.output.image)
        - --file=$(build.dockerfile)
        - $(build.source.contextDir)
      resources:
        limits:
          cpu: 500m
          memory: 1Gi
        requests:
          cpu: 500m
          memory: 1Gi
      volumeMounts:
        - name: buildah-images
          mountPath: /var/lib/containers/storage
    - name: buildah-push
      image: quay.io/buildah/stable:latest
      securityContext:
        privileged: true
      command:
        - /usr/bin/buildah
      args:
        - push
        - --tls-verify=false
        - docker://$(build.output.image)
      resources:
        limits:
          cpu: 100m
          memory: 65Mi
        requests:
          cpu: 100m
          memory: 65Mi
      volumeMounts:
        - name: buildah-images
          mountPath: /var/lib/containers/storage
`

// ClusterBuildStrategySingleStep defines a
// BuildStrategy for Buildah with a single step
// and container resources
const ClusterBuildStrategySingleStep = `
apiVersion: build.dev/v1alpha1
kind: ClusterBuildStrategy
metadata:
  name: buildah
spec:
  buildSteps:
    - name: buildah-bud
      image: quay.io/buildah/stable:latest
      workingDir: /workspace/source
      securityContext:
        privileged: true
      command:
        - /usr/bin/buildah
      args:
        - bud
        - --tag=$(build.output.image)
        - --file=$(build.dockerfile)
        - $(build.source.contextDir)
      resources:
        limits:
          cpu: 500m
          memory: 1Gi
        requests:
          cpu: 250m
          memory: 65Mi
      volumeMounts:
        - name: buildah-images
          mountPath: /var/lib/containers/storage
    - name: buildah-push
      image: quay.io/buildah/stable:latest
      securityContext:
        privileged: true
      command:
        - /usr/bin/buildah
      args:
        - push
        - --tls-verify=false
        - docker://$(build.output.image)
      resources:
        limits:
          cpu: 500m
          memory: 1Gi
        requests:
          cpu: 250m
          memory: 65Mi
      volumeMounts:
        - name: buildah-images
          mountPath: /var/lib/containers/storage
`
