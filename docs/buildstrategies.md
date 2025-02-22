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
- [Multi-arch Native Buildah](#multi-arch-native-buildah)
  - [Installing Multi-arch Native Buildah Strategy](#installing-multi-arch-native-buildah-strategy) 
- [Buildpacks v3](#buildpacks-v3)
  - [Installing Buildpacks v3 Strategy](#installing-buildpacks-v3-strategy)
- [Kaniko](#kaniko)
  - [Installing Kaniko Strategy](#installing-kaniko-strategy)
- [BuildKit](#buildkit)
  - [Cache Exporters](#cache-exporters)
  - [Build-args and secrets](#build-args-and-secrets)
  - [Multi-platform builds](#multi-platform-builds)
  - [Known Limitations](#known-limitations)
  - [Usage in Clusters with Pod Security Standards](#usage-in-clusters-with-pod-security-standards)
  - [Installing BuildKit Strategy](#installing-buildkit-strategy)
- [ko](#ko)
  - [Installing ko Strategy](#installing-ko-strategy)
  - [Parameters](#parameters)
  - [Volumes](#volumes)
- [Source to Image](#source-to-image)
  - [Installing Source to Image Strategy](#installing-source-to-image-strategy)
  - [Build Steps](#build-steps)
- [Strategy parameters](#strategy-parameters)
- [System parameters](#system-parameters)
  - [Output directory vs. output image](#output-directory-vs-output-image)
- [System parameters vs Strategy Parameters Comparison](#system-parameters-vs-strategy-parameters-comparison)
- [Securely referencing string parameters](#securely-referencing-string-parameters)
- [System results](#system-results)
- [Security Contexts](#security-contexts)
- [Steps Resource Definition](#steps-resource-definition)
  - [Strategies with different resources](#strategies-with-different-resources)
  - [How does Tekton Pipelines handle resources](#how-does-tekton-pipelines-handle-resources)
  - [Examples of Tekton resources management](#examples-of-tekton-resources-management)
- [Annotations](#annotations)
- [Volumes and VolumeMounts](#volumes-and-volumemounts)

## Overview

There are two types of strategies, the `ClusterBuildStrategy` (`clusterbuildstrategies.shipwright.io/v1beta1`) and the `BuildStrategy` (`buildstrategies.shipwright.io/v1beta1`). Both strategies define a shared group of steps, needed to fullfil the application build.

A `ClusterBuildStrategy` is available cluster-wide, while a `BuildStrategy` is available within a namespace.

## Available ClusterBuildStrategies

Well-known strategies can be bootstrapped from [here](../samples/v1beta1/buildstrategy). The currently supported Cluster BuildStrategy are:

| Name                                                                                                              | Supported platforms |
|-------------------------------------------------------------------------------------------------------------------|---------------------|
| [buildah](../samples/v1beta1/buildstrategy/buildah)                                                               | all                 |
| [multiarch-native-buildah](../samples/v1beta1/buildstrategy/multiarch-native-buildah)                             | all                 |
| [BuildKit](../samples/v1beta1/buildstrategy/buildkit/buildstrategy_buildkit_cr.yaml)                              | all                 |
| [buildpacks-v3-heroku](../samples/v1beta1/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3-heroku_cr.yaml) | linux/amd64 only    |
| [buildpacks-v3](../samples/v1beta1/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_cr.yaml)               | linux/amd64 only    |
| [kaniko](../samples/v1beta1/buildstrategy/kaniko/buildstrategy_kaniko_cr.yaml)                                    | all                 |
| [ko](../samples/v1beta1/buildstrategy/ko/buildstrategy_ko_cr.yaml)                                                | all                 |
| [source-to-image](../samples/v1beta1/buildstrategy/source-to-image/buildstrategy_source-to-image_cr.yaml)         | linux/amd64 only    |

## Available BuildStrategies

The current supported namespaces BuildStrategy are:

| Name                                                                                                                         | Supported platforms |
|------------------------------------------------------------------------------------------------------------------------------|---------------------|
| [buildpacks-v3-heroku](../samples/v1beta1/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3-heroku_namespaced_cr.yaml) | linux/amd64 only    |
| [buildpacks-v3](../samples/v1beta1/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_namespaced_cr.yaml)               | linux/amd64 only    |

---

## Buildah

The `buildah` ClusterBuildStrategy uses [`buildah`](https://github.com/containers/buildah) to build and push a container image, out of a `Dockerfile`. The `Dockerfile` should be specified on the `Build` resource.

The strategy is available in two formats:

- [`buildah-shipwright-managed-push`](../samples/v1beta1/buildstrategy/buildah/buildstrategy_buildah_shipwright_managed_push_cr.yaml)
- [`buildah-strategy-managed-push`](../samples/v1beta1/buildstrategy/buildah/buildstrategy_buildah_strategy_managed_push_cr.yaml)

Learn more about the differences of [shipwright-, or strategy-managed push](#output-directory-vs-output-image)

### Installing Buildah Strategy

To install use:

```sh
kubectl apply -f samples/v1beta1/buildstrategy/buildah/buildstrategy_buildah_shipwright_managed_push_cr.yaml
kubectl apply -f samples/v1beta1/buildstrategy/buildah/buildstrategy_buildah_strategy_managed_push_cr.yaml
```

---

## Multi-arch Native buildah

The [`multiarch-native-buildah` ClusterBuildStrategy](https://github.com/shipwright-io/build/blob/17d516a160/samples/v1beta1/buildstrategy/multiarch-native-buildah/buildstrategy_multiarch_native_buildah_cr.yaml) uses [`buildah`](https://github.com/containers/buildah) 
to build and push a container image, out of a `Dockerfile`. 

The strategy will build the image for the platforms that are listed in the `architectures` parameter of a `Build` object
that refers it.
The strategy will require the cluster to have the necessary infrastructure to run the builds: worker nodes
for each architecture that is listed in the `architectures` parameter. 

The ClusterBuildStrategy runs a main orchestrator pod. 
The orchestrator pod will create one auxiliary job for each architecture requested by the Build. 
The auxiliary jobs are responsible for building the container image and 
coordinate with the orchestrator pod.

When all the builds are completed, the orchestrator pod will compose a manifest-list image and push it to the target registry.

The service account that runs the strategy must be bound to a ClusterRole able to `create`, `list`, `get` and `watch` `batch/v1` `jobs` and `core/v1` `pods` resources. 
The ClusteRole also needs to allow the `create` verb for the `pods/exec` resource.
Finally, when running in OKD or OpenShift clusters, the service account must be able to use the 
`privileged` SecurityContextConstraint.

### Installing Multi-arch Native buildah Strategy

To install the cluster-scoped strategy, use:

```sh
kubectl apply -f samples/v1beta1/buildstrategy/multiarch-native-buildah/buildstrategy_multiarch_native_buildah_cr.yaml
```

For each namespace where you want to use the strategy, you also need to apply the RBAC rules that allow the service
account to run the strategy. If the service account is named `pipeline` (default), you can use:

```sh
kubectl apply -n <namespace> -f  samples/v1beta1/buildstrategy/multiarch-native-buildah/
```

### Parameters

The build strategy provides the following parameters that you can set in a Build or BuildRun to control its behavior:

| Parameter             | Description                                                                                                                                                                                                                                              | Default      |
|-----------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------------|
| `architectures`       | The list of architectures to build the image for                                                                                                                                                                                                         | \[ "amd64" ] |
| `build-args`          | The values for the args in the Dockerfile. Values must be in the format KEY=VALUE.                                                                                                                                                                       | empty array  |
| `dockerfile`          | The path to the Dockerfile to be used for building the image.                                                                                                                                                                                            | `Dockerfile` |
| `from`                | Image name used to replace the value in the first FROM instruction in the Dockerfile.                                                                                                                                                                    | empty string |
| `runtime-stage-from`  | Image name used to replace the value in the last FROM instruction in the Dockerfile.                                                                                                                                                                     | empty string |
| `build-contexts`      | Specify an additional build context using its short name and its location. Additional build contexts can be referenced in the same manner as we access different stages in COPY instruction. Use values in the form "name=value". See man buildah-build. | empty array  |
| `registries-block`    | A list of registries to block. Images from these registries will not be pulled during the build.                                                                                                                                                         | empty array  |
| `registries-insecure` | A list of registries that are insecure. Images from these registries will be pulled without verifying the TLS certificate.                                                                                                                               | empty array  |
| `registries-search`   | A list of registries to search for short name images. Images missing the fully-qualified name of a registry will be looked up in these registries.                                                                                                       | empty array  |
| `request-cpu`         | The CPU request to set for the auxiliary jobs.                                                                                                                                                                                                           | `250m`       |
| `request-memory`      | The memory request to set for the auxiliary jobs.                                                                                                                                                                                                        | `64Mi`       |
| `limit-cpu`           | The CPU limit to set for the auxiliary jobs.                                                                                                                                                                                                             | no limit     |
| `limit-memory`        | The memory limit to set for the auxiliary jobs.                                                                                                                                                                                                          | `2Gi`        |

### Volumes

| Volume              | Description                                                                                                                                                                                                                                                                                                                                |
|---------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| oci-archive-storage | Volume to contain the temporary single-arch images manifests in OCI format. It can be set to a persistent volume, e.g., for large images. The default is an emptyDir volume which means that the cached data is discarded at the end of a BuildRun and will make use of ephemeral storage (according to the cluster infrastructure setup). |

### Example build

```yaml
---
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: multiarch-native-buildah-ex
spec:
  source:
    type: Git
    git:
      url: https://github.com/shipwright-io/sample-go
    contextDir: docker-build
  strategy:
    name: multiarch-native-buildah
    kind: ClusterBuildStrategy
  paramValues:
    - name: architectures
      values:
        # This will require a cluster with both arm64 and amd64 nodes
        - value: "amd64"
        - value: "arm64"
    - name: build-contexts
      values:
        - value: "ghcr.io/shipwright-io/shipwright-samples/golang:1.18=docker://ghcr.io/shipwright-io/shipwright-samples/golang:1.18"
    # The buildah `--from` replaces the first FROM statement
    - name: from
      value: "" # Using the build-contexts for this example
    # The runtime-stage-from implements the logic to replace the last stage FROM image of a Dockerfile
    - name: runtime-stage-from
      value: docker://gcr.io/distroless/static:nonroot
    - name: dockerfile
      value: Dockerfile
  output:
    image: image-registry.openshift-image-registry.svc:5000/build-examples/taxi-app

```

---

## Buildpacks v3

The [buildpacks-v3][buildpacks] BuildStrategy/ClusterBuildStrategy uses a Cloud Native Builder ([CNB][cnb]) container image, and is able to implement [lifecycle commands][lifecycle].

### Installing Buildpacks v3 Strategy

You can install the `BuildStrategy` in your namespace or install the `ClusterBuildStrategy` at cluster scope so that it can be shared across namespaces.

To install the cluster scope strategy, you can choose between the Paketo and Heroku buildpacks family:

```sh
# Paketo
kubectl apply -f samples/v1beta1/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_cr.yaml

# Heroku
kubectl apply -f samples/v1beta1/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3-heroku_cr.yaml
```

To install the namespaced scope strategy, you can choose between the Paketo and Heroku buildpacks family:

```sh
# Paketo
kubectl apply -f samples/v1beta1/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_namespaced_cr.yaml

# Heroku
kubectl apply -f samples/v1beta1/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3-heroku_namespaced_cr.yaml
```

---

## Kaniko

The `kaniko` ClusterBuildStrategy is composed by Kaniko's `executor` [kaniko], with the objective of building a container-image, out of a `Dockerfile` and context directory.

### Installing Kaniko Strategy

To install the cluster scope strategy, use:

```sh
kubectl apply -f samples/v1beta1/buildstrategy/kaniko/buildstrategy_kaniko_cr.yaml
```

---

## BuildKit

[BuildKit](https://github.com/moby/buildkit) is composed of the `buildctl` client and the `buildkitd` daemon. For the `buildkit` ClusterBuildStrategy, it runs on a [daemonless](https://github.com/moby/buildkit#daemonless) mode, where both client and ephemeral daemon run in a single container. In addition, it runs without privileges (_[rootless](https://github.com/moby/buildkit/blob/master/docs/rootless.md)_).

### Cache Exporters

By default, the `buildkit` ClusterBuildStrategy will use caching to optimize the build times. When pushing an image to a registry, it will use the inline export cache, which appends cache information to the image that is built. Please refer to [export-cache docs](https://github.com/moby/buildkit#export-cache) for more information. Caching can be disabled by setting the `cache` parameter to `"disabled"`. See [Defining ParamValues](build.md#defining-paramvalues) for more information.

### Build-args and secrets

The sample build strategy contains array parameters to set values for [`ARG`s in your Dockerfile](https://docs.docker.com/engine/reference/builder/#arg), and for [mounts with type=secret](https://docs.docker.com/develop/develop-images/build_enhancements/#new-docker-build-secret-information). The parameter names are `build-args` and `secrets`. [Defining ParamValues](build.md#defining-paramvalues) contains example usage.

### Multi-platform builds

The sample build strategy contains a `platforms` array parameter that you can set to leverage [BuildKit's support to build multi-platform images](https://github.com/moby/buildkit/blob/master/docs/multi-platform.md). If you do not set this value, the image is built for the platform that is supported by the `FROM` image. If that image supports multiple platforms, then the image will be built for the platform of your Kubernetes node.

### Known Limitations

The `buildkit` ClusterBuildStrategy currently locks the following parameters:

- To allow running rootless, it requires both [AppArmor](https://kubernetes.io/docs/tutorials/clusters/apparmor/) and [SecComp](https://kubernetes.io/docs/tutorials/clusters/seccomp/) to be disabled using the `unconfined` profile.

### Usage in Clusters with Pod Security Standards

The BuildKit strategy contains fields with regard to security settings. It therefore depends on the respective cluster setup and administrative configuration. These settings are:

- Defining the `unconfined` profile for both AppArmor and seccomp as required by the underlying `rootlesskit`.
- The `allowPrivilegeEscalation` settings is set to `true` to be able to use binaries that have the `setuid` bit set in order to run with "root" level privileges. In case of BuildKit, this is required by `rootlesskit` in order to set the user namespace mapping file `/proc/<pid>/uid_map`.
- Use of non-root user with UID 1000/GID 1000 as the `runAsUser`.

These settings have no effect in case Pod Security Standards are not used.

_Please note:_ At this point in time, there is no way to run `rootlesskit` to start the BuildKit daemon without the `allowPrivilegeEscalation` flag set to `true`. Clusters with the `Restricted` security standard in place will not be able to use this build strategy.

### Installing BuildKit Strategy

To install the cluster scope strategy, use:

```sh
kubectl apply -f samples/v1beta1/buildstrategy/buildkit/buildstrategy_buildkit_cr.yaml
```

---

## ko

The `ko` ClusterBuilderStrategy is using [ko](https://github.com/ko-build/ko)'s publish command to build an image from a Golang main package.

### Installing ko Strategy

To install the cluster scope strategy, use:

```sh
kubectl apply -f samples/v1beta1/buildstrategy/ko/buildstrategy_ko_cr.yaml
```

### Parameters

The build strategy provides the following parameters that you can set in a Build or BuildRun to control its behavior:

| Parameter           | Description                                                                                                                                                                                                                                                                                        | Default   |
|---------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-----------|
| `go-flags`          | Value for the GOFLAGS environment variable.                                                                                                                                                                                                                                                        | Empty     |
| `go-version`        | Version of Go, must match a tag from [the golang image](https://hub.docker.com/_/golang?tab=tags)                                                                                                                                                                                                  | `1.23`    |
| `ko-version`        | Version of ko, must be either `latest` for the newest release, or a [ko release name](https://github.com/ko-build/ko/releases)                                                                                                                                                                     | `latest`  |
| `package-directory` | The directory inside the context directory containing the main package.                                                                                                                                                                                                                            | `.`       |
| `target-platform`   | Target platform to be built. For example: `linux/arm64`. Multiple platforms can be provided separated by comma, for example: `linux/arm64,linux/amd64`. The value `all` will build all platforms supported by the base image. The value `current` will build the platform on which the build runs. | `current` |

### Volumes

| Volume  | Description                                                                                                                                                                                                                  |
|---------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| gocache | Volume to contain the GOCACHE. Can be set to a persistent volume to optimize compilation performance for rebuilds. The default is an emptyDir volume which means that the cached data is discarded at the end of a BuildRun. |

## Source to Image

This BuildStrategy is composed by [`source-to-image`][s2i] and [`kaniko`][kaniko] in order to generate a `Dockerfile` and prepare the application to be built later on with a builder.

`s2i` requires a specially crafted image, which can be informed as `builderImage` parameter on the `Build` resource.

### Installing Source to Image Strategy

To install the cluster scope strategy use:

```sh
kubectl apply -f samples/v1beta1/buildstrategy/source-to-image/buildstrategy_source-to-image_cr.yaml
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

## Strategy parameters

Strategy parameters allow users to parameterize their strategy definition, by allowing users to control the _parameters_ values via the `Build` or `BuildRun` resources.

Users defining _parameters_ under their strategies require to understand the following:

- **Definition**: A list of parameters should be defined under `spec.parameters`. Each list item should consist of a _name_, a _description_, a _type_ (either `"array"` or `"string"`) and optionally a _default_ value (for type=string), or _defaults_ values (for type=array). If no default(s) are provided, then the user must define a value in the Build or BuildRun.
- **Usage**: In order to use a parameter in the strategy steps, use the following syntax for type=string: `$(params.your-parameter-name)`. String parameters can be used in all places in the `buildSteps`. Some example scenarios are:
  - `image`: to use a custom tag, for example `golang:$(params.go-version)` as it is done in the [ko sample build strategy](../samples/v1beta1/buildstrategy/ko/buildstrategy_ko_cr.yaml)
  - `args`: to pass data into your builder command
  - `env`: to force a user to provide a value for an environment variable.
  
  Arrays are referenced using `$(params.your-array-parameter-name[*])`, and can only be used in as the value for `args` or `command` because the defined as arrays by Kubernetes. For every item in the array, an arg will be set. For example, if you specify this in your build strategy step:

  ```yaml
  spec:
    parameters:
      - name: tool-args
        description: Parameters for the tool
        type: array
    steps:
      - name: a-step
        command:
          - some-tool
        args:
          - $(params.tool-args[*])
  ```

  If the build user sets the value of tool-args to ["--some-arg", "some-value"], then the Pod will contain these args:

  ```yaml
  spec:
    containers:
      - name: a-step
        args:
        ...
          - --some-arg
          - some-value
  ```

- **Parameterize**: Any `Build` or `BuildRun` referencing your strategy, can set a value for _your-parameter-name_ parameter if needed.

**Note**: Users can provide parameter values as simple strings or as references to keys in [ConfigMaps](https://kubernetes.io/docs/concepts/configuration/configmap/) and [Secrets](https://kubernetes.io/docs/concepts/configuration/secret/). If they use a ConfigMap or Secret, then the value can only be used if the parameter is used in the `command`, `args`, or `env` section of the `buildSteps`. For example, the above-mentioned scenario to set a step's `image` to `golang:$(params.go-version)` does not allow the usage of ConfigMaps or Secrets.

The following example is from the [BuildKit sample build strategy](../samples/v1beta1/buildstrategy/buildkit/buildstrategy_buildkit_cr.yaml). It defines and uses several parameters:

```yaml
---
apiVersion: shipwright.io/v1beta1
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
  - name: insecure-registry
    type: string
    description: "enables the push to an insecure registry"
    default: "false"
  - name: secrets
    description: "The secrets to pass to the build. Values must be in the format ID=FILE_CONTENT."
    type: array
    defaults: []
  - name: dockerfile
    description: The path to the Dockerfile to be used for building the image.
    type: string
    default: "Dockerfile"
  steps:
    ...
    - name: build-and-push
      image: moby/buildkit:v0.20.0-rootless
      imagePullPolicy: Always
      workingDir: $(params.shp-source-root)
      ...
      command:
        - /bin/ash
      args:
        - -c
        - |
          set -euo pipefail

          # Prepare the file arguments
          DOCKERFILE_PATH='$(params.shp-source-context)/$(params.dockerfile)'
          DOCKERFILE_DIR="$(dirname "${DOCKERFILE_PATH}")"
          DOCKERFILE_NAME="$(basename "${DOCKERFILE_PATH}")"

          # We only have ash here and therefore no bash arrays to help add dynamic arguments (the build-args) to the build command.

          echo "#!/bin/ash" > /tmp/run.sh
          echo "set -euo pipefail" >> /tmp/run.sh
          echo "buildctl-daemonless.sh \\" >> /tmp/run.sh
          echo "build \\" >> /tmp/run.sh
          echo "--progress=plain \\" >> /tmp/run.sh
          echo "--frontend=dockerfile.v0 \\" >> /tmp/run.sh
          echo "--opt=filename=\"${DOCKERFILE_NAME}\" \\" >> /tmp/run.sh
          echo "--local=context='$(params.shp-source-context)' \\" >> /tmp/run.sh
          echo "--local=dockerfile=\"${DOCKERFILE_DIR}\" \\" >> /tmp/run.sh
          echo "--output=type=image,name='$(params.shp-output-image)',push=true,registry.insecure=$(params.insecure-registry) \\" >> /tmp/run.sh
          if [ "$(params.cache)" == "registry" ]; then
            echo "--export-cache=type=inline \\" >> /tmp/run.sh
            echo "--import-cache=type=registry,ref='$(params.shp-output-image)' \\" >> /tmp/run.sh
          elif [ "$(params.cache)" == "disabled" ]; then
            echo "--no-cache \\" >> /tmp/run.sh
          else
            echo -e "An invalid value for the parameter 'cache' has been provided: '$(params.cache)'. Allowed values are 'disabled' and 'registry'."
            echo -n "InvalidParameterValue" > '$(results.shp-error-reason.path)'
            echo -n "An invalid value for the parameter 'cache' has been provided: '$(params.cache)'. Allowed values are 'disabled' and 'registry'." > '$(results.shp-error-message.path)'
            exit 1
          fi

          stage=""
          for a in "$@"
          do
            if [ "${a}" == "--build-args" ]; then
              stage=build-args
            elif [ "${a}" == "--secrets" ]; then
              stage=secrets
            elif [ "${stage}" == "build-args" ]; then
              echo "--opt=\"build-arg:${a}\" \\" >> /tmp/run.sh
            elif [ "${stage}" == "secrets" ]; then
              # Split ID=FILE_CONTENT into variables id and data

              # using head because the data could be multiline
              id="$(echo "${a}" | head -1 | sed 's/=.*//')"

              # This is hacky, we remove the suffix ${id}= from all lines of the data.
              # If the data would be multiple lines and a line would start with ${id}=
              # then we would remove it. We could force users to give us the secret
              # base64 encoded. But ultimately, the best solution might be if the user
              # mounts the secret and just gives us the path here.
              data="$(echo "${a}" | sed "s/^${id}=//")"

              # Write the secret data into a temporary file, once we have volume support
              # in the build strategy, we should use a memory based emptyDir for this.
              echo -n "${data}" > "/tmp/secret_${id}"

              # Add the secret argument
              echo "--secret id=${id},src="/tmp/secret_${id}" \\" >> /tmp/run.sh
            fi
          done

          echo "--metadata-file /tmp/image-metadata.json" >> /tmp/run.sh

          chmod +x /tmp/run.sh
          /tmp/run.sh

          # Store the image digest
          sed -E 's/.*containerimage.digest":"([^"]*).*/\1/' < /tmp/image-metadata.json > '$(results.shp-image-digest.path)'
        # That's the separator between the shell script and its args
        - --
        - --build-args
        - $(params.build-args[*])
        - --secrets
        - $(params.secrets[*])
```

See more information on how to use these parameters in a `Build` or `BuildRun` in the related [documentation](./build.md#defining-paramvalues).

## System parameters

Contrary to the strategy `spec.parameters`, you can use system parameters and their values defined at runtime when defining the steps of a build strategy to access system information as well as information provided by the user in their Build or BuildRun. The following parameters are available:

| Parameter                        | Description                                                                                                                                                                                                                                                                                                                                                                                           |
|----------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `$(params.shp-source-root)`      | The absolute path to the directory that contains the user's sources.                                                                                                                                                                                                                                                                                                                                  |
| `$(params.shp-source-context)`   | The absolute path to the context directory of the user's sources. If the user specified no value for `spec.source.contextDir` in their `Build`, then this value will equal the value for `$(params.shp-source-root)`. Note that this directory is not guaranteed to exist at the time the container for your step is started, you can therefore not use this parameter as a step's working directory. |
| `$(params.shp-output-directory)` | The absolute path to a directory that the build strategy should store the image in. You can store a single tarball containing a single image, or an OCI image layout.                                                                                                                                                                                                                                 |
| `$(params.shp-output-image)`     | The URL of the image that the user wants to push, as specified in the Build's `spec.output.image` or as an override from the BuildRun's `spec.output.image`.                                                                                                                                                                                                                                          |
| `$(params.shp-output-insecure)`  | A flag that indicates the output image's registry location is insecure because it uses a certificate not signed by a certificate authority, or uses HTTP.                                                                                                                                                                                                                                             |

### Output directory vs. output image

As a build strategy author, you decide whether your build strategy or Shipwright pushes the build image to the container registry:

- If you DO NOT use `$(params.shp-output-directory)`, then Shipwright assumes that your build strategy PUSHES the image. We call this a strategy-managed push.
- If you DO use `$(params.shp-output-directory)`, then Shipwright assumes that your build strategy does NOT PUSH the image. We call this a shipwright-managed push.

When you use the `$(params.shp-output-directory)` parameter, then Shipwright will also set the [image-related system results](#system-results).

If you are uncertain about how to implement your build strategy, then follow this guidance:

1. If your build strategy tool cannot locally store an image but always pushes it, then you must do the push operation. An example is the [Buildpacks strategy](../samples/v1beta1/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_cr.yaml). You SHOULD respect the `$(params.shp-output-insecure)` parameter.
2. If your build strategy tool can locally store an image, then the choice depends on how you expect your build users to make use of your strategy, and the nature of your strategy.
   1. Some build strategies do not produce all layers of an image, but use a common base image and put one or more layers on top with the application. An example is `ko`. Such base image layers are often already present in the destination registry (like in rebuilds). If the strategy can perform the push operation, then it can optimize the process and can omit the download of the base image when it is not required to push it. In the case of a shipwright-managed push, the complete image must be locally stored in `$(params.shp-output-directory)`, which implies that a base image must always be downloaded.
   2. Some build strategy tools do not make it easy to determine the digest or size of the image, which can make it complex for you to set the [strategy results](#system-results). In the case of a shipwright-managed push, Shipwright has the responsibility to set them.
   3. Build users can configure the build to amend additional annotations, or labels to the final image. In the case of a shipwright-managed push, these can be set directly and the image will only be pushed once. In a strategy-managed push scenario, your build strategy will push the first version of the image without those annotations and labels. Shipwright will then mutate the image and push it again with the updated annotations and labels. Such a duplicate push can cause unexpected behavior with registries that trigger other actions when an image gets pushed, or that do not allow overwriting a tag.
   4. The Shipwright maintainers plan to provide more capabilities in the future that need the image locally, such as vulnerability scanning, or software bill of material (SBOM) creation. These capabilities may be only fully supported with shipwright-managed push.

## System parameters vs Strategy Parameters Comparison

| Parameter Type     | User Configurable | Definition                                          |
|--------------------|-------------------|-----------------------------------------------------|
| System Parameter   | No                | At run-time, by the `BuildRun` controller.          |
| Strategy Parameter | Yes               | At build-time, during the `BuildStrategy` creation. |

## Securely referencing string parameters

In build strategy steps, string parameters are referenced using `$(params.PARAM_NAME)`. This applies to system parameters, and those parameters defined in the build strategy. You can reference those parameters at many locations in the build steps, such as environment variables values, arguments, image, and more. In the Pod, all `$(params.PARAM_NAME)` tokens will be replaced by simple string replaces. This is safe in most locations but requires your attention when you define an inline script using an argument. For example:

```yaml
spec:
  parameters:
    - name: sample-parameter
      description: A sample parameter
      type: string
  steps:
    - name: sample-step
      command:
        - /bin/bash
      args:
        - -c
        - |
          set -euo pipefail

          some-tool --sample-argument "$(params.sample-parameter)"
```

This opens the door to script injection, for example if the user sets the `sample-parameter` to `argument-value" && malicious-command && echo "`, the resulting pod argument will look like this:

```yaml
        - |
          set -euo pipefail

          some-tool --sample-argument "argument-value" && malicious-command && echo ""
```

To securely pass a parameter value into a script-style argument, you can choose between these two approaches:

1. Using environment variables. This is used in some of our sample strategies, for example [ko](../samples/v1beta1/buildstrategy/ko/buildstrategy_ko_cr.yaml), or [buildpacks](../samples/v1beta1/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_cr.yaml). Basically, instead of directly using the parameter inside the script, you pass it via environment variable. Using quoting, shells ensure that no command injection is possible:

   ```yaml
   spec:
     parameters:
       - name: sample-parameter
         description: A sample parameter
         type: string
     steps:
       - name: sample-step
         env:
           - name: PARAM_SAMPLE_PARAMETER
             value: $(params.sample-parameter)
         command:
           - /bin/bash
         args:
           - -c
           - |
             set -euo pipefail

             some-tool --sample-argument "${PARAM_SAMPLE_PARAMETER}"
   ```

2. Using arguments. This is used in some of our sample build strategies, for example [buildah](../samples/v1beta1/buildstrategy/buildah/buildstrategy_buildah_shipwright_managed_push_cr.yaml). Here, you use arguments to your own inline script. Appropriate shell quoting guards against command injection.

   ```yaml
   spec:
     parameters:
       - name: sample-parameter
         description: A sample parameter
         type: string
     steps:
       - name: sample-step
         command:
           - /bin/bash
         args:
           - -c
           - |
             set -euo pipefail

             SAMPLE_PARAMETER="$1"

             some-tool --sample-argument "${SAMPLE_PARAMETER}"
           - --
           - $(params.sample-parameter)
   ```

## System results

If you are using a strategy-managed push, see [output directory vs output image](#output-directory-vs-output-image), you can optionally store the size and digest of the image your build strategy created to a set of files.

| Result file                        | Description                                     |
|------------------------------------|-------------------------------------------------|
| `$(results.shp-image-digest.path)` | File to store the digest of the image.          |
| `$(results.shp-image-size.path)`   | File to store the compressed size of the image. |

You can look at sample build strategies, such as [Buildpacks](../samples/v1beta1/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_cr.yaml), to see how they fill some or all of the results files.

This information will be available in the `.status.output` section of the BuildRun.

```yaml
apiVersion: shipwright.io/v1beta1
kind: BuildRun
# [...]
status:
 # [...]
  output:
    digest: sha256:07626e3c7fdd28d5328a8d6df8d29cd3da760c7f5e2070b534f9b880ed093a53
    size: 1989004
  # [...]
```

Additionally, you can store error details for debugging purposes when a BuildRun fails using your strategy.

| Result file                         | Description                      |
|-------------------------------------|----------------------------------|
| `$(results.shp-error-reason.path)`  | File to store the error reason.  |
| `$(results.shp-error-message.path)` | File to store the error message. |

Reason is intended to be a one-word CamelCase classification of the error source, with the first letter capitalized.
Error details are only propagated if the build container terminates with a non-zero exit code.
This information will be available in the `.status.failureDetails` section of the BuildRun.

```yaml
apiVersion: shipwright.io/v1beta1
kind: BuildRun
# [...]
status:
  # [...]
  failureDetails:
    location:
      container: step-source-default
      pod: baran-build-buildrun-gzmv5-b7wbf-pod-bbpqr
    message: The source repository does not exist, or you have insufficient permission
      to access it.
    reason: GitRemotePrivate
```

## Security Contexts

In a build strategy, it is recommended that you define a `securityContext` with a runAsUser and runAsGroup:

```yaml
spec:
  securityContext:
    runAsUser: 1000
    runAsGroup: 1000
```

This runAs configuration will be used for all shipwright-managed steps such as the step that retrieves the source code, and for the steps you define in the build strategy. This configuration ensures that all steps share the same runAs configuration which eliminates file permission problems.

Without a `securityContext` for the build strategy, shipwright-managed steps will run with the `runAsUser` and `runAsGroup` that is defined in the [configuration's container templates](configuration.md) that is potentially a different user than you use in your build strategy. This can result in issues when for example source code is downloaded as user A as defined by the Git container template, but your strategy accesses it as user B.

In build strategy steps you can define a step-specific `securityContext` that matches [Kubernetes' security context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/) where you can configure other security aspects such as capabilities or privileged containers.

## Steps Resource Definition

All strategies steps can include a definition of resources(_limits and requests_) for CPU, memory and disk. For strategies with more than one step, each step(_container_) could require more resources than others. Strategy admins are free to define the values that they consider the best fit for each step. Also, identical strategies with the same steps that are only different in their name and step resources can be installed on the cluster to allow users to create a build with smaller and larger resource requirements.

### Strategies with different resources

If the strategy admins required to have multiple flavours of the same strategy, where one strategy has more resources that the other. Then, multiple strategies for the same type should be defined on the cluster. In the following example, we use Kaniko as the type:

```yaml
---
apiVersion: shipwright.io/v1beta1
kind: ClusterBuildStrategy
metadata:
  name: kaniko-small
spec:
  steps:
    - name: build-and-push
      image: gcr.io/kaniko-project/executor:v1.23.2
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
        - --dockerfile=$(params.dockerfile)
        - --context=$(params.shp-source-context)
        - --destination=$(params.shp-output-image)
        - --snapshot-mode=redo
        - --push-retry=3
      resources:
        limits:
          cpu: 250m
          memory: 65Mi
        requests:
          cpu: 250m
          memory: 65Mi
  parameters:
  - name: dockerfile
    description: The path to the Dockerfile to be used for building the image.
    type: string
    default: "Dockerfile"
---
apiVersion: shipwright.io/v1beta1
kind: ClusterBuildStrategy
metadata:
  name: kaniko-medium
spec:
  steps:
    - name: build-and-push
      image: gcr.io/kaniko-project/executor:v1.23.2
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
        - --dockerfile=$(params.dockerfile)
        - --context=$(params.shp-source-context)
        - --destination=$(params.shp-output-image)
        - --snapshot-mode=redo
        - --push-retry=3
      resources:
        limits:
          cpu: 500m
          memory: 1Gi
        requests:
          cpu: 500m
          memory: 1Gi
  parameters:
  - name: dockerfile
    description: The path to the Dockerfile to be used for building the image.
    type: string
    default: "Dockerfile"
```

The above provides more control and flexibility for the strategy admins. For `end-users`, all they need to do, is to reference the proper strategy. For example:

```yaml
---
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: kaniko-medium
spec:
  source:
    git:  
      url: https://github.com/shipwright-io/sample-go
    contextDir: docker-build
  strategy:
    name: kaniko
    kind: ClusterBuildStrategy
  paramValues:
  - name: dockerfile
    value: Dockerfile
```

### How does Tekton Pipelines handle resources

The **Build** controller relies on the Tekton [pipeline controller](https://github.com/tektoncd/pipeline) to schedule the `pods` that execute the above strategy steps. In a nutshell, the **Build** controller creates on run-time a Tekton **TaskRun**, and the **TaskRun** generates a new pod in the particular namespace. In order to build an image, the pod executes all the strategy steps one-by-one.

Tekton manage each step resources **request** in a very particular way, see the [docs](https://github.com/tektoncd/pipeline/blob/main/docs/tasks.md#defining-steps). From this document, it mentions the following:

> The CPU, memory, and ephemeral storage resource requests will be set to zero, or, if specified, the minimums set through LimitRanges in that Namespace, if the container image does not have the largest resource request out of all container images in the Task. This ensures that the Pod that executes the Task only requests enough resources to run a single container image in the Task rather than hoard resources for all container images in the Task at once.

### Examples of Tekton resources management

For a more concrete example, let´s take a look on the following scenarios:

---

**Scenario 1.**  Namespace without `LimitRange`, both steps with the same resource values.

If we apply the following resources:

- [buildahBuild](../samples/v1beta1/build/build_buildah_shipwright_managed_push_cr.yaml)
- [buildahBuildRun](../samples/v1beta1/buildrun/buildrun_buildah_cr.yaml)
- [buildahClusterBuildStrategy](../samples/v1beta1/buildstrategy/buildah/buildstrategy_buildah_shipwright_managed_push_cr.yaml)

We will see some differences between the `TaskRun` definition and the `pod` definition.

For the `TaskRun`, as expected we can see the resources on each `step`, as we previously define on our [strategy](../samples/v1beta1/buildstrategy/buildah/buildstrategy_buildah_shipwright_managed_push_cr.yaml).

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

If we apply the following resources:

- [buildahBuild](../samples/v1beta1/build/build_buildah_shipwright_managed_push_cr.yaml)
- [buildahBuildRun](../samples/v1beta1/buildrun/buildrun_buildah_cr.yaml)
- We will use a modified buildah strategy, with the following steps resources:

  ```yaml
    - name: buildah-bud
      image: quay.io/containers/buildah:v1.38.1
      workingDir: $(params.shp-source-root)
      securityContext:
        privileged: true
      command:
        - /usr/bin/buildah
      args:
        - bud
        - --tag=$(params.shp-output-image)
        - --file=$(params.dockerfile)
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
      image: quay.io/containers/buildah:v1.38.1
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

## Volumes and VolumeMounts

Build Strategies can declare `volumes`. These `volumes` can be referred to by the build steps using `volumeMount`.
Volumes in Build Strategy follow the declaration of [Pod Volumes](https://kubernetes.io/docs/concepts/storage/volumes/), so 
all the usual `volumeSource` types are supported.

Volumes can be overridden by `Build`s and `BuildRun`s, so Build Strategies' volumes support an `overridable` flag, which
is a boolean, and is `false` by default. In case volume is not overridable, `Build` or `BuildRun` that tries to override it,
will fail.

Build steps can declare a `volumeMount`, which allows them to access volumes defined by `BuildStrategy`, `Build` or `BuildRun`.

Here is an example of `BuildStrategy` object that defines `volumes` and `volumeMount`s:
```
apiVersion: shipwright.io/v1beta1
kind: BuildStrategy
metadata:
  name: buildah
spec:
  steps:
    - name: build
      image: quay.io/containers/buildah:v1.27.0
      workingDir: $(params.shp-source-root)
      command:
        - buildah
        - bud
        - --tls-verify=false
        - --layers
        - -f
        - $(params.dockerfile)
        - -t
        - $(params.shp-output-image)
        - $(params.shp-source-context)
      volumeMounts:
        - name: varlibcontainers
          mountPath: /var/lib/containers
  volumes:
    - name: varlibcontainers
      overridable: true
      emptyDir: {}
  # ...
```
