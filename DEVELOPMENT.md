<!--
Copyright 2018, 2020 The Tekton Authors
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0

Documentation inspired from https://github.com/tektoncd/pipeline/blob/ce7591acec8a6aa726d88e5cc057588665881ace/DEVELOPMENT.md
-->

# Developing

## Getting started

1.  [Ramp up on kubernetes and CRDs](#ramp-up)
1.  Create [a GitHub account](https://github.com/join)
1.  Setup
    [GitHub access via SSH](https://help.github.com/articles/connecting-to-github-with-ssh/)
1.  [Create and checkout a repo fork](#checkout-your-fork)
1.  Install [requirements](#requirements)
1.  [Set up a Kubernetes cluster](#create-a-cluster-and-a-repo)
1.  Set up your [shell environment](#environment-setup)
1.  [Configure kubectl to use your cluster](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/)
1.  [Install Shipwright Build in your cluster](#install-shipwright-build)

### Ramp up

Welcome to the project!! You may find these resources helpful to ramp up on some
of the technology this project is built on. This project extends Kubernetes (aka
`k8s`) with Custom Resource Definitions (CRDs). To find out more:

-   [The Kubernetes docs on Custom Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) -
    These will orient you on what words like "Resource" and "Controller"
    concretely mean
-   [Understanding Kubernetes objects](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/) -
    This will further solidify k8s nomenclature
-   [API conventions - Types(kinds)](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#types-kinds) -
    Another useful set of words describing words. "Objects" and "Lists" in k8s
    land
-   [Extend the Kubernetes API with CustomResourceDefinitions](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/)-
    A tutorial demonstrating how a Custom Resource Definition can be added to
    Kubernetes without anything actually "happening" beyond being able to list
    Objects of that kind

At this point, you may find it useful to return to these `Shipwright Build` docs:

-   [Shipwright Build](https://github.com/shipwright-io/build/blob/master/README.md) -
    Some of the terms here may make more sense!
-   Install via [getting started for development](#getting-started)
-   [Shipwright Build overview and tutorial](https://github.com/shipwright-io/build/blob/master/docs/README.md) -
    Define `BuildStrategies`, `Builds`, and `BuildRuns`, see what happens when
    they are run

### Checkout your fork

The Go tools require that you clone the repository to the
`src/github.com/shipwright-io/build` directory in your
[`GOPATH`](https://github.com/golang/go/wiki/SettingGOPATH).

To check out this repository:

1.  Create your own
    [fork of this repo](https://help.github.com/articles/fork-a-repo/)
1.  Clone it to your machine:

```shell
mkdir -p ${GOPATH}/src/github.com/shipwright-io
cd ${GOPATH}/src/github.com/shipwright-io
git clone git@github.com:${YOUR_GITHUB_USERNAME}/build.git
cd build
git remote add upstream git@github.com:shipwright-io/build.git
git remote set-url --push upstream no_push
```

_Adding the `upstream` remote sets you up nicely for regularly
[syncing your fork](https://help.github.com/articles/syncing-a-fork/)._

### Requirements

You must install these tools:

1.  [`go`](https://golang.org/doc/install): The language Shipwright Build is
    built in
1.  [`git`](https://help.github.com/articles/set-up-git/): For source control
1   [`ko`](https://github.com/google/ko): To build and deploy changes.
1.  [`kubectl`](https://kubernetes.io/docs/tasks/tools/install-kubectl/): For
    interacting with your kube cluster

### Create a cluster and a repo

1. Follow the instructions in the Kubernetes doc to [Set up a kubernetes cluster](https://kubernetes.io/docs/setup/)
1. Set up a container image repository for pushing images. Any container image registry that is accessible to your cluster can be used for your repository. This can be a public registry like [Docker Hub](https://docs.docker.com/docker-hub/), [quay.io](https://quay.io), or a container registry runs by your cloud provider

**Note**: We support Kubernetes version `1.17` and `1.18`, 1 cluster worker node for basic usage, 2+ cluster worker nodes for HA

## Environment Setup

To run your controller, you'll need to set these environment variables (we recommend adding them to your `.bashrc`):

1.  `GOPATH`: If you don't have one, simply pick a directory and add `export
    GOPATH=...`
1.  `$GOPATH/bin` on `PATH`: This is so that tooling installed via `go get` will
    work properly.

`.bashrc` example:

```shell
export GOPATH="$HOME/go"
export PATH="${PATH}:${GOPATH}/bin"
```

Make sure to configure [authentication](https://github.com/google/ko#authenticating) if required. To be able to push images to the container registry, you need to run this once:

```sh
ko login [OPTIONS] [SERVER]
```

Note: This is roughly equivalent to [`docker login`](https://docs.docker.com/engine/reference/commandline/login/).

## Install Shipwright Build

The following set of steps highlight how to deploy a Build controller pod into an existing Kubernetes cluster.

1. Target your Kubernetes cluster and install the Shipwright Build. Run this from the root of the source repo:

    ```sh
    ./hack/install-tekton.sh
    ```

1. Set your `KO_DOCKER_REPO` environment variable. This will be the container
   image registry you push to, or `kind.local` if you're using
[KinD](https://kind.sigs.k8s.io).

1. Build and deploy the controller from source, from within the root of the repo:

   ```sh
   ko apply -P -R -f deploy/
   ```

The above steps give you a running Build controller that executes the code from your current branch.

### Redeploy controller

As you make changes to the code, you can redeploy your controller with:

   ```sh
   ko apply -P -R -f deploy/
   ```
You may use the following command to re-generate CRDs of build and buildrun if you change their spec:
   ```sh
   make generate-crds
   ```

### Tear it down

You can clean up everything with:

   ```sh
   kubectl delete -R -f deploy/
   ```

### Accessing logs

To look at the controller logs, run:

```sh
kubectl -n shipwright-build logs $(kubectl -n shipwright-build get pods -l name=shipwright-build-controller -o name)
```

