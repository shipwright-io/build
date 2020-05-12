# Build

- [Overview](#overview)
- [Build Controller](#build-controller)
- [Configuring a Build](#configuring-a-build)
  - [Defining the Source](#defining-the-source)
  - [Defining the Strategy](#defining-the-strategy)
  - [Defining the Builder or Dockerfile](#defining-the-builder-or-dockerfile)
  - [Defining Resources](#defining-resources)
  - [Defining the Output](#defining-the-output)

## Overview

A `Build` resource allows the user to define:

- source
- strategy
- builder
- dockerfile
- resources
- output

A `Build` is available within a namespace.

## Build Controller

The controller watches for:

- Updates on the `Build` resource (_CRD instance_)

When the controller reconciles it:

- Validates if the referenced `StrategyRef` exists.
- Validates if the container `registry` output secret exists.

## Configuring a Build

The `Build` definition supports the following fields:

- Required:
  - [`apiVersion`](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields) - Specifies the API version, for example `build.dev/v1alpha1`.
  - [`kind`](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields) - Specifies the Kind type, for example `Build`.
  - [`metadata`](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields) - Metadata that identify the CRD instance, for example the name of the `Build`.
  - `spec.source.URL` - Refers to the Git repository containing the source code.
  - `spec.strategy` - Refers to the `BuildStrategy` to be used, see the [examples](../samples/buildstrategy)
  - `spec.builder.image` - Refers to the image containing the build tools to build the source code. (_Use this path for Dockerless strategies_)
  - `spec.output`- Refers to the location where the generated image would be pushed.
  - `spec.output.credentials.name`- Reference an existing secret to get access to the container registry.

- Optional:
  - `spec.resources` - Refers to the compute [resources](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/) used on the container where the image is built.
  - `spec.parameters` - Refers to a list of `name-value` that could be used to loosely type parameters in the `BuildStrategy`.
  - `spec.dockerfile` - Path to a Dockerfile to be used for building an image. (_Use this path for strategies that require a Dockerfile_)

### Defining the Source

A `Build` resource can specify a Git source, together with other parameters like:

- `source.credentials.name` - For private repositories, the name is a reference to an existing secret on the same namespace containing the require `ssh` data.
- `source.revision` - An specific revision to select from the source repository, this can be a commit or branch name.
- `source.contextDir` - For repositories where the source code is not located at the root folder, you can specify this path here. Currently only supported by the `buildah` and `kaniko` build strategies.

Example of a `Build` with a source with **credentials** defined by the user.

```yaml
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: buildpack-nodejs-build
spec:
  source:
    url: https://github.com/sclorg/nodejs-ex
    credentials:
      name: source-repository-credentials
```

Example of a `Build` with a source that specifies an specific subfolder on the repository.

```yaml
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: buildah-custom-context-dockerfile
spec:
  source:
    url: https://github.com/SaschaSchwarze0/npm-simple
    contextDir: renamed
```

Example of a `Build` that specifies an specific branch on the git repository:

```yaml
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: buildah-golang-build
spec:
  source:
    url: https://github.com/sbose78/taxi
    revision: master
```

### Defining the Strategy

A `Build` resource can specify the `BuildStrategy` to use, these are:

- [Source-to-Image](samples/buildstrategy/source-to-image/README.md)
- [Buildpacks-v3](samples/buildstrategy/buildpacks-v3/README.md)
- [Buildah](samples/buildstrategy/buildah/README.md)
- [Kaniko](samples/buildstrategy/kaniko/README.md)

Defining the strategy is straightforward, you need to define the `name` and the `kind`. For example:

```yaml
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: buildpack-nodejs-build
spec:
  strategy:
    name: buildpacks-v3
    kind: ClusterBuildStrategy
```

### Defining the Builder or Dockerfile

A `Build` resource can specify an image containing the tools to build the final image. Users can do this via the `spec.builder` or the `spec.dockerfile`. For example, the user choose  the `Dockerfile` file under the source repository.

```yaml
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: buildah-golang-build
spec:
  source:
    url: https://github.com/sbose78/taxi
    revision: master
  strategy:
    name: buildah
    kind: ClusterBuildStrategy
  dockerfile: Dockerfile
```

Another example, when the user chooses to use a `builder` image:

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
  builder:
    image: gcr.io/paketo-buildpacks/builder:latest
```

### Defining Resources

A `Build` CRD instance can specify resources for the pod that builds the image, for example:

```yaml
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: kaniko-golang-build
spec:
  source:
    url: https://github.com/sbose78/taxi
  strategy:
    name: kaniko
    kind: ClusterBuildStrategy
  dockerfile: Dockerfile
  resources:
    limits:
      cpu: "500m"
      memory: "1Gi"
```

### Defining the Output

A `Build` resource can specify the output where the image should be pushed. For external private registries it is recommended to specify a secret with the related data to access it.

For example, the user specify a public registry:

```yaml
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: s2i-nodejs-build
spec:
  source:
    url: https://github.com/sclorg/nodejs-ex
  strategy:
    name: source-to-image
    kind: ClusterBuildStrategy
  builder:
    image: docker.io/centos/nodejs-10-centos7
  output:
    image: image-registry.openshift-image-registry.svc:5000/build-examples/nodejs-ex
```

Another example, is when the user specify a private registry:

```yaml
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: s2i-nodejs-build
spec:
  source:
    url: https://github.com/sclorg/nodejs-ex
  strategy:
    name: source-to-image
    kind: ClusterBuildStrategy
  builder:
    image: docker.io/centos/nodejs-10-centos7
  output:
    image: us.icr.io/source-to-image-build/nodejs-ex
    credentials:
      name: icr-knbuild
```
