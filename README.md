<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

<p align="center">
    <img alt="Work in Progress" src="https://img.shields.io/badge/Status-Work%20in%20Progress-informational">
    <a alt="GoReport" href="https://goreportcard.com/report/github.com/shipwright-io/build">
        <img src="https://goreportcard.com/badge/github.com/shipwright-io/build">
    </a>
    <a alt="Travis-CI Status" href="https://travis-ci.org/github/shipwright-io/build">
        <img src="https://travis-ci.org/shipwright-io/build.svg?branch=master">
    </a>
    <img alt="License" src="https://img.shields.io/github/license/shipwright-io/build">
    <a href="https://pkg.go.dev/mod/github.com/shipwright-io/build"> <img src="https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white"></a>
</p>

# Shipwright - a framework for building container images on Kubernetes

Shipwright is an extensible framework for building container images on Kubernetes. With Shipwright,
developers can define and reuse build strategies that build container images for their CI/CD
pipelines. Any tool that builds images within a container can be supported, such
as [Kaniko](https://github.com/GoogleContainerTools/kaniko),
[Cloud Native Buildpacks](https://buildpacks.io/), and [Buildah](https://buildah.io/).

## Dependencies

| Dependency                                | Supported versions                     |
| ----------------------------------------- | -------------------------------------- |
| [Kubernetes](https://kubernetes.io/)      | v1.15.\*, v1.16.\*, v1.17.\*, v1.18.\* |
| [Tekton](https://cloud.google.com/tekton) | v0.20.1                                |

## Build Strategies

The following [Build Strategies](docs/buildstrategies.md) are installed by default:

* [Source-to-Image](docs/buildstrategies.md#source-to-image)
* [Buildpacks-v3](docs/buildstrategies.md#buildpacks-v3)
* [Buildah](docs/buildstrategies.md#buildah)
* [Kaniko](docs/buildstrategies.md#kaniko)

Users have the option to define their own `BuildStrategy` or `ClusterBuildStrategy` resources and make them available for consumption via the `Build` resource.

## Custom Resources

Shipwright defines four CRDs:

* The `BuildStrategy` CRD and the `ClusterBuildStrategy` CRD is used to register a strategy.
* The `Build` CRD is used to define a build configuration.
* The `BuildRun` CRD is used to start the actually image build using a registered strategy.

## Read the Docs

| Version | Docs                           | Examples                    |
| ------- | ------------------------------ | --------------------------- |
| HEAD    | [Docs @ HEAD](/docs/README.md) | [Examples @ HEAD](/samples) |
| [v0.3.0](https://github.com/shipwright-io/build/releases/tag/v0.3.0)    | [Docs @ v0.3.0](https://github.com/shipwright-io/build/tree/v0.3.0/docs) | [Examples @ v0.3.0](https://github.com/shipwright-io/build/tree/v0.3.0/samples) |
| [v0.2.0](https://github.com/shipwright-io/build/releases/tag/v0.2.0)    | [Docs @ v0.2.0](https://github.com/shipwright-io/build/tree/v0.2.0/docs) | [Examples @ v0.2.0](https://github.com/shipwright-io/build/tree/v0.2.0/samples) |
| [v0.1.1](https://github.com/shipwright-io/build/releases/tag/v0.1.1)    | [Docs @ v0.1.1](https://github.com/shipwright-io/build/tree/v0.1.1/docs) | [Examples @ v0.1.1](https://github.com/shipwright-io/build/tree/v0.1.1/samples) |
| [v0.1.0](https://github.com/shipwright-io/build/releases/tag/v0.1.0)    | [Docs @ v0.1.0](https://github.com/shipwright-io/build/tree/v0.1.0/docs) | [Examples @ v0.1.0](https://github.com/shipwright-io/build/tree/v0.1.0/samples) |

## Examples

Examples of `Build` resource using the example strategies installed by default.

* [`buildah`](samples/build/build_buildah_cr.yaml)
* [`buildpacks-v3-heroku`](samples/build/build_buildpacks-v3-heroku_cr.yaml)
* [`buildpacks-v3`](samples/build/build_buildpacks-v3_cr.yaml)
* [`kaniko`](samples/build/build_kaniko_cr.yaml)
* [`source-to-image`](samples/build/build_source-to-image_cr.yaml)

## Try it!

* Get a [Kubernetes](https://kubernetes.io/) cluster and [`kubectl`](https://kubernetes.io/docs/reference/kubectl/overview/) set up to connect to your cluster.
* Clone this repository from GitHub at the v0.3.0 tag:

  ```bash
  $ git clone --branch v0.3.0 https://github.com/shipwright-io/build.git
  ...
  $ cd build/
  ```

  _Coming soon - install Shipwright Build via kubectl!_

* Install [Tekton](https://cloud.google.com/tekton) by running [hack/install-tekton.sh](hack/install-tekton.sh), it installs v0.20.1.

  ```bash
  $ hack/install-tekton.sh
  ```

* Install Shipwright and sample strategies via `make`:

  ```bash
  $ make install
  ```

* Add a push secret to your container image repository, such as one on Docker Hub or quay.io:

  ```yaml
  $ kubectl create secret generic push-secret \
  --from-file=.dockerconfigjson=$HOME/.docker/config.json \
  --type=kubernetes.io/dockerconfigjson
  ```

* Create a [Cloud Native Buildpacks](samples/build/build_buildpacks_v3_cr.yaml) build, replacing
  `<MY_REGISTRY>/<MY_USERNAME>/<MY_REPO>` with the registry hostname, username, and repository your
  cluster has access to and that you have permission to push images to.

  ```bash
  $ kubectl apply -f - <<EOF
  apiVersion: build.dev/v1alpha1
  kind: Build
  metadata:
    name: buildpack-nodejs-build
  spec:
    source:
      url: https://github.com/adambkaplan/shipwright-example-nodejs.git
    strategy:
      name: buildpacks-v3
      kind: ClusterBuildStrategy
    output:
      image: <MY_REGISTRY>/<MY_USERNAME>/<MY_REPO>:latest
      credentials:
        name: push-secret
  EOF
  ```

* Run your build:

  ```bash
  $ kubectl apply -f - <<EOF
  apiVersion: build.dev/v1alpha1
  kind: BuildRun
  metadata:
    name: buildpack-nodejs-build-1
  spec:
    buildRef:
      name: buildpack-nodejs-build
    serviceAccount:
      name: default
  EOF
  ```

## Roadmap

### Build Strategies Support

| Build Strategy                                                                                  | Alpha | Beta | GA |
| ----------------------------------------------------------------------------------------------- | ----- | ---- | -- |
| [Source-to-Image](samples/buildstrategy/source-to-image/buildstrategy_source-to-image_cr.yaml)  | ☑     |      |    |
| [Buildpacks-v3-heroku](samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3-heroku_cr.yaml)        | ☑️     |      |    |
| [Buildpacks-v3](samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_cr.yaml)        | ☑️     |      |    |
| [Kaniko](samples/buildstrategy/kaniko/buildstrategy_kaniko_cr.yaml)                             | ☑️     |      |    |
| [Buildah](samples/buildstrategy/buildah/buildstrategy_buildah_cr.yaml)                          | ☑️     |      |    |

## Community meetings

We host weekly meetings for contributors, maintainers and anyone interested in the project. The weekly meetings take place on Monday´s at 2pm UTC.

* Meeting [minutes](https://github.com/shipwright-io/build/issues?q=is%3Aissue+label%3Acommunity+label%3Ameeting+is%3Aopen)
* Public calendar [invite](https://calendar.google.com/calendar/u/1?cid=Y19iMWVndjc3anUyczJkbWNkM2R1ZnAxazhuNEBncm91cC5jYWxlbmRhci5nb29nbGUuY29t)

## Want to contribute

We are so excited to have you!

* See [CONTRIBUTING.md](CONTRIBUTING.md) for an overview of our processes
* See [DEVELOPMENT.md](DEVELOPMENT.md) for how to get started
* See [HACK.md](HACK.md) for how to build, test & run
  (advanced reading material)
- Look at our
  [good first issues](https://github.com/shipwright-io/build/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22)
  and our
  [help wanted issues](https://github.com/shipwright-io/build/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22)
- Contact us:
  - Kubernetes Slack: [#shipwright](https://kubernetes.slack.com/messages/shipwright)
  - Users can discuss help, feature requests, or potential bugs at [shipwright-users@lists.shipwright.io](https://lists.shipwright.io/archives/list/shipwright-users@lists.shipwright.io/).
  Click [here](https://lists.shipwright.io/admin/lists/shipwright-users.lists.shipwright.io/) to join.
  - Contributors can discuss active development topics at [shipwright-dev@lists.shipwright.io](https://lists.shipwright.io/archives/list/shipwright-dev@lists.shipwright.io/).
  Click [here](https://lists.shipwright.io/admin/lists/shipwright-dev.lists.shipwright.io/) to join.
