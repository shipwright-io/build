# Deploying the operator pod

The following set of steps highlight how to deploy a Build operator pod into an existing Kubernetes cluster.

1. Build a custom docker image from this repository. This can be done with Docker, for example:

   ```sh
   pushd $GOPATH/src/github.com/redhat-developer/build
   docker build -t eeeoo/build-operator:master .
   docker push eeeoo/build-operator:master
   popd
   ```
   Just to illustrate the above, you can find the image in [dockerhub](https://hub.docker.com/repository/docker/eeeoo/build-operator)

2. Reference the generated image name in the [operator.yaml](../../deploy/operator.yaml). The `spec.template.containers[0].image` value should be modified.

3. Target your Kubernetes cluster and create the `build-operator` namespace.

    ```sh
    kubectl create ns build-operator
    ```

4. Install Tekton pipeline controller:

    ```sh
    pushd $GOPATH/src/github.com/redhat-developer/build
    ./hack/install-tekton.sh
    popd
    ```

5. Install the Build operator pod.

    ```sh
    pushd $GOPATH/src/github.com/redhat-developer/build
    ./hack/crd.sh install
    popd
    ```

The above five steps give you a running Build operator that executes the code from your current branch.

