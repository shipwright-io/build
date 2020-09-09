<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

# Deploying the operator pod

The following set of steps highlight how to deploy a Build operator pod into an existing Kubernetes cluster.

1. Build a custom docker image from this repository. This can be done with Docker, for example:

   ```sh
   pushd $GOPATH/src/github.com/shipwright-io/build
   docker build -t eeeoo/build-operator:master .
   docker push eeeoo/build-operator:master
   popd
   ```

   Just to illustrate the above, you can find the image in [dockerhub](https://hub.docker.com/repository/docker/eeeoo/build-operator)

2. Reference the generated image name in the [operator.yaml](../../deploy/operator.yaml). The `spec.template.containers[0].image` value should be modified.

3. Target your Kubernetes cluster and install the Tekton pipeline:

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
