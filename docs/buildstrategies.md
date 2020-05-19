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
- [Source to Image](#source-to-image)
  - [Installing Source to Image Strategy](#installing-source-to-image-strategy)
  - [Build Steps](#build-steps)

## Overview

There are two types of strategies, the `ClusterBuildStrategy` (`clusterbuildstrategies.build.dev/v1alpha1`) and the `BuildStrategy` (`buildstrategies.build.dev/v1alpha1`). Both strategies define a shared group of steps, needed to fullfil the application build.

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
- [`gcr.io/paketo-buildpacks/builder:latest`](https://console.cloud.google.com/gcr/images/paketo-buildpacks/GLOBAL/builder?gcrImageListsize=30)

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
  apiVersion: build.dev/v1alpha1
  kind: Build
  metadata:
    name: buildpack-nodejs-build
  spec:
    source:
      url: https://github.com/sclorg/nodejs-ex
    strategy:
      name: buildpacks-v3
      kind: ClusterBuildStrategy
    builderImage: gcr.io/paketo-buildpacks/builder:latest
    output:
      image: quay.io/yourorg/yourrepo
      credentials: <your-kubernetes-container-registry-secret>
  ```

- Start a `BuildRun` resource.

  ```yaml
  apiVersion: build.dev/v1alpha1
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

## Source to Image

This BuildStrategy is composed by [`source-to-image`][s2i] and [`buildah`][buildah] in order to generate a `Dockerfile` and prepare the application to be built later on with a builder. Typically `s2i` requires a specially crafted image, which can be
informed as `builderImage` parameter on the `Build` resource.

### Installing Source to Image Strategy

To install the ClusterBuildStratey use:

```sh
kubectl apply -f samples/buildstrategy/source-to-image/buildstrategy_source-to-image_cr.yaml
```

### Build Steps

1. `s2i` in order to generate a `Dockerfile` and prepare source-code for image build;
2. `buildah` to create the container image;
3. `buildah` to push container-image on `output.image` parameter;

[buildpacks]: https://buildpacks.io/
[cnb]: https://buildpacks.io/docs/concepts/components/builder/
[lifecycle]: https://buildpacks.io/docs/concepts/components/lifecycle/
[hubheroku]: https://hub.docker.com/r/heroku/buildpacks/
[hubcloudfoundry]: https://hub.docker.com/r/cloudfoundry/cnb
[kaniko]: https://github.com/GoogleContainerTools/kaniko
[s2i]: https://github.com/openshift/source-to-image
[buildah]: https://github.com/containers/buildah