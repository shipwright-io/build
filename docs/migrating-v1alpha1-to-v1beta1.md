<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

# Migrating from v1alpha1 to v1beta1

- [Migrating from v1alpha1 to v1beta1](#migrating-from-v1alpha1-to-v1beta1)
  - [Overview](#overview)
  - [What the conversion webhook does for you](#what-the-conversion-webhook-does-for-you)
  - [What you must change by hand](#what-you-must-change-by-hand)
  - [Changes to `Build`](#changes-to-build)
    - [Build spec changes](#build-spec-changes)
    - [Build fields removed in v1beta1](#build-fields-removed-in-v1beta1)
    - [Build example](#build-example)
  - [Changes to `BuildRun`](#changes-to-buildrun)
    - [BuildRun spec changes](#buildrun-spec-changes)
    - [BuildRun status changes](#buildrun-status-changes)
    - [BuildRun example](#buildrun-example)
  - [Changes to `BuildStrategy` and `ClusterBuildStrategy`](#changes-to-buildstrategy-and-clusterbuildstrategy)
    - [Strategy spec changes](#strategy-spec-changes)
    - [Strategy placeholder changes](#strategy-placeholder-changes)
    - [Strategy example](#strategy-example)
  - [Fields only available in v1beta1](#fields-only-available-in-v1beta1)
  - [References](#references)

## Overview

The `shipwright.io/v1beta1` API was introduced with the _v0.12.0_ release. It is the **storage version** for all four Shipwright resources (`Build`, `BuildRun`, `BuildStrategy` and `ClusterBuildStrategy`). The `shipwright.io/v1alpha1` API is still **served**, so existing manifests keep working, but it is deprecated and will be removed in a future release.

This guide describes how to move your Custom Resources from `v1alpha1` to `v1beta1`. It focuses on the spec paths that changed. For the complete field reference of the beta API, see [`Build`](build.md), [`BuildRun`](buildrun.md) and [`BuildStrategy`](buildstrategies.md).

Throughout this document:

- **Renamed / moved** means the same information now lives under a different path.
- **Removed** means there is no equivalent in `v1beta1` and the data is dropped.
- Paths are written relative to the resource, for example `.spec.source.git.url`.

## What the conversion webhook does for you

Shipwright ships a conversion webhook that translates between the two versions on every API request. When you `kubectl get` a `v1alpha1` object, it is read from the `v1beta1` storage and converted back, and vice versa. As a consequence:

- Existing `v1alpha1` manifests can still be applied without any change.
- Objects created as `v1alpha1` are readable as `v1beta1` and are stored as `v1beta1`.
- All of the renames and moves listed in the tables below are applied automatically, in both directions, unless the notes say otherwise.

The webhook also performs two conversions that are not simple renames:

| v1alpha1 | v1beta1 | Notes |
|----------|---------|-------|
| `Build` annotation `build.shipwright.io/build-run-deletion` | `.spec.retention.atBuildDeletion` | The string value `"true"` becomes the boolean `true`. The annotation is removed from the converted `v1beta1` object. |
| `BuildRun` `.spec.serviceAccount.generate: true` | `.spec.serviceAccount: ".generate"` | Generated service accounts are deprecated and may be removed in a future release. |

## What you must change by hand

Conversion is lossy in a few places. The following data is **not** carried over, so review these before you rely on the webhook:

- `Build` `.spec.sources[]` entries whose `type` is not `LocalCopy` — for example the deprecated `HTTP` remote artifacts — are dropped. Only the first `LocalCopy` entry is converted.
- `Build` `.spec.builder` sub-fields other than `image` (`credentials`, `insecure`, `annotations`, `labels`, `timestamp`) are dropped. Only `.spec.builder.image` becomes a parameter value.
- `BuildRun` `.spec.output.insecure` and `.spec.output.timestamp` exist in both versions, but are not carried by the conversion webhook. Set them explicitly on the `v1beta1` object.
- `.spec.volumes[].description` on `Build` and `BuildRun` is removed, along with `.spec.strategy.apiVersion` and `BuildRun` `.spec.buildRef.apiVersion`.
- `BuildStrategy` and `ClusterBuildStrategy` steps no longer embed a full Kubernetes `Container`. Any container field outside the supported set is dropped — see [Strategy spec changes](#strategy-spec-changes).

Independently of the webhook, you should update the manifests that you keep under version control, in your GitOps repository, or in your CI pipelines. The webhook keeps old clients working; it is not a substitute for migrating your source of truth.

## Changes to `Build`

### Build spec changes

| v1alpha1 path | v1beta1 path | Notes |
|---------------|--------------|-------|
| _n/a_ | `.spec.source.type` | New and required. One of `Git`, `OCI` or `Local`. The webhook infers it from the `v1alpha1` source that is set. |
| `.spec.source.url` | `.spec.source.git.url` | Requires `.spec.source.type: Git`. The field is no longer optional inside `git`. |
| `.spec.source.revision` | `.spec.source.git.revision` | |
| `.spec.source.credentials.name` | `.spec.source.git.cloneSecret` | For `Git` sources. The type changed from a `LocalObjectReference` to a plain secret name. |
| `.spec.source.credentials.name` | `.spec.source.ociArtifact.pullSecret` | For `OCI` sources. Same type change. |
| `.spec.source.bundleContainer` | `.spec.source.ociArtifact` | Requires `.spec.source.type: OCI`. |
| `.spec.source.bundleContainer.image` | `.spec.source.ociArtifact.image` | |
| `.spec.source.bundleContainer.prune` | `.spec.source.ociArtifact.prune` | Values are unchanged: `Never`, `AfterPull`. |
| `.spec.source.contextDir` | `.spec.source.contextDir` | Unchanged. Stays at the `source` level, next to `type`. |
| `.spec.sources[]` with `type: LocalCopy` | `.spec.source.local` | Requires `.spec.source.type: Local`. `name` and `timeout` are carried over. The plural list is gone. |
| `.spec.sources[]` with any other `type` | _removed_ | The deprecated multiple-sources feature, including `HTTP` remote artifacts, has no beta equivalent. |
| `.spec.dockerfile` | `.spec.paramValues[]` with `name: dockerfile` | The strategy must declare a `dockerfile` parameter. |
| `.spec.builder.image` | `.spec.paramValues[]` with `name: builder-image` | The strategy must declare a `builder-image` parameter. |
| `.spec.output.credentials.name` | `.spec.output.pushSecret` | Type changed from `LocalObjectReference` to a plain secret name. |
| `.spec.trigger.secretRef.name` | `.spec.trigger.triggerSecret` | Type changed from `LocalObjectReference` to a plain secret name. |
| `metadata.annotations["build.shipwright.io/build-run-deletion"]` | `.spec.retention.atBuildDeletion` | Moved from an annotation into the spec, and from a string into a boolean. |
| `.spec.strategy.name`, `.spec.strategy.kind` | unchanged | |
| `.spec.paramValues[]` | unchanged | Including `value`, `values[]`, `configMapValue` and `secretValue`. |
| `.spec.output.image`, `.insecure`, `.annotations`, `.labels`, `.timestamp` | unchanged | |
| `.spec.timeout`, `.spec.env[]` | unchanged | |
| `.spec.retention.failedLimit`, `.succeededLimit`, `.ttlAfterFailed`, `.ttlAfterSucceeded` | unchanged | `v1beta1` additionally caps the limits at `10000`. |
| `.spec.volumes[].name` and the inline volume source | unchanged | |
| `.spec.trigger.when[]` | unchanged | `name`, `type`, `github`, `image` and `objectRef` keep their shape. |
| `.status.registered`, `.status.reason`, `.status.message` | unchanged | The `Build` status is deprecated in both versions. |

### Build fields removed in v1beta1

| v1alpha1 path | Alternative |
|---------------|-------------|
| `.spec.sources[]` | `.spec.source`, only for the `Local` type |
| `.spec.dockerfile` | `.spec.paramValues[]` with `name: dockerfile` |
| `.spec.builder` | `.spec.paramValues[]` with `name: builder-image`, for the image only |
| `.spec.builder.credentials` | none |
| `.spec.volumes[].description` | none |
| `.spec.strategy.apiVersion` | none |

### Build example

Adapted from [`samples/v1alpha1/build/build_kaniko_cr.yaml`](../samples/v1alpha1/build/build_kaniko_cr.yaml) and [`samples/v1beta1/build/build_kaniko_cr.yaml`](../samples/v1beta1/build/build_kaniko_cr.yaml).

```yaml
# v1alpha1
---
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: kaniko-golang-build
  annotations:
    build.shipwright.io/build-run-deletion: "true"
spec:
  source:
    url: https://github.com/shipwright-io/sample-go
    revision: main
    contextDir: docker-build
    credentials:
      name: source-repository-credentials
  strategy:
    name: kaniko
    kind: ClusterBuildStrategy
  dockerfile: Dockerfile
  output:
    image: registry.example.com/build-examples/taxi-app:latest
    credentials:
      name: registry-credentials
```

```yaml
# v1beta1
---
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: kaniko-golang-build
spec:
  source:
    type: Git
    git:
      url: https://github.com/shipwright-io/sample-go
      revision: main
      cloneSecret: source-repository-credentials
    contextDir: docker-build
  strategy:
    name: kaniko
    kind: ClusterBuildStrategy
  paramValues:
    - name: dockerfile
      value: Dockerfile
  retention:
    atBuildDeletion: true
  output:
    image: registry.example.com/build-examples/taxi-app:latest
    pushSecret: registry-credentials
```

## Changes to `BuildRun`

### BuildRun spec changes

| v1alpha1 path | v1beta1 path | Notes |
|---------------|--------------|-------|
| `.spec.buildRef.name` | `.spec.build.name` | |
| `.spec.buildRef.apiVersion` | _removed_ | |
| `.spec.buildSpec` | `.spec.build.spec` | The embedded spec follows the [`Build` mapping](#build-spec-changes). |
| `.spec.sources[]` with `type: LocalCopy` | `.spec.source` with `type: Local` | `name` and `timeout` move to `.spec.source.local`. `Local` is the only source type a `BuildRun` may set. |
| `.spec.serviceAccount.name` | `.spec.serviceAccount` | The object became a plain string. |
| `.spec.serviceAccount.generate: true` | `.spec.serviceAccount: ".generate"` | Deprecated feature, kept for compatibility. |
| `.spec.output.credentials.name` | `.spec.output.pushSecret` | |
| `.spec.output.image`, `.annotations`, `.labels` | unchanged | |
| `.spec.output.insecure`, `.spec.output.timestamp` | unchanged | Present in both versions, but **not** carried by the conversion webhook. Set them explicitly. |
| `.spec.paramValues[]`, `.spec.timeout`, `.spec.env[]`, `.spec.state` | unchanged | |
| `.spec.retention.ttlAfterFailed`, `.ttlAfterSucceeded` | unchanged | |
| `.spec.volumes[].name` and the inline volume source | unchanged | |
| `.spec.volumes[].description` | _removed_ | |

### BuildRun status changes

| v1alpha1 path | v1beta1 path | Notes |
|---------------|--------------|-------|
| `.status.sources[]` | `.status.source` | Cardinality changed from a list to a single object. |
| `.status.sources[].name` | _removed_ | |
| `.status.sources[].git` | `.status.source.git` | |
| `.status.sources[].bundle` | `.status.source.ociArtifact` | |
| `.status.sources[].timestamp` | `.status.source.timestamp` | |
| `.status.latestTaskRunRef` | `.status.taskRunName` | `.status.taskRunName` is itself deprecated in favour of `.status.executor`, which reports both the `name` and the `kind` of the executing resource. |
| `.status.failedAt` | `.status.failureDetails.location` | `.status.failedAt` was already deprecated in `v1alpha1`. |
| `.status.failureDetails.reason`, `.message` | unchanged | |
| `.status.output.digest`, `.size` | unchanged | |
| `.status.conditions[]`, `.startTime`, `.completionTime` | unchanged | |
| `.status.buildSpec` | `.status.buildSpec` | The snapshot follows the [`Build` mapping](#build-spec-changes). |

### BuildRun example

Adapted from [`samples/v1alpha1/buildrun/buildrun_buildah_cr.yaml`](../samples/v1alpha1/buildrun/buildrun_buildah_cr.yaml) and [`samples/v1beta1/buildrun/buildrun_buildah_cr.yaml`](../samples/v1beta1/buildrun/buildrun_buildah_cr.yaml).

```yaml
# v1alpha1
---
apiVersion: shipwright.io/v1alpha1
kind: BuildRun
metadata:
  name: buildah-golang-buildrun
spec:
  buildRef:
    name: buildah-golang-build
  serviceAccount:
    generate: true
  sources:
    - name: local-source
      type: LocalCopy
      timeout: 3m
  output:
    image: registry.example.com/build-examples/taxi-app:dev
    credentials:
      name: registry-credentials
```

```yaml
# v1beta1
---
apiVersion: shipwright.io/v1beta1
kind: BuildRun
metadata:
  name: buildah-golang-buildrun
spec:
  build:
    name: buildah-golang-build
  serviceAccount: ".generate"
  source:
    type: Local
    local:
      name: local-source
      timeout: 3m
  output:
    image: registry.example.com/build-examples/taxi-app:dev
    pushSecret: registry-credentials
```

## Changes to `BuildStrategy` and `ClusterBuildStrategy`

Both kinds share the same spec, so the same mapping applies to each.

### Strategy spec changes

| v1alpha1 path | v1beta1 path | Notes |
|---------------|--------------|-------|
| `.spec.buildSteps[]` | `.spec.steps[]` | |
| `.spec.buildSteps[]` (inline `corev1.Container`) | `.spec.steps[]` (dedicated `Step` type) | A step is no longer a full Kubernetes container. Only `name`, `image`, `command`, `args`, `workingDir`, `env`, `resources`, `volumeMounts`, `imagePullPolicy` and `securityContext` are supported. Any other container field, such as `envFrom`, `ports`, `lifecycle` or the probes, is dropped. |
| `.spec.parameters[]` | unchanged | `name`, `description`, `type`, `default` and `defaults` keep their shape. |
| `.spec.securityContext.runAsUser`, `.runAsGroup` | unchanged | |
| `.spec.volumes[]` | unchanged | Including `overridable`, `name`, `description` and the inline volume source. |

### Strategy placeholder changes

The `v1alpha1` `Build` fields `.spec.dockerfile` and `.spec.builder` became parameter values in `v1beta1`, so the placeholders that strategies used to reference them changed too. The conversion webhook rewrites these placeholders and, when it finds one, appends the matching parameter definition to `.spec.parameters`.

| v1alpha1 placeholder | v1beta1 placeholder | Parameter to declare |
|----------------------|---------------------|----------------------|
| `$(build.dockerfile)` | `$(params.dockerfile)` | `name: dockerfile`, `type: string`, `default: Dockerfile` |
| `$(params.DOCKERFILE)` | `$(params.dockerfile)` | `name: dockerfile`, `type: string`, `default: Dockerfile` |
| `$(build.builder.image)` | `$(params.builder-image)` | `name: builder-image`, `type: string`, no default |

### Strategy example

Excerpt adapted from [`samples/v1alpha1/buildstrategy/source-to-image/buildstrategy_source-to-image_cr.yaml`](../samples/v1alpha1/buildstrategy/source-to-image/buildstrategy_source-to-image_cr.yaml) and its [`v1beta1` counterpart](../samples/v1beta1/buildstrategy/source-to-image/buildstrategy_source-to-image_cr.yaml).

```yaml
# v1alpha1
---
apiVersion: shipwright.io/v1alpha1
kind: ClusterBuildStrategy
metadata:
  name: source-to-image
spec:
  volumes:
    - name: gen-source
      emptyDir: {}
  buildSteps:
    - name: s2i-build-as-dockerfile
      image: quay.io/openshift-pipeline/s2i:nightly
      imagePullPolicy: Always
      command:
        - /usr/local/bin/s2i
        - build
        - $(params.shp-source-context)
        - $(build.builder.image)
        - '--as-dockerfile'
        - /gen-source/Dockerfile.gen
      volumeMounts:
        - mountPath: /gen-source
          name: gen-source
      workingDir: $(params.shp-source-root)
```

```yaml
# v1beta1
---
apiVersion: shipwright.io/v1beta1
kind: ClusterBuildStrategy
metadata:
  name: source-to-image
spec:
  volumes:
    - name: gen-source
      emptyDir: {}
  parameters:
    - name: builder-image
      description: The builder image.
      type: string
  steps:
    - name: s2i-build-as-dockerfile
      image: quay.io/openshift-pipeline/s2i:nightly
      imagePullPolicy: Always
      command:
        - /usr/local/bin/s2i
        - build
        - $(params.shp-source-context)
        - $(params.builder-image)
        - '--as-dockerfile'
        - /gen-source/Dockerfile.gen
      volumeMounts:
        - mountPath: /gen-source
          name: gen-source
      workingDir: $(params.shp-source-root)
```

## Fields only available in v1beta1

These fields have no `v1alpha1` counterpart. They are listed here so you know what you gain by migrating; none of them is required.

| Resource | Field | Purpose |
|----------|-------|---------|
| `Build` | `.spec.source.git.depth` | Depth of the shallow clone. |
| `Build` | `.spec.retention.atBuildDeletion` | Replaces the `build.shipwright.io/build-run-deletion` annotation. |
| `Build` | `.spec.output.vulnerabilityScan` | Vulnerability scanning of the produced image. See [Defining the vulnerabilityScan](build.md#defining-the-vulnerabilityscan). |
| `Build` | `.spec.output.platforms[]` | Multi-platform builds assembled into an OCI image index. |
| `Build`, `BuildRun` | `.spec.nodeSelector`, `.spec.tolerations[]`, `.spec.schedulerName`, `.spec.runtimeClassName` | Pod scheduling and runtime controls. |
| `Build` | `.spec.strategy.stepResources[]` | Per-step resource overrides. See [Defining Step Resources](build.md#defining-step-resources). |
| `BuildRun` | `.spec.stepResources[]` | Per-step resource overrides, taking precedence over the `Build`. |
| `BuildRun` | `.status.executor` | Name **and** kind of the resource executing the `BuildRun`. |
| `BuildRun` | `.status.output.vulnerabilities[]` | Vulnerabilities detected in the produced image. |

## References

- [Introducing Shipwright Beta API](https://shipwright.io/blog/introducing-shipwright-beta-api/)
- [SHIP 0035: Beta API changes](https://github.com/shipwright-io/community/blob/main/ships/0035-beta-api-changes.md)
- [`Build`](build.md), [`BuildRun`](buildrun.md), [`BuildStrategy`](buildstrategies.md)
- [`v1alpha1` samples](../samples/v1alpha1) and [`v1beta1` samples](../samples/v1beta1)
