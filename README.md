<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

<p align="center">
    <img alt="Work in Progress" src="https://img.shields.io/badge/Status-Work%20in%20Progress-informational">
    <a alt="GoReport" href="https://goreportcard.com/report/github.com/shipwright-io/build">
        <img src="https://goreportcard.com/badge/github.com/shipwright-io/build">
    </a>
    <img alt="License" src="https://img.shields.io/github/license/shipwright-io/build">
    <a href="https://pkg.go.dev/mod/github.com/shipwright-io/build"> <img src="https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white"></a>
</p>

# ![shipwright-logo](./.docs/shipwright-logo-lightbg-512.png)

Shipwright is an extensible framework for building container images on Kubernetes.

## Why?

With Shipwright, developers get a simplified approach for building container images, by defining a minimal YAML that does not require
any previous knowledge of containers or container tooling. All you need is your source code in git and access to a container registry.

Shipwright supports any tool that can build container images in Kubernetes clusters, such as:

- [Kaniko](https://github.com/GoogleContainerTools/kaniko)
- [Cloud Native Buildpacks](https://buildpacks.io/)
- [BuildKit](https://github.com/moby/buildkit)
- [Buildah](https://buildah.io/)

## Try It!

* We assume you already have a Kubernetes cluster (v1.17+). If you don't, you can use [KinD](https://kind.sigs.k8s.io), which you can install by running [`./hack/install-kind.sh`](./hack/install-kind.sh).

* We also require a Tekton installation (v0.19+). To install the latest version, run:

  ```bash
  $ kubectl apply --filename https://storage.googleapis.com/tekton-releases/pipeline/previous/v0.20.1/release.yaml
  ```

* Install the Shipwright deployment. To install the latest version, run:

  ```bash
  $ kubectl apply --filename https://github.com/shipwright-io/build/releases/download/nightly/nightly-2021-03-24-1616591545.yaml
  ```

* Install the Shipwright strategies. To install the latest version, run:

  ```bash
  $ kubectl apply --filename https://github.com/shipwright-io/build/releases/download/nightly/default_strategies.yaml
  ```

* Generate a secret to access your container registry, such as one on [Docker Hub](https://hub.docker.com/) or [Quay.io](https://quay.io/):

  ```bash
  $ REGISTRY_SERVER=https://index.docker.io/v1/ REGISTRY_USER=<your_registry_user> REGISTRY_PASSWORD=<your_registry_password>
  $ kubectl create secret docker-registry push-secret \
      --docker-server=$REGISTRY_SERVER \
      --docker-username=$REGISTRY_USER \
      --docker-password=$REGISTRY_PASSWORD  \
      --docker-email=me@here.com
  ```

* Create a Build object, replacing `<REGISTRY_ORG>` with the registry username your `push-secret` secret have access to:

  ```bash
  $ REGISTRY_ORG=<your_registry_org>
  $ cat <<EOF | kubectl apply -f -
  apiVersion: shipwright.io/v1alpha1
  kind: Build
  metadata:
    name: buildpack-nodejs-build
  spec:
    source:
      url: https://github.com/shipwright-io/sample-nodejs
      contextDir: source-build
    strategy:
      name: buildpacks-v3
      kind: ClusterBuildStrategy
    output:
      image: docker.io/${REGISTRY_ORG}/sample-nodejs:latest
      credentials:
        name: push-secret
  EOF
  ```

  ```bash
  $ kubectl get builds
  NAME                     REGISTERED   REASON      BUILDSTRATEGYKIND      BUILDSTRATEGYNAME   CREATIONTIME
  buildpack-nodejs-build   True         Succeeded   ClusterBuildStrategy   buildpacks-v3       68s
  ```

* Submit your buildrun:

  ```bash
  $ cat <<EOF | kubectl create -f -
  apiVersion: shipwright.io/v1alpha1
  kind: BuildRun
  metadata:
    generateName: buildpack-nodejs-buildrun-
  spec:
    buildRef:
      name: buildpack-nodejs-build
  EOF
  ```

* Wait until your buildrun is completed:

  ```bash
  $ kubectl get buildruns
  NAME                              SUCCEEDED   REASON      STARTTIME   COMPLETIONTIME
  buildpack-nodejs-buildrun-xyzds   True        Succeeded   69s         2s
  ```

  or

  ```bash
  $ kubectl get buildrun --output name | xargs kubectl wait --for=condition=Succeeded --timeout=180s
  ```

* After your buildrun is completed, check your container registry, you will find the new generated image uploaded there.

## Please tell us more!

Depending on your source code, you might want to build it differently with Shipwright.

To find out more on what's the best strategy or what else can Shipwright do for you, please visit our [tutorial](./docs/tutorials/README.md)!

## More information

### Read the Docs

| Version | Docs                           | Examples                    |
| ------- | ------------------------------ | --------------------------- |
| HEAD    | [Docs @ HEAD](/docs/README.md) | [Examples @ HEAD](/samples) |
| [v0.4.0](https://github.com/shipwright-io/build/releases/tag/v0.4.0)    | [Docs @ v0.4.0](https://github.com/shipwright-io/build/tree/v0.4.0/docs) | [Examples @ v0.4.0](https://github.com/shipwright-io/build/tree/v0.4.0/samples) |
| [v0.3.0](https://github.com/shipwright-io/build/releases/tag/v0.3.0)    | [Docs @ v0.3.0](https://github.com/shipwright-io/build/tree/v0.3.0/docs) | [Examples @ v0.3.0](https://github.com/shipwright-io/build/tree/v0.3.0/samples) |
| [v0.2.0](https://github.com/shipwright-io/build/releases/tag/v0.2.0)    | [Docs @ v0.2.0](https://github.com/shipwright-io/build/tree/v0.2.0/docs) | [Examples @ v0.2.0](https://github.com/shipwright-io/build/tree/v0.2.0/samples) |
| [v0.1.1](https://github.com/shipwright-io/build/releases/tag/v0.1.1)    | [Docs @ v0.1.1](https://github.com/shipwright-io/build/tree/v0.1.1/docs) | [Examples @ v0.1.1](https://github.com/shipwright-io/build/tree/v0.1.1/samples) |
| [v0.1.0](https://github.com/shipwright-io/build/releases/tag/v0.1.0)    | [Docs @ v0.1.0](https://github.com/shipwright-io/build/tree/v0.1.0/docs) | [Examples @ v0.1.0](https://github.com/shipwright-io/build/tree/v0.1.0/samples) |

### Dependencies

| Dependency                           | Supported versions           |
| -------------------------------------| ---------------------------- |
| [Kubernetes](https://kubernetes.io/) | v1.17.\*, v1.18.\*, v1.19.\*, v1.20.\* |
| [Tekton](https://tekton.dev)         | v0.19.0, v0.20.\*, v0.21.0, v0.22.0, v0.23.0 |

### Platform support

We are building container images of the Shipwright Build controller for all platforms supported by the base image that we are using which is [registry.access.redhat.com/ubi8/ubi-minimal](https://catalog.redhat.com/software/containers/ubi8/ubi-minimal/5c359a62bed8bd75a2c3fba8). Those are:

- linux/amd64
- linux/arm64
- linux/ppc64le
- linux/s390x

All these platforms are also supported by our Tekton Pipelines dependency. Our own tests as part of our CI pipeline are all only running on and testing the linux/amd64 platform.

Our sample build strategies are all functional on linux/amd64. Their support on other platforms relies on the tools being used there to be available for other platforms. For detailed information, please see [Available ClusterBuildStrategies](docs/buildstrategies.md#available-clusterbuildstrategies).

## Want to get involved?

### Community meetings

We host weekly meetings for users, contributors, maintainers and anyone interested in the project. The weekly meetings take place on Mondays at 1pm UTC.

* Meeting [minutes](https://github.com/shipwright-io/build/issues?q=is%3Aissue+label%3Acommunity+label%3Ameeting+is%3Aopen)
* Public calendar [invite](https://calendar.google.com/calendar/u/1?cid=Y19iMWVndjc3anUyczJkbWNkM2R1ZnAxazhuNEBncm91cC5jYWxlbmRhci5nb29nbGUuY29t)

### Want to contribute

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
