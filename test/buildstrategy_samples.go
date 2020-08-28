package test

// MinimalBuildahBuildStrategy defines a
// BuildStrategy for Buildah with two steps
// each of them with different container resources
const MinimalBuildahBuildStrategy = `
apiVersion: build.dev/v1alpha1
kind: BuildStrategy
metadata:
  name: buildah
spec:
  buildSteps:
    - name: step-buildah-bud
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
    - name: step-buildah-push
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

// BuildahBuildStrategySingleStep defines a
// BuildStrategy for Buildah with a single step
// and container resources
const BuildahBuildStrategySingleStep = `
apiVersion: build.dev/v1alpha1
kind: BuildStrategy
metadata:
  name: buildah
spec:
  buildSteps:
    - name: build
      image: "$(build.builder.image)"
      workingDir: /workspace/source
      command:
        - buildah
        - bud
        - --tls-verify=false
        - --layers
        - -f
        - $(build.dockerfile)
        - -t
        - $(build.output.image)
        - $(build.source.contextDir)
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
apiVersion: build.dev/v1alpha1
kind: BuildStrategy
metadata:
  name: buildpacks-v3
spec:
  buildSteps:
    - name: build
      image: "$(build.builder.image)"
      workingDir: /workspace/source
      command:
        - /cnb/lifecycle/builder
        - -app
        - /workspace/source
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
