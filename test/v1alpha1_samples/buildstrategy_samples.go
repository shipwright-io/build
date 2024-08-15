// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package testalpha

// MinimalBuildahBuildStrategy defines a
// BuildStrategy for Buildah with two steps
// each of them with different container resources
const MinimalBuildahBuildStrategy = `
apiVersion: shipwright.io/v1alpha1
kind: BuildStrategy
metadata:
  name: buildah
spec:
  volumes:
    - name: buildah-images
      emptyDir: {}
  buildSteps:
    - name: buildah-bud
      image: quay.io/containers/buildah:v1.37.0
      workingDir: $(params.shp-source-root)
      securityContext:
        capabilities:
          add: ["SETFCAP"]
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
      image: quay.io/containers/buildah:v1.37.0
      securityContext:
        capabilities:
          add: ["SETFCAP"]
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

// MinimalBuildahBuildStrategyWithEnvs defines a
// BuildStrategy for Buildah with two steps
// each of them with different container resources
// and env vars
const MinimalBuildahBuildStrategyWithEnvs = `
apiVersion: shipwright.io/v1alpha1
kind: BuildStrategy
metadata:
  name: buildah
spec:
  volumes:
    - name: buildah-images
      emptyDir: {}
  parameters:
    - name: storage-driver
      description: "The storage driver to use, such as 'overlay' or 'vfs'"
      type: string
      default: "vfs"
  buildSteps:
    - name: buildah-bud
      image: quay.io/containers/buildah:v1.37.0
      workingDir: $(params.shp-source-root)
      securityContext:
        capabilities:
          add: ["SETFCAP"]
      command:
        - /usr/bin/buildah
      args:
        - --storage-driver=$(params.storage-driver)
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
      env:
        - name: MY_VAR_1
          value: "my-var-1-buildstrategy-value"
        - name: MY_VAR_2
          valueFrom:
            fieldRef:
              fieldPath: "my-fieldpath"
    - name: buildah-push
      image: quay.io/containers/buildah:v1.37.0
      securityContext:
        capabilities:
          add: ["SETFCAP"]
      command:
        - /usr/bin/buildah
      args:
        - --storage-driver=$(params.storage-driver)
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

// BuildahBuildStrategySingleStep defines a
// BuildStrategy for Buildah with a single step
// and container resources
const BuildahBuildStrategySingleStep = `
apiVersion: shipwright.io/v1alpha1
kind: BuildStrategy
metadata:
  annotations:
    kubernetes.io/ingress-bandwidth: 1M
    clusterbuildstrategy.shipwright.io/dummy: aValue
    kubectl.kubernetes.io/last-applied-configuration: anotherValue
    kubernetes.io/egress-bandwidth: 1M
  name: buildah
spec:
  volumes:
    - name: varlibcontainers
      emptyDir: {}
  parameters:
    - name: storage-driver
      description: "The storage driver to use, such as 'overlay' or 'vfs'"
      type: string
      default: "vfs"
  buildSteps:
    - name: build
      image: "$(build.builder.image)"
      workingDir: $(params.shp-source-root)
      command:
        - buildah
        - --storage-driver=$(params.storage-driver)
        - bud
        - --tls-verify=false
        - --layers
        - -f
        - $(build.dockerfile)
        - -t
        - $(params.shp-output-image)
        - $(params.shp-source-context)
      resources:
        limits:
          cpu: 500m
          memory: 2Gi
        requests:
          cpu: 500m
          memory: 2Gi
      volumeMounts:
        - name: varlibcontainers
          mountPath: /var/lib/containers
`

// BuildpacksBuildStrategySingleStep defines a
// BuildStrategy for Buildpacks with a single step
// and container resources
const BuildpacksBuildStrategySingleStep = `
apiVersion: shipwright.io/v1alpha1
kind: BuildStrategy
metadata:
  name: buildpacks-v3
spec:
  volumes:
    - name: varlibcontainers
      emptyDir: {}
  buildSteps:
    - name: build
      image: "$(build.builder.image)"
      workingDir: $(params.shp-source-root)
      command:
        - /cnb/lifecycle/builder
        - -app
        - $(params.shp-source-context)
        - -layers
        - /layers
        - -group
        - /layers/group.toml
        - plan
        - /layers/plan.toml
      resources:
        limits:
          cpu: 500m
          memory: 2Gi
        requests:
          cpu: 500m
          memory: 2Gi
      volumeMounts:
        - name: varlibcontainers
          mountPath: /var/lib/containers
`

// BuildStrategyWithParameters is a strategy that uses a
// sleep command with a value for its spec.parameters
const BuildStrategyWithParameters = `
apiVersion: shipwright.io/v1alpha1
kind: BuildStrategy
metadata:
  name: strategy-with-param
spec:
  parameters:
  - name: sleep-time
    description: "time in seconds for sleeping"
    default: "1"
  - name: array-param
    description: "An arbitrary array"
    type: array
    defaults: []
  buildSteps:
  - name: sleep30
    image: alpine:latest
    command:
    - sleep
    args:
    - $(params.sleep-time)
  - name: echo-array-sum
    image: alpine:latest
    command:
    - /bin/bash
    args:
    - -c
    - |
      set -euo pipefail

      sum=0

      for var in "$@"
      do
          sum=$((sum+var))
      done

      echo "Sum: ${sum}"
    - --
    - $(params.array-param[*])
`

// BuildStrategyWithoutDefaultInParameter is a strategy that uses a
// sleep command with a value from its spec.parameters, where the parameter
// have no default
const BuildStrategyWithoutDefaultInParameter = `
apiVersion: shipwright.io/v1alpha1
kind: BuildStrategy
metadata:
  name: strategy-with-param-and-no-default
spec:
  parameters:
  - name: sleep-time
    description: "time in seconds for sleeping"
  buildSteps:
  - name: sleep30
    image: alpine:latest
    command:
    - sleep
    args:
    - $(params.sleep-time)
`

// BuildStrategyWithErrorResult is a strategy that always fails
// and surfaces and error reason and message to the user
const BuildStrategyWithErrorResult = `
apiVersion: shipwright.io/v1alpha1
kind: BuildStrategy
metadata:
  name: strategy-with-error-results
spec:
  buildSteps:
  - name: fail-with-error-result
    image: alpine:latest
    command:
    - sh
    args:
    - -c
    - |
      printf "integration test error reason" > $(results.shp-error-reason.path);
      printf "integration test error message" > $(results.shp-error-message.path);
      return 1
`

// BuildStrategyWithParameterVerification is a strategy that verifies that parameters can be used at all expected places
const BuildStrategyWithParameterVerification = `
apiVersion: shipwright.io/v1alpha1
kind: BuildStrategy
metadata:
  name: strategy-with-parameter-verification
spec:
  parameters:
  - name: env1
    description: "This parameter will be used in the env of the build step"
    type: string
  - name: env2
    description: "This parameter will be used in the env of the build step"
    type: string
  - name: env3
    description: "This parameter will be used in the env of the build step"
    type: string
  - name: image
    description: "This parameter will be used as the image of the build step"
    type: string
  - name: commands
    description: "This parameter will be used as the command of the build step"
    type: array
  - name: args
    description: "This parameter will be used as the args of the build step"
    type: array
  buildSteps:
  - name: calculate-sum
    image: $(params.image)
    env:
    - name: ENV_FROM_PARAMETER1
      value: $(params.env1)
    - name: ENV_FROM_PARAMETER2
      value: $(params.env2)
    - name: ENV_FROM_PARAMETER3
      value: $(params.env3)
    command:
    - $(params.commands[*])
    args:
    - |
      set -euo pipefail

      sum=$((ENV_FROM_PARAMETER1 + ENV_FROM_PARAMETER2 + ENV_FROM_PARAMETER3))

      for var in "$@"
      do
          sum=$((sum+var))
      done

      echo "Sum: ${sum}"
      # Once we have strategy-defined results, then those would be better suitable
      # Until then, just store it as image size :-)
      echo -n "${sum}" > '$(results.shp-image-size.path)'
    - --
    - $(params.args[*])
  securityContext:
    runAsUser: 1000
    runAsGroup: 1000
`

// BuildStrategyWithoutPush is a strategy that writes an image tarball and pushes nothing
const BuildStrategyWithoutPush = `
apiVersion: shipwright.io/v1alpha1
kind: BuildStrategy
metadata:
  name: strategy-without-push
spec:
  buildSteps:
  - name: store-tarball
    image: gcr.io/go-containerregistry/crane:v0.19.2
    command:
    - crane
    args:
    - export
    - busybox
    - $(params.shp-output-directory)/image.tar
`
