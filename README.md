<p align="center">
    <img alt="Work in Progress" src="https://img.shields.io/badge/Status-Work%20in%20Progress-informational">
    <a alt="GoReport" href="https://goreportcard.com/report/github.com/redhat-developer/build">
        <img src="https://goreportcard.com/badge/github.com/redhat-developer/build">
    </a>
    <a alt="Travis-CI Status" href="https://travis-ci.com/redhat-developer/build">
        <img src="https://travis-ci.com/redhat-developer/build.svg?branch=master">
    </a>
</p>

## `build-v2` Kubernetes Operator

An API to build container-images on Kubernetes using popular strategies and tools like
`source-to-image`, `buildpack-v3`, `kaniko` and `buildah`, in an extensible way.

## How

The following are the `BuildStrategies` supported by this operator, out-of-the-box:

* [Source-to-Image](samples/buildstrategy/source-to-image/README.md);
* [Buildpacks-v3](samples/buildstrategy/buildpacks-v3/README.md);
* [Buildah](samples/buildstrategy/buildah/README.md);
* [Kaniko](samples/buildstrategy/kaniko/README.md);

Users have the option to define their own `BuildStrategies` and make them available for consumption
by `Build`s.

## Operator Resources

This operator ships two resources in order to define your strategy and register the actual
application builds.

### `BuildStrategy`

The resource `BuildStrategy` (`builds.build.dev/v1alpha1`) allows you to define a shared group of
steps needed to fullfil the application build. Those steps are defined as
[`containers/v1`][corev1container] entries.

```yml
---
apiVersion: build.dev/v1alpha1
kind: BuildStrategy
metadata:
  name: source-to-image
spec:
  buildSteps:
...
```

Attributes:
* `spec.buildSteps`: array of `containers.core/v1` resources;

More example of strategies can be found on this [directory](samples/buildstrategy).

### `Build`

The resource `Build` (`builds.dev/v1alpha1`) binds together source-code and `BuildStrategy`
culminating in the actual appplication build process being executed in Kubernetes. Please consider
the following example:

```yml
---
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: example-build-buildpack
spec:
  source:
    url: https://github.com/sclorg/nodejs-ex
    credentials:
      name: source-repository-credentials
  strategy:
    name: buildpacks-v3
    namespace: openshift
  builderImage: heroku/buildpacks:18
  output:
    image: quay.io/olemefer/nodejs-ex:v1
    credentials:
      name: quayio-olemefer
```

Attributes:

* `spec.source.url`: source-code repository URL;
* `spec.source.credentials.name`: Kubernetes Secret name carrying source repository credentials;
* `spec.strategy.name`: `BuildStrategy` name;
* `spec.strategy.namespace`: `BuildStrategy` namespace;
* `spec.builderImage`: container image employed during the build process;
* `spec.output.image`: container image to be produced;
* `spec.output.image`: Kubernetes Secret name with container registry credentials;

The resource is updated as soon as the current building status changes:

```
$ kubectl get builds.build.dev buildpacks
NAME         STATUS
buildpacks   Running
```

And finally:

```
$ kubectl get builds.build.dev buildpacks
NAME         STATUS
buildpacks   Succeeded
```

#### Examples

Examples of `Build` resource using the example strategies shipped with this operator.

* [`buildah`](./samples/build/build_buildah_cr.yaml);
* [`buildpacks-v3`](./samples/build/build_buildpacks-v3_cr.yaml);
* [`kaniko`](./samples/build/build_kaniko_cr.yaml);
* [`source-to-image`](.samples/build/build_source-to-image_cr.yaml);

----

## Try it!

- Install Tekton, where optionally you could use
[OpenShift Pipelines Community Operator][pipelinesoperator];
- Install [`operator-sdk`][operatorsdk];
- Execute `make local`;
- Start a sample [Kaniko](samples/build/build_kaniko_cr.yaml) build;

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
| Runtime Base Image    | ⚪️    |      |    |
| Binary builds         |       |      |    |
| Image Caching         |       |      |    |
| ImageStreams support  |       |      |    |
| Entitlements          |       |      |    |

[corev1container]: https://github.com/kubernetes/api/blob/v0.17.3/core/v1/types.go#L2106
[pipelinesoperator]: https://www.openshift.com/learn/topics/pipelines
[operatorsdk]: https://github.com/operator-framework/operator-sdk