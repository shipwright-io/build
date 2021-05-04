<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

# Build

- [Overview](#overview)
- [Build Controller](#build-controller)
- [Build Validations](#build-validations)
- [Configuring a Build](#configuring-a-build)
  - [Defining the Source](#defining-the-source)
  - [Defining the Strategy](#defining-the-strategy)
  - [Defining the Builder or Dockerfile](#defining-the-builder-or-dockerfile)
  - [Defining the Output](#defining-the-output)
  - [Runtime-Image](#Runtime-Image)
- [BuildRun deletion](#BuildRun-deletion)

## Overview

A `Build` resource allows the user to define:

- source
- sources
- strategy
- builder
- dockerfile
- output

A `Build` is available within a namespace.

## Build Controller

The controller watches for:

- Updates on the `Build` resource (_CRD instance_)

When the controller reconciles it:

- Validates if the referenced `StrategyRef` exists.
- Validates if the container `registry` output secret exists.
- Validates if the referenced `spec.source.url` endpoint exists.

## Build Validations

In order to prevent users from triggering `BuildRuns` (_execution of a Build_) that will eventually fail because of wrong or missing dependencies or configuration settings, the Build controller will validate them in advance. If all validations are successful, users can expect a `Succeeded` `Status.Reason`, however if any of the validations failed, users can rely on the `Status.Reason` and `Status.Message` fields, in order to understand the root cause.

| Status.Reason | Description |
| --- | --- |
| BuildStrategyNotFound   | The referenced namespace-scope strategy doesn't exist. |
| ClusterBuildStrategyNotFound   | The referenced cluster-scope strategy doesn't exist. |
| SetOwnerReferenceFailed   | Setting ownerreferences between a Build and a BuildRun failed. This is triggered when making use of the `build.shipwright.io/build-run-deletion` annotation in a Build. |
| SpecSourceSecretRefNotFound | The secret used to authenticate to git doesn't exist. |
| SpecOutputSecretRefNotFound | The secret used to authenticate to the container registry doesn't exist. |
| SpecBuilderSecretRefNotFound | The secret used to authenticate to the container registry doesn't exist.|
| MultipleSecretRefNotFound | More than one secret is missing. At the moment, only three paths on a Build can specify a secret. |
| RuntimePathsCanNotBeEmpty | The Runtime feature is used, but the runtime path was not defined. This is mandatory. |
| RemoteRepositoryUnreachable | The defined `spec.source.url` was not found. This validation only take place for http/https protocols. |

## Configuring a Build

The `Build` definition supports the following fields:

- Required:
  - [`apiVersion`](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields) - Specifies the API version, for example `shipwright.io/v1alpha1`.
  - [`kind`](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields) - Specifies the Kind type, for example `Build`.
  - [`metadata`](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields) - Metadata that identify the CRD instance, for example the name of the `Build`.
  - `spec.source.URL` - Refers to the Git repository containing the source code.
  - `spec.strategy` - Refers to the `BuildStrategy` to be used, see the [examples](../samples/buildstrategy)
  - `spec.builder.image` - Refers to the image containing the build tools to build the source code. (_Use this path for Dockerless strategies, this is just required for `source-to-image` buildStrategy_)
  - `spec.output`- Refers to the location where the generated image would be pushed.
  - `spec.output.credentials.name`- Reference an existing secret to get access to the container registry.

- Optional:
  - `spec.parameters` - Refers to a list of `name-value` that could be used to loosely type parameters in the `BuildStrategy`.
  - `spec.dockerfile` - Path to a Dockerfile to be used for building an image. (_Use this path for strategies that require a Dockerfile_)
  - `spec.sources` - [Sources](#Sources) describes a slice of artifacts that will be imported into project context, before the actual build process starts.
  - `spec.runtime` - Runtime-Image settings, to be used for a multi-stage build.
  - `spec.timeout` - Defines a custom timeout. The value needs to be parsable by [ParseDuration](https://golang.org/pkg/time/#ParseDuration), for example `5m`. The default is ten minutes. The value can be overwritten in the `BuildRun`.
  - `metadata.annotations[build.shipwright.io/build-run-deletion]` - Defines if delete all related BuildRuns when deleting the Build. The default is `false`.

### Defining the Source

A `Build` resource can specify a Git source, together with other parameters like:

- `source.credentials.name` - For private repositories, the name is a reference to an existing secret on the same namespace containing the `ssh` data.
- `source.revision` - An specific revision to select from the source repository, this can be a commit or branch name. If not defined, it will fallback to the git repository default branch.
- `source.contextDir` - For repositories where the source code is not located at the root folder, you can specify this path here. Currently, only supported by `buildah`, `kaniko` and `buildpacks` build strategies.

By default, the Build controller won't validate that the Git repository exists. If the validation is desired, users can define the `build.shipwright.io/verify.repository` annotation with `true` explicitly. For example:

Example of a `Build` with the **build.shipwright.io/verify.repository** annotation, in order to enable the `spec.source.url` validation.

```yaml
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: buildah-golang-build
  annotations:
    build.shipwright.io/verify.repository: "true"
spec:
  source:
    url: https://github.com/shipwright-io/sample-go
    contextDir: docker-build
```

_Note_: The Build controller only validates two scenarios. The first one where the endpoint uses an `http/https` protocol, the second one when a `ssh` protocol (_e.g. `git@`_) is defined and none referenced secret was provided(_e.g. source.credentials.name_).

Example of a `Build` with a source with **credentials** defined by the user.

```yaml
apiVersion: shipwright.io/v1alpha1
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
apiVersion: shipwright.io/v1alpha1
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
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: buildah-golang-build
spec:
  source:
    url: https://github.com/shipwright-io/sample-go
    contextDir: docker-build
```

### Defining the Strategy

A `Build` resource can specify the `BuildStrategy` to use, these are:

- [Buildah](buildstrategies.md#buildah)
- [Buildpacks-v3](buildstrategies.md#buildpacks-v3)
- [BuildKit](buildstrategies.md#buildkit)
- [Kaniko](buildstrategies.md#kaniko)
- [ko](buildstrategies.md#ko)
- [Source-to-Image](buildstrategies.md#source-to-image)

Defining the strategy is straightforward, you need to define the `name` and the `kind`. For example:

```yaml
apiVersion: shipwright.io/v1alpha1
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
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: buildah-golang-build
spec:
  source:
    url: https://github.com/shipwright-io/sample-go
    contextDir: docker-build
  strategy:
    name: buildah
    kind: ClusterBuildStrategy
  dockerfile: Dockerfile
```

Another example, when the user chooses to use a `builder` image ( This is required for `source-to-image` buildStrategy, because for different code languages, they have different builders. ):

```yaml
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: s2i-nodejs-build
spec:
  source:
    url: https://github.com/shipwright-io/sample-nodejs
    contextDir: source-build/
  strategy:
    name: source-to-image
    kind: ClusterBuildStrategy
  builder:
    image: docker.io/centos/nodejs-10-centos7
```

### Defining the Output

A `Build` resource can specify the output where the image should be pushed. For external private registries it is recommended to specify a secret with the related data to access it.

For example, the user specify a public registry:

```yaml
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: s2i-nodejs-build
spec:
  source:
    url: https://github.com/shipwright-io/sample-nodejs
    contextDir: source-build/
  strategy:
    name: source-to-image
    kind: ClusterBuildStrategy
  builder:
    image: docker.io/centos/nodejs-10-centos7
  output:
    image: image-registry.openshift-image-registry.svc:5000/build-examples/nodejs-ex
```

Another example, is when the user specifies a private registry:

```yaml
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: s2i-nodejs-build
spec:
  source:
    url: https://github.com/shipwright-io/sample-nodejs
    contextDir: source-build/
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

### Sources

Represents remote artifacts, as in external entities that will be added to the build context before the actual build starts. Therefore, you may employ `.spec.sources` to download artifacts from external repositories.

```yaml
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: nodejs-ex
spec:
  sources:
    - name: project-logo
      url: https://gist.github.com/project/image.png
```

Under `.spec.sources` we have the following attributes:

- `.name`: represents the name of resource, required attribute.
- `.url`: universal resource location (URL), required attribute.

When downloading artifacts the process is executed in the same directory where the application source-code is located, by default `/workspace/source`.

Additionally, we have plan to keep evolving `.spec.sources` by adding more types of remote data declaration, this API field works as an extension point to support external and internal resource locations.

At this initial stage, authentication is not supported therefore you can only download from sources without this mechanism in place.

### Runtime-Image

Runtime-image is a new image composed with build-strategy outcome. On which you can compose a multi-stage image build, copying parts out the original image into a new one. This feature allows replacing the base-image of any container-image, creating leaner images, and other use-cases.

The following examples illustrates how to the `runtime`:

```yml
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: nodejs-ex-runtime
spec:
  strategy:
    name: buildpacks-v3
    kind: ClusterBuildStrategy
  source:
    url: https://github.com/sclorg/nodejs-ex.git
  output:
    image: image-registry.openshift-image-registry.svc:5000/build-examples/nodejs-ex
  runtime:
    base:
      image: docker.io/node:latest
    workDir: /home/node/app
    run:
      - echo "Before copying data..."
    user:
      name: node
      group: "1000"
    paths:
      - $(workspace):/home/node/app
    entrypoint:
      - npm
      - start
```

This build will produce a Node.js based application where a single directory is imported from the image built by buildpacks strategy. The data copied is using the `.spec.runtime.user` directive, and the image also runs based on it.

Please consider the description of the attributes under `.spec.runtime`:

- `.base`: specifies the runtime base-image to be used, using Image as type
- `.workDir`: path to WORKDIR in runtime-image
- `.env`: runtime-image additional environment variables, key-value
- `.labels`: runtime-image additional labels, key-value
- `.run`: arbitrary commands to be executed as `RUN` blocks, before `COPY`
- `.user.name`: username employed on `USER` directive, and also to change ownership of files copied to the runtime-image
- `.user.group`: group name (or GID), employed to change ownership and on `USER` directive
- `.paths`: list of files or directory paths to be copied to runtime-image, those can be defined as `<source>:<destination>` split by colon (`:`). You can use the `$(workspace)` placeholder to access the directory where your source repository is cloned, if `spec.source.contextDir` is defined, then `$(workspace)` to context directory location
- `.entrypoint`: entrypoint command, specified as a list

> ⚠️ **Image Tag Overwrite**
>
> Specifying the runtime section will cause a `BuildRun` to push `spec.output.image` twice. First, the image produced by chosen `BuildStrategy` is pushed, and next it gets reused to construct the runtime-image, which is pushed again, overwriting `BuildStrategy` outcome.
> Be aware, specially in situations where the image push action triggers automation steps. Since the same tag will be reused, you might need to take this in consideration when using runtime-images.

Under the cover, the runtime image will be an additional step in the generated Task spec of the TaskRun. It uses [Kaniko](https://github.com/GoogleContainerTools/kaniko) to run a container build using the `gcr.io/kaniko-project/executor:v1.6.0` image. You can overwrite this image by adding the environment variable `KANIKO_CONTAINER_IMAGE` to the [build controller deployment](../deploy/controller.yaml).

## BuildRun deletion

A `Build` can automatically delete a related `BuildRun`. To enable this feature set the  `build.shipwright.io/build-run-deletion` annotation to `true` in the `Build` instance. By default the annotation is never present in a `Build` definition. See an example of how to define this annotation:

```yaml
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: kaniko-golang-build
  annotations:
    build.shipwright.io/build-run-deletion: "true"
```
