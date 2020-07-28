<p align="center">
    <img alt="Work in Progress" src="https://img.shields.io/badge/Status-Work%20in%20Progress-informational">
    <a alt="GoReport" href="https://goreportcard.com/report/github.com/redhat-developer/build">
        <img src="https://goreportcard.com/badge/github.com/redhat-developer/build">
    </a>
    <a alt="Travis-CI Status" href="https://travis-ci.com/redhat-developer/build">
        <img src="https://travis-ci.com/redhat-developer/build.svg?branch=master">
    </a>
</p>

# The `build` Kubernetes API

Codenamed build-v2

An API to build container-images on Kubernetes using popular strategies and tools like `source-to-image`, `buildpack-v3`, `kaniko`, `jib` and `buildah`, in an extensible way.

## Dependencies

| Dependency                                | Supported versions           |
| ----------------------------------------- | ---------------------------- |
| [Kubernetes](https://kubernetes.io/)      | v1.15.\*, v1.16.\*, v1.17.\* |
| [Tekton](https://cloud.google.com/tekton) | v0.14.2                      |

## How

The following are the build strategies supported by this operator, out-of-the-box:

* [Source-to-Image](docs/buildstrategies.md#source-to-image)
* [Buildpacks-v3](docs/buildstrategies.md#buildpacks-v3)
* [Buildah](docs/buildstrategies.md#buildah)
* [Kaniko](docs/buildstrategies.md#kaniko)

Users have the option to define their own `BuildStrategy` or `ClusterBuildStrategy` resources and make them available for consumption via the `Build` resource.

## Operator Resources

This operator ships four CRDs :

* The `BuildStragegy` CRD and the `ClusterBuildStrategy` CRD is used to register a strategy.
* The `Build` CRD is used to define a build configuration.
* The `BuildRun` CRD is used to start the actually image build using a registered strategy.

## Read the Docs

| Version | Docs                           | Examples                    |
| ------- | ------------------------------ | --------------------------- |
| HEAD    | [Docs @ HEAD](/docs/README.md) | [Examples @ HEAD](/samples) |

## Examples

Examples of `Build` resource using the example strategies shipped with this operator.

* [`buildah`](samples/build/build_buildah_cr.yaml)
* [`buildpacks-v3-heroku`](samples/build/build_buildpacks-v3-heroku_cr.yaml)
* [`buildpacks-v3`](samples/build/build_buildpacks-v3_cr.yaml)
* [`kaniko`](samples/build/build_kaniko_cr.yaml)
* [`source-to-image`](samples/build/build_source-to-image_cr.yaml)

## Try it!

* Get a [Kubernetes](https://kubernetes.io/) cluster and [`kubectl`](https://kubernetes.io/docs/reference/kubectl/overview/) set up to connect to your cluster.
* Install [Tekton](https://cloud.google.com/tekton) by running [install-tekton.sh](hack/install-tekton.sh), it installs v0.14.2.
* Install [operator-sdk][operatorsdk] by running [install-operator-sdk.sh](hack/install-operator-sdk.sh), it installs v0.17.0.
* Create a namespace called **build-examples** by running `kubectl create namespace build-examples`.
* Execute `make local` to register [well-known build strategies](samples/buildstrategy) including **Kaniko** and start the operator locally.
* Create a [Kaniko](samples/build/build_kaniko_cr.yaml) build.

```yaml
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: kaniko-golang-build
  namespace: build-examples
spec:
  source:
    url: https://github.com/sbose78/taxi
    contextDir: .
  strategy:
    name: kaniko
    kind: ClusterBuildStrategy
  dockerfile: Dockerfile
  output:
    image: image-registry.openshift-image-registry.svc:5000/build-examples/taxi-app
```

* Start a [Kaniko](samples/buildrun/buildrun_kaniko_cr.yaml) buildrun

```yaml
apiVersion: build.dev/v1alpha1
kind: BuildRun
metadata:
  name: kaniko-golang-buildrun
  namespace: build-examples
spec:
  buildRef:
    name: kaniko-golang-build
  serviceAccount:
    generate: true
```

## Development

* Build, test & run using [HACK.md](HACK.md).

----

## Roadmap

### Build Strategies Support

| Build Strategy                                                                                  | Alpha | Beta | GA |
| ----------------------------------------------------------------------------------------------- | ----- | ---- | -- |
| [Source-to-Image](samples/buildstrategy/source-to-image/buildstrategy_source-to-image_cr.yaml)  | ☑     |      |    |
| [Buildpacks-v3-heroku](samples/buildstrategy/buildstrategy_buildpacks-v3-heroku_cr.yaml)        | ☑️     |      |    |
| [Buildpacks-v3](samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_cr.yaml)        | ☑️     |      |    |
| [Kaniko](samples/buildstrategy/kaniko/buildstrategy_kaniko_cr.yaml)                             | ☑️     |      |    |
| [Buildah](samples/buildstrategy/buildah/buildstrategy_buildah_cr.yaml)                          | ☑️     |      |    |

### Features

| Feature               | Alpha | Beta | GA |
| --------------------- | ----- | ---- | -- |
| Private Git Repos     | ☑️     |      |    |
| Private Output Image Registry     | ☑️     |      |    |
| Private Builder Image Registry     | ☑️     |      |    |
| Cluster scope BuildStrategy     | ☑️     |      |    |
| Runtime Base Image    | ⚪️    |      |    |
| Binary builds         |       |      |    |
| Image Caching         |       |      |    |
| ImageStreams support  |       |      |    |
| Entitlements          |       |      |    |

[corev1container]: https://github.com/kubernetes/api/blob/v0.17.3/core/v1/types.go#L2106
[pipelinesoperator]: https://www.openshift.com/learn/topics/pipelines
[operatorsdk]: https://github.com/operator-framework/operator-sdk
