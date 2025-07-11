// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package testbeta

// MinimalBuildahClusterBuildStrategy defines a
// BuildStrategy for Buildah with two steps
// each of them with different container resources
const MinimalBuildahClusterBuildStrategy = `
apiVersion: shipwright.io/v1beta1
kind: BuildStrategy
metadata:
  name: buildah
spec:
  volumes:
    - name: buildah-images
      volumeSource:
        emptyDir: {}
  steps:
    - name: buildah-bud
      image: quay.io/containers/buildah:v1.40.1
      workingDir: $(params.shp-source-root)
      securityContext:
        privileged: true
      command:
        - /usr/bin/buildah
      args:
        - bud
        - --tag=$(params.shp-output-image)
        - --file=$(build.dockerfile)
        - $(params.shp-source-context)
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
      image: quay.io/containers/buildah:v1.40.1
      securityContext:
        privileged: true
      command:
        - /usr/bin/buildah
      args:
        - push
        - --tls-verify=false
        - docker://$(params.shp-output-image)
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
apiVersion: shipwright.io/v1beta1
kind: ClusterBuildStrategy
metadata:
  name: buildah
spec:
  volumes:
    - name: buildah-images
      volumeSource:
        emptyDir: {}
  steps:
    - name: buildah-bud
      image: quay.io/containers/buildah:v1.40.1
      workingDir: $(params.shp-source-root)
      securityContext:
        privileged: true
      command:
        - /usr/bin/buildah
      args:
        - bud
        - --tag=$(params.shp-output-image)
        - --file=$(build.dockerfile)
        - $(params.shp-source-context)
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
      image: quay.io/containers/buildah:v1.40.1
      securityContext:
        privileged: true
      command:
        - /usr/bin/buildah
      args:
        - push
        - --tls-verify=false
        - docker://$(params.shp-output-image)
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

// ClusterBuildStrategyForVulnerabilityScanning is a strategy that does nothing and has no dependencies
const ClusterBuildStrategyForVulnerabilityScanning = `
apiVersion: shipwright.io/v1beta1
kind: ClusterBuildStrategy
metadata:
  name: crane-pull
spec:
  steps:
  - name: crane-pull
    image: gcr.io/go-containerregistry/crane:latest
    workingDir: $(params.shp-source-root)
    securityContext:
      runAsUser: 1000
      runAsGroup: 1000
    env:
    - name: DOCKER_CONFIG
      value: /tekton/home/.docker
    - name: HOME
      value: /tekton/home
    command:
    - crane
    args:
    - pull
    - "--format=tarball"
    - "ghcr.io/shipwright-io/shipwright-samples/node:12"
    - "$(params.shp-output-directory)/image.tar"
    resources:
      limits:
        cpu: 250m
        memory: 128Mi
      requests:
        cpu: 250m
        memory: 128Mi
`

// ClusterBuildStrategySingleStepKaniko is a cluster build strategy based on
// Kaniko, which is very close to the actual Kaniko build strategy example in
// the project
const ClusterBuildStrategySingleStepKaniko = `
apiVersion: shipwright.io/v1beta1
kind: ClusterBuildStrategy
metadata:
  name: kaniko
spec:
  steps:
    - name: step-build-and-push
      image: gcr.io/kaniko-project/executor:v1.24.0
      workingDir: $(params.shp-source-root)
      securityContext:
        runAsUser: 0
        capabilities:
          add:
            - CHOWN
            - DAC_OVERRIDE
            - FOWNER
            - SETGID
            - SETUID
            - SETFCAP
            - KILL
      env:
        - name: DOCKER_CONFIG
          value: /tekton/home/.docker
        - name: AWS_ACCESS_KEY_ID
          value: NOT_SET
        - name: AWS_SECRET_KEY
          value: NOT_SET
      command:
        - /kaniko/executor
      args:
        - --skip-tls-verify=true
        - --dockerfile=$(build.dockerfile)
        - --context=$(params.shp-source-context)
        - --destination=$(params.shp-output-image)
        - --snapshot-mode=redo
        - --push-retry=3
      resources:
        limits:
          cpu: 500m
          memory: 1Gi
        requests:
          cpu: 250m
          memory: 65Mi
`

// ClusterBuildStrategySingleStepKanikoError is a Kaniko based cluster build
// strategy that has a configuration error (misspelled command flag) so that
// it will fail in Tekton
const ClusterBuildStrategySingleStepKanikoError = `
apiVersion: shipwright.io/v1beta1
kind: ClusterBuildStrategy
metadata:
  name: kaniko
spec:
  steps:
    - name: step-build-and-push
      image: gcr.io/kaniko-project/executor:v1.24.0
      workingDir: $(params.shp-source-root)
      securityContext:
        runAsUser: 0
        capabilities:
          add:
            - CHOWN
            - DAC_OVERRIDE
            - FOWNER
            - SETGID
            - SETUID
            - SETFCAP
            - KILL
      env:
        - name: DOCKER_CONFIG
          value: /tekton/home/.docker
        - name: AWS_ACCESS_KEY_ID
          value: NOT_SET
        - name: AWS_SECRET_KEY
          value: NOT_SET
      command:
        - /kaniko/executor
      args:
        - --skips-tlss-verifys=true
        - --dockerfile=$(build.dockerfile)
        - --context=$(params.shp-source-context)
        - --destination=$(params.shp-output-image)
        - --snapshot-mode=redo
        - --push-retry=3
      resources:
        limits:
          cpu: 500m
          memory: 1Gi
        requests:
          cpu: 250m
          memory: 65Mi
`

// ClusterBuildStrategyNoOp is a strategy that does nothing and has no dependencies
const ClusterBuildStrategyNoOp = `
apiVersion: shipwright.io/v1beta1
kind: ClusterBuildStrategy
metadata:
  name: noop
spec:
  parameters:
  - name: exit-command
    description: "Exit command for the pod"
    default: "true"
  steps:
  - name: step-no-and-op
    image: alpine:latest
    workingDir: $(params.shp-source-root)
    securityContext:
      runAsUser: 0
      capabilities:
        add:
        - CHOWN
        - DAC_OVERRIDE
        - FOWNER
        - SETGID
        - SETUID
        - SETFCAP
        - KILL
    env:
    - name: DOCKER_CONFIG
      value: /tekton/home/.docker
    - name: AWS_ACCESS_KEY_ID
      value: NOT_SET
    - name: AWS_SECRET_KEY
      value: NOT_SET
    command:
    - $(params.exit-command)
    resources:
      limits:
        cpu: 250m
        memory: 128Mi
      requests:
        cpu: 250m
        memory: 128Mi
`

// ClusterBuildStrategySleep30s is a strategy that does only sleep 30 seconds
const ClusterBuildStrategySleep30s = `
apiVersion: build.dev/v1beta1
kind: ClusterBuildStrategy
metadata:
  name: noop
spec:
  steps:
  - name: sleep30
    image: alpine:latest
    command:
    - sleep
    args:
    - "30s"
    resources:
      limits:
        cpu: 250m
        memory: 128Mi
      requests:
        cpu: 250m
        memory: 128Mi
`

// ClusterBuildStrategyWithAnnotations is a cluster build strategy that contains annotations
const ClusterBuildStrategyWithAnnotations = `
apiVersion: shipwright.io/v1beta1
kind: ClusterBuildStrategy
metadata:
  annotations:
    kubernetes.io/ingress-bandwidth: 1M
    clusterbuildstrategy.shipwright.io/dummy: aValue
    kubectl.kubernetes.io/last-applied-configuration: anotherValue
  name: kaniko
spec:
  steps:
    - name: step-build-and-push
      image: gcr.io/kaniko-project/executor:v1.24.0
      workingDir: $(params.shp-source-root)
      securityContext:
        runAsUser: 0
        capabilities:
          add:
            - CHOWN
            - DAC_OVERRIDE
            - FOWNER
            - SETGID
            - SETUID
            - SETFCAP
            - KILL
      env:
        - name: DOCKER_CONFIG
          value: /tekton/home/.docker
        - name: AWS_ACCESS_KEY_ID
          value: NOT_SET
        - name: AWS_SECRET_KEY
          value: NOT_SET
      command:
        - /kaniko/executor
      args:
        - --skip-tls-verify=true
        - --dockerfile=$(build.dockerfile)
        - --context=$(params.shp-source-root)
        - --destination=$(params.shp-output-image)
        - --snapshot-mode=redo
        - --push-retry=3
      resources:
        limits:
          cpu: 500m
          memory: 1Gi
        requests:
          cpu: 250m
          memory: 65Mi
`

// ClusterBuildStrategyWithParameters is a strategy that uses a
// sleep command with a value for its spec.parameters
const ClusterBuildStrategyWithParameters = `
apiVersion: build.dev/v1beta1
kind: ClusterBuildStrategy
metadata:
  name: strategy-with-param
spec:
  parameters:
  - name: sleep-time
    description: "time in seconds for sleeping"
    default: "1"
  steps:
  - name: sleep30
    image: alpine:latest
    command:
    - sleep
    args:
    - $(params.sleep-time)
`
