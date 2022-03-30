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
  - [Defining ParamValues](#defining-paramvalues)
  - [Defining the Builder or Dockerfile](#defining-the-builder-or-dockerfile)
  - [Defining the Output](#defining-the-output)
- [BuildRun deletion](#BuildRun-deletion)

## Overview

A `Build` resource allows the user to define:

- source
- sources
- strategy
- params
- builder
- dockerfile
- output
- env

A `Build` is available within a namespace.

## Build Controller

The controller watches for:

- Updates on the `Build` resource (_CRD instance_)

When the controller reconciles it:

- Validates if the referenced `StrategyRef` exists.
- Validates if the specified `params` exists on the referenced strategy parameters. It also validates if the `params` names collide with the Shipwright reserved names.
- Validates if the container `registry` output secret exists.
- Validates if the referenced `spec.source.url` endpoint exists.

## Build Validations

In order to prevent users from triggering `BuildRuns` (_execution of a Build_) that will eventually fail because of wrong or missing dependencies or configuration settings, the Build controller will validate them in advance. If all validations are successful, users can expect a `Succeeded` `status.reason`, however if any of the validations failed, users can rely on the `status.reason` and `status.message` fields, in order to understand the root cause.

| Status.Reason | Description |
| --- | --- |
| BuildStrategyNotFound   | The referenced namespace-scope strategy doesn't exist. |
| ClusterBuildStrategyNotFound   | The referenced cluster-scope strategy doesn't exist. |
| SetOwnerReferenceFailed   | Setting ownerreferences between a Build and a BuildRun failed. This is triggered when making use of the `build.shipwright.io/build-run-deletion` annotation in a Build. |
| SpecSourceSecretRefNotFound | The secret used to authenticate to git doesn't exist. |
| SpecOutputSecretRefNotFound | The secret used to authenticate to the container registry doesn't exist. |
| SpecBuilderSecretRefNotFound | The secret used to authenticate to the container registry doesn't exist.|
| MultipleSecretRefNotFound | More than one secret is missing. At the moment, only three paths on a Build can specify a secret. |
| RestrictedParametersInUse | One or many defined `params` are colliding with Shipwright reserved parameters. See [Defining Params](#defining-params) for more information. |
| UndefinedParameter | One or many defined `params` are not defined in the referenced strategy. Please ensure that the strategy defines them under its `spec.parameters` list. |
| RemoteRepositoryUnreachable | The defined `spec.source.url` was not found. This validation only take place for http/https protocols. |
| BuildNameInvalid | The defined `Build` name (`metadata.name`) is invalid. The `Build` name should be a [valid label value](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set). |
| SpecEnvNameCanNotBeBlank | Indicates that the name for a user provided environment variable is blank. |
| SpecEnvValueCanNotBeBlank | Indicates that the value for a user provided environment variable is blank. |

## Configuring a Build

The `Build` definition supports the following fields:

- Required:
  - [`apiVersion`](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields) - Specifies the API version, for example `shipwright.io/v1alpha1`.
  - [`kind`](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields) - Specifies the Kind type, for example `Build`.
  - [`metadata`](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields) - Metadata that identify the CRD instance, for example the name of the `Build`.
  - `spec.source` - Refers to the location of the source code, for example a Git repository or source bundle image.
  - `spec.strategy` - Refers to the `BuildStrategy` to be used, see the [examples](../samples/buildstrategy)
  - `spec.builder.image` - Refers to the image containing the build tools to build the source code. (_Use this path for Dockerless strategies, this is just required for `source-to-image` buildStrategy_)
  - `spec.output`- Refers to the location where the generated image would be pushed.
  - `spec.output.credentials.name`- Reference an existing secret to get access to the container registry.

- Optional:
  - `spec.paramValues` - Refers to a name-value(s) list to specify values for `parameters` defined in the `BuildStrategy`.
  - `spec.dockerfile` - Path to a Dockerfile to be used for building an image. (_Use this path for strategies that require a Dockerfile_)
  - `spec.sources` - [Sources](#Sources) describes a slice of artifacts that will be imported into project context, before the actual build process starts.
  - `spec.timeout` - Defines a custom timeout. The value needs to be parsable by [ParseDuration](https://golang.org/pkg/time/#ParseDuration), for example `5m`. The default is ten minutes. The value can be overwritten in the `BuildRun`.
  - `metadata.annotations[build.shipwright.io/build-run-deletion]` - Defines if delete all related BuildRuns when deleting the Build. The default is `false`.
  - `spec.output.annotations` - Refers to a list of `key/value` that could be used to [annotate](https://github.com/opencontainers/image-spec/blob/main/annotations.md) the output image.
  - `spec.output.labels` - Refers to a list of `key/value` that could be used to label the output image.
  - `spec.env` - Specifies additional environment variables that should be passed to the build container. The available variables depend on the tool that is being used by the chosen build strategy.

### Defining the Source

A `Build` resource can specify a Git repository or bundle image source, together with other parameters like:

- `source.url` - Specify the source location using a Git repository.
- `source.bundleContainer.image` - Specify a source bundle container image to be used as the source.
- `source.bundleContainer.prune` - Configure whether the source bundle image should be deleted after the source was obtained (defaults to `Never`, other option is `AfterPull` to delete the image after a successful image pull).
- `source.credentials.name` - For private repositories/registries, the name is a reference to an existing secret on the same namespace containing the SSH private key, or Docker access credentials, respectively.
- `source.revision` - An specific revision to select from the source repository, this can be a commit, tag or branch name. If not defined, it will fallback to the Git repository default branch.
- `source.contextDir` - For repositories where the source code is not located at the root folder, you can specify this path here.

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

Example of a `Build` that specifies the tag `v.0.1.0` for the git repository:

```yaml
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: buildah-golang-build
spec:
  source:
    url: https://github.com/shipwright-io/sample-go
    contextDir: docker-build
    revision: v0.1.0
```

Example of a `Build` that specifies environment variables:

```yaml
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: buildah-golang-build
spec:
  source:
    url: https://github.com/shipwright-io/sample-go
    contextDir: docker-build
  env:
    - name: EXAMPLE_VAR_1
      value: "example-value-1"
    - name: EXAMPLE_VAR_2
      value: "example-value-2"
```

Example of a `Build` that uses the Kubernetes Downward API to
expose a `Pod` field as an environment variable:

```yaml
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: buildah-golang-build
spec:
  source:
    url: https://github.com/shipwright-io/sample-go
    contextDir: docker-build
  env:
    - name: POD_NAME
      valueFrom:
        fieldRef:
          fieldPath: metadata.name
```

Example of a `Build` that uses the Kubernetes Downward API to
expose a `Container` field as an environment variable:

```yaml
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: buildah-golang-build
spec:
  source:
    url: https://github.com/shipwright-io/sample-go
    contextDir: docker-build
  env:
    - name: MEMORY_LIMIT
      valueFrom:
        resourceFieldRef:
          containerName: my-container
          resource: limits.memory
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

### Defining ParamValues

A `Build` resource can specify _paramValues_ for parameters that are defined in the referenced `BuildStrategy`. This allows one to control how the steps of the build strategy behave. Values can be overwritten in the `BuildRun` resource. See the related [documentation](./buildrun.md#defining-params) for more information.

The build strategy author can define a parameter to be either a simple string, or an array. Depending on that, you must specify the value accordingly. The build strategy parameter can be specified with a default value. For parameters without a default, a value must be specified in the `Build` or `BuildRun`.

You can either specify values directly, or reference keys from [ConfigMaps](https://kubernetes.io/docs/concepts/configuration/configmap/) and [Secrets](https://kubernetes.io/docs/concepts/configuration/secret/). **Note**: the usage of ConfigMaps and Secrets is limited by the usage of the parameter in the build strategy steps. You can only use them if the parameter is used in the command, arguments, or as environment variable values.

When using _paramValues_, users should avoid:

- Defining a `spec.paramValues` name that doesn't match one of the `spec.parameters` defined in the `BuildStrategy`.
- Defining a `spec.paramValues` name that collides with the Shipwright reserved parameters. These are _BUILDER\_IMAGE_, _DOCKERFILE_, _CONTEXT\_DIR_ and any name starting with _shp-_.

In general, _paramValues_ are tightly bound to Strategy _parameters_, please make sure you understand the contents of your strategy of choice, before defining _paramValues_ in the _Build_.

#### Example

The [BuildKit sample `BuildStrategy`](../samples/buildstrategy/buildkit/buildstrategy_buildkit_cr.yaml) contains various parameters. Two of them are outlined here:

```yaml
apiVersion: shipwright.io/v1alpha1
kind: ClusterBuildStrategy
metadata:
  name: buildkit
  ...
spec:
  parameters:
  - name: build-args
    description: "The values for the ARGs in the Dockerfile. Values must be in the format KEY=VALUE."
    type: array
    defaults: []
  - name: cache
    description: "Configure BuildKit's cache usage. Allowed values are 'disabled' and 'registry'. The default is 'registry'."
    type: string
    default: registry
  ...
  buildSteps:
  ...
```

The `cache` parameter is a simple string. You can provide it like this in your Build:

```yaml
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: a-build
  namespace: a-namespace
spec:
  paramValues:
  - name: cache
    value: disabled
  strategy:
    name: buildkit
    kind: ClusterBuildStrategy
  source:
  ...
  output:
  ...
```

If you have multiple Builds and want to centrally control this parameter, then you can create a ConfigMap:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: buildkit-configuration
  namespace: a-namespace
data:
  cache: disabled
```

You reference the ConfigMap as a parameter value like this:

```yaml
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: a-build
  namespace: a-namespace
spec:
  paramValues:
  - name: cache
    configMapValue:
      name: buildkit-configuration
      key: cache
  strategy:
    name: buildkit
    kind: ClusterBuildStrategy
  source:
  ...
  output:
  ...
```

The `build-args` parameter is defined as an array. In the BuildKit strategy, it is used to set the values of [`ARG`s in the Dockerfile](https://docs.docker.com/engine/reference/builder/#arg), specified as key-value pairs separated by an equals sign, for example `NODE_VERSION=16`. Your Build then looks like this (the value for `cache` is retained to outline how multiple _paramValue_ can be set):

```yaml
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: a-build
  namespace: a-namespace
spec:
  paramValues:
  - name: cache
    configMapValue:
      name: buildkit-configuration
      key: cache
  - name: build-args
    values:
    - value: NODE_VERSION=16
  strategy:
    name: buildkit
    kind: ClusterBuildStrategy
  source:
  ...
  output:
  ...
```

Similar to simple values, you can also reference ConfigMaps and Secrets for every item in the array. Example:

```yaml
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: a-build
  namespace: a-namespace
spec:
  paramValues:
  - name: cache
    configMapValue:
      name: buildkit-configuration
      key: cache
  - name: build-args
    values:
    - configMapValue:
        name: project-configuration
        key: node-version
        format: NODE_VERSION=${CONFIGMAP_VALUE}
    - value: DEBUG_MODE=true
    - secretValue:
        name: npm-registry-access
        key: npm-auth-token
        format: NPM_AUTH_TOKEN=${SECRET_VALUE}
  strategy:
    name: buildkit
    kind: ClusterBuildStrategy
  source:
  ...
  output:
  ...
```

Here, we pass three items in the `build-args` array:

1. The first item references a ConfigMap. As the ConfigMap just contains the value (for example `"16"`) as the data of the `node-version` key, the `format` setting is used to prepend `NODE_VERSION=` to make it a complete key-value pair.
2. The second item is just a hard-coded value.
3. The third item references a Secret. This works in the same way as with ConfigMaps.

**NOTE**: the logging output of BuildKit contains expanded `ARG`s in `RUN` commands. Also, such information ends up in the final container image if you use such args in the [final stage of your Dockerfile](https://docs.docker.com/develop/develop-images/multistage-build/). An alternative approach to pass secrets is using [secret mounts](https://docs.docker.com/develop/develop-images/build_enhancements/#new-docker-build-secret-information). The BuildKit sample strategy supports them using the `secrets` parameter.

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

A `Build` resource can specify the output where the image should be pushed. For external private registries it is recommended to specify a secret with the related data to access it. There is an option available to specify the annotation and labels for the output image (annotations and labels mentioned here are specific to the container image and do not have any relation with the `Build` annotations).

**NOTE**: When you specify annotations or labels, the output image will get pushed twice. The first push comes from the build strategy. A follow-on update will then change the image configuration to add the annotations and labels. If you have automation in place based on push events in your container registry, be aware of this behavior.

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

Example of user specifies image annotations and labels:

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
    annotations:
      "org.opencontainers.image.source": "https://github.com/org/repo"
      "org.opencontainers.image.url": "https://my-company.com/images"
    labels:
      "maintainer": "team@my-company.com"
      "description": "This is my cool image"
```

Annotations added to the output image can be verified by running the command:

```sh
  docker manifest inspect us.icr.io/source-to-image-build/nodejs-ex | jq ".annotations"
```

Labels added to the output image can be verified by running the command (image should be available in host machine):

```sh
  docker inspect us.icr.io/source-to-image-build/nodejs-ex | jq ".[].Config.Labels"
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
