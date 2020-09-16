<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

# Developing

## Getting started

1.  [Ramp up on kubernetes and CRDs](#ramp-up)
1.  Create [a GitHub account](https://github.com/join)
1.  Setup
    [GitHub access via SSH](https://help.github.com/articles/connecting-to-github-with-ssh/)
1.  [Create and checkout a repo fork](#checkout-your-fork)
1.  Set up your [shell environment](#environment-setup)
1.  Install [requirements](#requirements)
1.  [Set up a Kubernetes cluster](#kubernetes-cluster)
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
1.  [`docker`](https://docs.docker.com/get-docker/): For building operator image
1.  [`kubectl`](https://kubernetes.io/docs/tasks/tools/install-kubectl/): For
    interacting with your kube cluster

## Kubernetes cluster

The recommended configuration is:

-   Kubernetes version `1.17` or later
-   4 vCPU nodes (`n1-standard-4`)
-   Node autoscaling, up to 3 nodes
-   API scopes for cloud-platform


### To setup a cluster using MiniKube:

- Follow instructions for your platform to [Install Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/) and start a session as follows:

```bash
minikube start eval $(minikube docker-env)
```

### To setup a cluster with Docker Desktop:

Docker Desktop versions come integrated with an edge version of Kubernetes that has been proven to work for both developing and running Shipwright Build. To find out what Kubernetes a specific version of Docker Desktop includes, please refer to the release notes for your platform here: https://docs.docker.com/.

To enable the Kubernetes that comes with Docker Desktop:

1.  From the Docker Desktop dropdown menu, open the `preferences...` interface.

1. Under the `Resources` tab ensure that in the `ADVANCED` menuitem you have at allocated at least 4 CPUs, 8.0 GiB Memory, and 1.0 GiB Swap.

1.  Under the `Kubernetes` tab, check the   `Enable Kubernetes` box.

    * *Note: the Kubernetes version Docker Desktop will use is displayed at the top of the window.*

1.  Click the `Apply and Restart` button to save the preferences.

1.  Switch the proper `kubectl` config context:

    ```bash
    kubectl config get-contexts # You should see docker-for-desktop in the previous command output
    kubectl config use-context docker-for-desktop
    ```
    * *Note: Docker Desktop menu provides a `Kubernetes` menuitem that allows you to select between contexts which is equivalent to the `kubectl` command.*

## Environment Setup

To run your operator, you'll need to set these environment variables (we recommend adding them to your `.bashrc`):

1.  `GOPATH`: If you don't have one, simply pick a directory and add `export
    GOPATH=...`
1.  `$GOPATH/bin` on `PATH`: This is so that tooling installed via `go get` will
    work properly.

`.bashrc` example:

```shell
export GOPATH="$HOME/go"
export PATH="${PATH}:${GOPATH}/bin"
```

Make sure to configure
[authentication](https://docs.docker.com/engine/reference/commandline/login/)
for your `docker` if required. To be able to push images to the container registry, you need to run this once:

```sh
docker login [OPTIONS] [SERVER]
```

## Install Shipwright Build

The following set of steps highlight how to deploy a Build operator pod into an existing Kubernetes cluster.

1. Build a custom docker image from this repository. This can be done with Docker, for example:

   ```sh
   pushd $GOPATH/src/github.com/shipwright-io/build
   docker build -t eeeoo/build-operator:master .
   docker push eeeoo/build-operator:master
   popd
   ```

   You can also find the official versioned operator image in our [quay.io](https://quay.io/repository/shipwright/shipwright-operator?tab=tags)

2. Reference the generated image name in the [operator.yaml](deploy/operator.yaml). The `spec.template.containers[0].image` value should be modified.

3. Target your Kubernetes cluster and install the Shipwright Build:

    ```sh
    pushd $GOPATH/src/github.com/shipwright-io/build
    ./hack/install-tekton.sh
    popd
    ```

4. Install the Build operator pod and all related resources.

    ```sh
    pushd $GOPATH/src/github.com/shipwright-io/build
    ./hack/crd.sh install
    popd
    ```

The above four steps give you a running Build operator that executes the code from your current branch.

### Redeploy operator

As you make changes to the code, you can redeploy your operator with:

```sh
pushd $GOPATH/src/github.com/shipwright-io/build
./hack/crd.sh install
popd
```

### Tear it down

You can clean up everything with:

```sh
pushd $GOPATH/src/github.com/shipwright-io/build
./hack/crd.sh uninstall
popd
```

### Accessing logs

To look at the operator logs, run:

```sh
kubectl -n build-operator logs $(kubectl -n build-operator get pods -l name=build-operator -o name)
```
