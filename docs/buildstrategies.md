<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

# BuildStrategies

- [Overview](#overview)
- [Available ClusterBuildStrategies](#available-clusterbuildstrategies)
- [Available BuildStrategies](#available-buildstrategies)
- [Buildah](#buildah)
  - [Installing Buildah Strategy](#installing-buildah-strategy)
- [Buildpacks v3](#buildpacks-v3)
  - [Installing Buildpacks v3 Strategy](#installing-buildpacks-v3-strategy)
  - [Try it](#try-it)
- [Kaniko](#kaniko)
  - [Installing Kaniko Strategy](#installing-kaniko-strategy)
- [ko](#ko)
  - [Installing ko Strategy](#installing-ko-strategy)
- [Source to Image](#source-to-image)
  - [Installing Source to Image Strategy](#installing-source-to-image-strategy)
  - [Build Steps](#build-steps)
- [Steps Resource Definition](#steps-resource-definition)
  - [Strategies with different resources](#strategies-with-different-resources)
  - [How does Tekton Pipelines handle resources](#how-does-tekton-pipelines-handle-resources)
  - [Examples of Tekton resources management](#examples-of-tekton-resources-management)
- [Annotations](#annotations)

## Overview

There are two types of strategies, the `ClusterBuildStrategy` (`clusterbuildstrategies.shipwright.io/v1alpha1`) and the `BuildStrategy` (`buildstrategies.shipwright.io/v1alpha1`). Both strategies define a shared group of steps, needed to fullfil the application build.

A `ClusterBuildStrategy` is available cluster-wide, while a `BuildStrategy` is available within a namespace.

## Available ClusterBuildStrategies

Well-known strategies can be boostrapped from [here](../samples/buildstrategy). The current supported Cluster BuildStrategy are:

- [buildah](../samples/buildstrategy/buildah/buildstrategy_buildah_cr.yaml)
- [buildpacks-v3-heroku](../samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3-heroku_cr.yaml)
- [buildpacks-v3](../samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_cr.yaml)
- [kaniko](../samples/buildstrategy/kaniko/buildstrategy_kaniko_cr.yaml)
- [source-to-image](../samples/buildstrategy/source-to-image/buildstrategy_source-to-image_cr.yaml)

## Available BuildStrategies

The current supported namespaces BuildStrategy are:

- [buildpacks-v3-heroku](../samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3-heroku_namespaced_cr.yaml)
- [buildpacks-v3](../samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_namespaced_cr.yaml)

---

## Buildah

The `buildah` ClusterBuildStrategy consists of using [`buildah`](https://github.com/containers/buildah) to build and push a container image, out of a `Dockerfile`. The `Dockerfile` should be specified on the `Build` resource. Also, instead of the `spec.dockerfile`, the `spec.builderImage` can be used with `quay.io/buildah/stable` as the value when defining the `Build` resource.

### Installing Buildah Strategy

To install use:

```sh
kubectl apply -f samples/buildstrategy/buildah/buildstrategy_buildah_cr.yaml
```

---

## Buildpacks v3

The [buildpacks-v3][buildpacks] BuildStrategy/ClusterBuildStrategy uses a Cloud Native Builder ([CNB][cnb]) container image, and is able to implement [lifecycle commands][lifecycle]. The following CNB images are the most common options:

- [`heroku/buildpacks:18`][hubheroku]
- [`cloudfoundry/cnb:bionic`][hubcloudfoundry]
- [`docker.io/paketobuildpacks/builder:full`](https://hub.docker.com/r/paketobuildpacks/builder/tags)

### Installing Buildpacks v3 Strategy

You can install the `BuildStrategy` in your namespace or install the `ClusterBuildStrategy` at cluster scope so that it can be shared across namespaces.

To install the cluster scope strategy, use (below is a heroku example, you can also use paketo sample):

```sh
kubectl apply -f samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3-heroku_cr.yaml
```

To install the namespaced scope strategy, use:

```sh
kubectl apply -f samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3-heroku_namespaced_cr.yaml
```

### Try it

To use this strategy follow this steps:

- Create the Kubernetes secret that host the configuration to access the container registry.

- Create a `Build` resource that uses `quay.io` or `DockerHub` image repository for pushing the image. Also, provide credentials to access it.

  ```yaml
  apiVersion: shipwright.io/v1alpha1
  kind: Build
  metadata:
    name: buildpack-nodejs-build
  spec:
    source:
      url: https://github.com/sclorg/nodejs-ex
    strategy:
      name: buildpacks-v3
      kind: ClusterBuildStrategy
    output:
      image: quay.io/yourorg/yourrepo
      credentials: <your-kubernetes-container-registry-secret>
  ```

- Start a `BuildRun` resource.

  ```yaml
  apiVersion: shipwright.io/v1alpha1
  kind: BuildRun
  metadata:
    name: buildpack-nodejs-buildrun
  spec:
    buildRef:
      name: buildpack-nodejs-build
  ```

---

## Kaniko

The `kaniko` ClusterBuildStrategy is composed by Kaniko's `executor` [kaniko], with the objective of building a container-image, out of a `Dockerfile` and context directory.

### Installing Kaniko Strategy

To install the cluster scope strategy, use:

```sh
kubectl apply -f samples/buildstrategy/kaniko/buildstrategy_kaniko_cr.yaml
```

---

## ko

The `ko` ClusterBuilderStrategy is using [ko](https://github.com/google/ko)'s publish command to build an image from a Golang main package.

### Installing ko Strategy

To install the cluster scope strategy, use:

```sh
kubectl apply -f samples/buildstrategy/ko/buildstrategy_ko_cr.yaml
```

**Note**: The build strategy currently uses the `spec.contextDir` of the Build in a different way than this property is designed for: the Git repository must be a Go module with the go.mod file at the root. The `contextDir` specifies the path to the main package. You can check the [example](../samples/build/build_ko_cr.yaml) which is set up to build the Shipwright Build controller. This behavior will eventually be corrected once [Exhaustive list of generalized Build API/CRD attributes #184](https://github.com/shipwright-io/build/issues/184) / [Custom attributes from the Build CR could be used as parameters while defining a BuildStrategy #537](https://github.com/shipwright-io/build/issues/537) are done.

## Source to Image

This BuildStrategy is composed by [`source-to-image`][s2i] and [`kaniko`][kaniko] in order to generate a `Dockerfile` and prepare the application to be built later on with a builder.

`s2i` requires a specially crafted image, which can be informed as `builderImage` parameter on the `Build` resource.

### Installing Source to Image Strategy

To install the cluster scope strategy use:

```sh
kubectl apply -f samples/buildstrategy/source-to-image/buildstrategy_source-to-image_cr.yaml
```

### Build Steps

1. `s2i` in order to generate a `Dockerfile` and prepare source-code for image build;
2. `kaniko` to create and push the container image to what is defined as `output.image`;

[buildpacks]: https://buildpacks.io/
[cnb]: https://buildpacks.io/docs/concepts/components/builder/
[lifecycle]: https://buildpacks.io/docs/concepts/components/lifecycle/
[hubheroku]: https://hub.docker.com/r/heroku/buildpacks/
[hubcloudfoundry]: https://hub.docker.com/r/cloudfoundry/cnb
[kaniko]: https://github.com/GoogleContainerTools/kaniko
[s2i]: https://github.com/openshift/source-to-image
[buildah]: https://github.com/containers/buildah

## Steps Resource Definition

All strategies steps can include a definition of resources(_limits and requests_) for CPU, memory and disk. For strategies with more than one step, each step(_container_) could require more resources than others. Strategy admins are free to define the values that they consider the best fit for each step. Also, identical strategies with the same steps that are only different in their name and step resources can be installed on the cluster to allow users to create a build with smaller and larger resource requirements.

### Strategies with different resources

If the strategy admins would require to have multiple flavours of the same strategy, where one strategy has more resources that the other. Then, multiple strategies for the same type should be defined on the cluster. In the following example, we use Kaniko as the type:

```yaml
---
apiVersion: shipwright.io/v1alpha1
kind: ClusterBuildStrategy
metadata:
  name: kaniko-small
spec:
  buildSteps:
    - name: build-and-push
      image: gcr.io/kaniko-project/executor:v1.5.1
      workingDir: /workspace/source
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
        - --context=/workspace/source/$(build.source.contextDir)
        - --destination=$(build.output.image)
        - --oci-layout-path=/workspace/output/image
        - --snapshotMode=redo
        - --push-retry=3
      resources:
        limits:
          cpu: 250m
          memory: 65Mi
        requests:
          cpu: 250m
          memory: 65Mi
---
apiVersion: shipwright.io/v1alpha1
kind: ClusterBuildStrategy
metadata:
  name: kaniko-medium
spec:
  buildSteps:
    - name: build-and-push
      image: gcr.io/kaniko-project/executor:v1.5.1
      workingDir: /workspace/source
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
        - --context=/workspace/source/$(build.source.contextDir)
        - --destination=$(build.output.image)
        - --oci-layout-path=/workspace/output/image
        - --snapshotMode=redo
        - --push-retry=3
      resources:
        limits:
          cpu: 500m
          memory: 1Gi
        requests:
          cpu: 500m
          memory: 1Gi
```

The above provides more control and flexibility for the strategy admins. For `end-users`, all they need to do, is to reference the proper strategy. For example:

```yaml
---
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: kaniko-medium
spec:
  source:
    url: https://github.com/shipwright-io/sample-go
    contextDir: docker-build
  strategy:
    name: kaniko
    kind: ClusterBuildStrategy
  dockerfile: Dockerfile
```

### How does Tekton Pipelines handle resources

The **Build** controller relies on the Tekton [pipeline controller](https://github.com/tektoncd/pipeline) to schedule the `pods` that execute the above strategy steps. In a nutshell, the **Build** controller creates on run-time a Tekton **TaskRun**, and the **TaskRun** generates a new pod in the particular namespace. In order to build an image, the pod executes all the strategy steps one-by-one.

Tekton manage each step resources **request** in a very particular way, see the [docs](https://github.com/tektoncd/pipeline/blob/master/docs/tasks.md#defining-steps). From this document, it mentions the following:

> The CPU, memory, and ephemeral storage resource requests will be set to zero, or, if specified, the minimums set through LimitRanges in that Namespace, if the container image does not have the largest resource request out of all container images in the Task. This ensures that the Pod that executes the Task only requests enough resources to run a single container image in the Task rather than hoard resources for all container images in the Task at once.

### Examples of Tekton resources management

For a more concrete example, let´s take a look on the following scenarios:

---

**Scenario 1.**  Namespace without `LimitRange`, both steps with the same resource values.

If we will apply the following resources:

- [buildahBuild](../samples/build/buildah/build_buildah_cr.yaml)
- [buildahBuildRun](../samples/buildrun/buildah/buildrun_buildah_cr.yaml)
- [buildahClusterBuildStrategy](../samples/buildstrategy/buildah/buildstrategy_buildah_cr.yaml)

We will see some differences between the `TaskRun` definition and the `pod` definition.

For the `TaskRun`, as expected we can see the resources on each `step`, as we previously define on our [strategy](../samples/buildstrategy/buildah/buildstrategy_buildah_cr.yaml).

```sh
$ kubectl -n test-build get tr buildah-golang-buildrun-9gmcx-pod-lhzbc -o json | jq '.spec.taskSpec.steps[] | select(.name == "step-buildah-bud" ) | .resources'
{
  "limits": {
    "cpu": "500m",
    "memory": "1Gi"
  },
  "requests": {
    "cpu": "250m",
    "memory": "65Mi"
  }
}

$ kubectl -n test-build get tr buildah-golang-buildrun-9gmcx-pod-lhzbc -o json | jq '.spec.taskSpec.steps[] | select(.name == "step-buildah-push" ) | .resources'
{
  "limits": {
    "cpu": "500m",
    "memory": "1Gi"
  },
  "requests": {
    "cpu": "250m",
    "memory": "65Mi"
  }
}
```

The pod definition is different, while Tekton will only use the **highest** values of one container, and set the rest(lowest) to zero:

```sh
$ kubectl -n test-build get pods buildah-golang-buildrun-9gmcx-pod-lhzbc -o json | jq '.spec.containers[] | select(.name == "step-step-buildah-bud" ) | .resources'
{
  "limits": {
    "cpu": "500m",
    "memory": "1Gi"
  },
  "requests": {
    "cpu": "250m",
    "ephemeral-storage": "0",
    "memory": "65Mi"
  }
}

$ kubectl -n test-build get pods buildah-golang-buildrun-9gmcx-pod-lhzbc -o json | jq '.spec.containers[] | select(.name == "step-step-buildah-push" ) | .resources'
{
  "limits": {
    "cpu": "500m",
    "memory": "1Gi"
  },
  "requests": {
    "cpu": "0",               <------------------- See how the request is set to ZERO.
    "ephemeral-storage": "0", <------------------- See how the request is set to ZERO.
    "memory": "0"             <------------------- See how the request is set to ZERO.
  }
}
```

In this scenario, only one container can have the `spec.resources.requests` definition. Even when both steps have the same values, only one container will get them, the others will be set to zero.

---

**Scenario 2.**  Namespace without `LimitRange`, steps with different resources:

If we will apply the following resources:

- [buildahBuild](../samples/build/buildah/build_buildah_cr.yaml)
- [buildahBuildRun](../samples/buildrun/buildah/buildrun_buildah_cr.yaml)
- We will use a modified buildah strategy, with the following steps resources:

  ```yaml
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
          memory: 100Mi  <------ See how we provide more memory to step-buildah-push, compared to the 65Mi of the other step
  ```

For the `TaskRun`, as expected we can see the resources on each `step`.

```sh
$ kubectl -n test-build get tr buildah-golang-buildrun-skgrp -o json | jq '.spec.taskSpec.steps[] | select(.name == "step-buildah-bud" ) | .resources'
{
  "limits": {
    "cpu": "500m",
    "memory": "1Gi"
  },
  "requests": {
    "cpu": "250m",
    "memory": "65Mi"
  }
}

$ kubectl -n test-build get tr buildah-golang-buildrun-skgrp -o json | jq '.spec.taskSpec.steps[] | select(.name == "step-buildah-push" ) | .resources'
{
  "limits": {
    "cpu": "500m",
    "memory": "1Gi"
  },
  "requests": {
    "cpu": "250m",
    "memory": "100Mi"
  }
}
```

The pod definition is different, while Tekton will only use the **highest** values of one container, and set the rest(lowest) to zero:

```sh
$ kubectl -n test-build get pods buildah-golang-buildrun-95xq8-pod-mww8d -o json | jq '.spec.containers[] | select(.name == "step-step-buildah-bud" ) | .resources'
{
  "limits": {
    "cpu": "500m",
    "memory": "1Gi"
  },
  "requests": {
    "cpu": "250m",                <------------------- See how the CPU is preserved
    "ephemeral-storage": "0",
    "memory": "0"                 <------------------- See how the memory is set to ZERO
  }
}
$ kubectl -n test-build get pods buildah-golang-buildrun-95xq8-pod-mww8d -o json | jq '.spec.containers[] | select(.name == "step-step-buildah-push" ) | .resources'
{
  "limits": {
    "cpu": "500m",
    "memory": "1Gi"
  },
  "requests": {
    "cpu": "0",                     <------------------- See how the CPU is set to zero.
    "ephemeral-storage": "0",
    "memory": "100Mi"               <------------------- See how the memory is preserved on this container
  }
}
```

In the above scenario, we can see how the maximum numbers for resource requests are distributed between containers. The container `step-buildah-push` gets the `100mi` for the memory requests, while it was the one defining the highest number. At the same time, the container `step-buildah-bud` is assigned a `0` for its memory request.

---

**Scenario 3.**  Namespace **with** a `LimitRange`.

When a `LimitRange` exists on the namespace, `Tekton Pipeline` controller will do the same approach as stated in the above two scenarios. The difference is that for the containers that have lower values, instead of zero, they will get the `minimum values of the LimitRange`.

## Annotations

Annotations can be defined for a BuildStrategy/ClusterBuildStrategy as for any other Kubernetes object. Annotations are propagated to the TaskRun and from there, Tekton propagates them to the Pod. Use cases for this are for example:

- The Kubernetes [Network Traffic Shaping](https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/network-plugins/#support-traffic-shaping) feature looks for the `kubernetes.io/ingress-bandwidth` and `kubernetes.io/egress-bandwidth` annotations to limit the network bandwidth the `Pod` is allowed to use.
- The [AppArmor profile of a container](https://kubernetes.io/docs/tutorials/clusters/apparmor/) is defined using the `container.apparmor.security.beta.kubernetes.io/<container_name>` annotation.

The following annotations are not propagated:

- `kubectl.kubernetes.io/last-applied-configuration`
- `clusterbuildstrategy.shipwright.io/*`
- `buildstrategy.shipwright.io/*`
- `build.shipwright.io/*`
- `buildrun.shipwright.io/*`

A Kubernetes administrator can further restrict the usage of annotations by using policy engines like [Open Policy Agent](https://www.openpolicyagent.org/).
