<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->
# Shipwright Build Tutorial

So you just successfully built a container image via the [Try It!](../../README.md#try-it) section and you want to know more?

At Shipwright, we've spent a lot of time trying to figure out the best ways to simplify the experience when
building container images. Shipwright provides an alternative for securely building container images in your Kubernetes cluster.


What if we could

- Support any existing tooling for building container images in Kubernetes clusters.
- Minimize the burden of learning a new tool for building images, by abstracting it from users.
- Simplify the user experience when building images via a standardize minimal API.
- Allow users to re-use their existing cluster for building and deploying container images.

## Concepts

| Concept     | Description |
| ----------- | ----------- |
| **`Strategy`**      | Refers to a particular tool that will be used when building a container image, such as Kaniko, Buildah, ko, etc. |
| **`Build`**   | Resource used to define a build configuration. |
| **`BuildRun`**   | Resource used to start the image build mechanism. |
| **`BuildStrategy` / `ClusterBuildStrategy`**   | Resource that holds a template that dictates how to build via a particular strategy. |
| **`Dockerfile-less strategy`**   | Is a category given to strategies that can build container images from source code, without the notion of a Dockerfile. |
| **`Dockerfile-based strategy`**   | Is a category given to strategies that can build container images from source code, with a reference to a Dockerfile. |

With the above concepts in mind, lets see how they all play together.

## Strategies

Shipwright ships with a set of strategies that are available across the cluster.

The default installation includes these [buildstrategies](/docs/buildstrategies.md):

* [Buildpacks-v3](../buildstrategies.md#buildpacks-v3)
* [Kaniko](../buildstrategies.md#kaniko)
* [BuildKit](../buildstrategies.md#buildkit)
* [Source-to-Image](../buildstrategies.md#source-to-image)
* [Buildah](../buildstrategies.md#buildah)
* [ko](../buildstrategies.md#ko)

For more information about strategies see the related [docs](../buildstrategies.md).

## Examples

* [Example with Kaniko](building_with_kaniko.md)
* [Example with Buildpacks](building_with_buildpacks.md)
* [Example with BuildKit](building_with_buildkit.md)

Depending on your source code you might want to try a specific example. The following table serves as a guide to help you understand which
strategy to choose:

| Sample code | Repository | ContextDir | Strategy Type | Strategy to use |
| ----------- | ----------- | ------------- | ------------- | ------------- |
| A go app with a Dockerfile | [shipwright-io/sample-go](https://github.com/shipwright-io/sample-go) | `/docker-build` | Dockerfile-based | Kaniko, BuildKit, Buildah |
| A go app | [shipwright-io/sample-go](https://github.com/shipwright-io/sample-go) | `/source-build` | Dockerfile-less | buildpacks-v3, buildpacks-v3-heroku |
| A ruby app | [shipwright-io/sample-ruby](https://github.com/shipwright-io/sample-ruby) | `/source-build` | Dockerfile-less | buildpacks-v3, buildpacks-v3-heroku |
| A java app with a Dockerfile | [shipwright-io/sample-java](https://github.com/shipwright-io/sample-java) | `/docker-build` | Dockerfile-based | Kaniko, BuildKit, Buildah |
| Shipwright/Build | [shipwright-io/build](https://github.com/shipwright-io/build) |  `/cmd/shipwright-build-controller` | Dockerfile-less | ko |

_Note_: `ContextDir` is the path under the repository where the source code is located.

_Note_: `Buildpacks-v3` support is provided via Paketo and Heroku. Paketo is our default tool, so any reference to buildpacks-v3 usually implies the usage of [Paketo](https://paketo.io/).
