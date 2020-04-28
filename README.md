<p align="center">
    <img alt="Work in Progress" src="https://img.shields.io/badge/Status-Work%20in%20Progress-informational">
    <a alt="GoReport" href="https://goreportcard.com/report/github.com/redhat-developer/build">
        <img src="https://goreportcard.com/badge/github.com/redhat-developer/build">
    </a>
    <a alt="Travis-CI Status" href="https://travis-ci.com/redhat-developer/build">
        <img src="https://travis-ci.com/redhat-developer/build.svg?branch=master">
    </a>
</p>

## The `build` Kubernetes API

Codenamed build-v2

An API to build container-images on Kubernetes using popular strategies and tools like `source-to-image`, `buildpack-v3`, `kaniko`, `jib` and `buildah`, in an extensible way.

## How

The following are the `BuildStrategies` supported by this operator, out-of-the-box:

* [Source-to-Image](samples/buildstrategy/source-to-image/README.md)
* [Buildpacks-v3](samples/buildstrategy/buildpacks-v3/README.md)
* [Buildah](samples/buildstrategy/buildah/README.md)
* [Kaniko](samples/buildstrategy/kaniko/README.md)

Users have the option to define their own `BuildStrategies` resources and make them available for consumption
via the `Build` resource.

## Operator Resources

This operator ships two CRDs(the `Build` and `BuildRun`) in order to register a strategy and then start the actual application builds using a registered strategy.

## Read the Docs

| Version | Docs | Examples |
| ------- | ---- | -------- |
| HEAD | [Docs @ HEAD](/docs/README.md) | [Examples @ HEAD](/samples) |

## Examples

Examples of `Build` resource using the example strategies shipped with this operator.

* [`buildah`](./samples/build/build_buildah_cr.yaml)
* [`buildpacks-v3`](./samples/build/build_buildpacks-v3_cr.yaml)
* [`kaniko`](./samples/build/build_kaniko_cr.yaml)
* [`source-to-image`](.samples/build/build_source-to-image_cr.yaml)

## Try it!

- Install Tekton, optionally you could use
[OpenShift Pipelines Community Operator][pipelinesoperator]

- Install [operator-sdk][operatorsdk]

- Create a project or namespace called **build-examples** by using `kubectl create namespace build-examples`

- Execute `make local` to register [well-known build strategies](samples/buildstrategies) including **Kaniko**
and start the operator.

- Create a [Kaniko](samples/build/build_kaniko_cr.yaml) build

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

- Start a [Kaniko](samples/buildrun/buildrun_kaniko_cr.yaml) buildrun

```yaml
apiVersion: build.dev/v1alpha1
kind: BuildRun
metadata:
  name: kaniko-golang-buildrun
  namespace: build-examples
spec:
  buildRef:
    name: kaniko-golang-build
```

## Development

* Build, test & run using [HACK.md](HACK.md).

----

## Roadmap

### Build Strategies Support

| Build Strategy                                                                  | Alpha | Beta | GA |
| ------------------------------------------------------------------------------- | ----- | ---- | -- |
| [Source-to-Image](samples/buildstrategy/buildstrategy_source-to-image_cr.yaml)  | ☑     |      |    |
| [Buildpacks-v3](samples/buildstrategy/buildstrategy_buildpacks-v3-cr.yaml)      | ☑️     |      |    |
| [Kaniko](samples/buildstrategy/buildstrategy_kaniko_cr.yaml)                    | ☑️     |      |    |
| [Buildah](samples/buildstrategy/buildstrategy_buildah_cr.yaml)                  | ☑️     |      |    |


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
