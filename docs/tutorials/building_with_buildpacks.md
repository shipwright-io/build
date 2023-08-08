<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

# Building with Paketo

Before starting, make sure you have Tekton and Shipwright Build installed.

See the [Try It!](../../README.md#try-it) section for more information.

## Getting Started

### Registry Authentication

For this tutorial, we will require to create a `tutorial-secret` Kubernetes secret to access a [DockerHub](https://hub.docker.com/) registry, as follows:

```sh
$ REGISTRY_SERVER=https://index.docker.io/v1/ REGISTRY_USER=<your_registry_user> REGISTRY_PASSWORD=<your_registry_password>
$ kubectl create secret docker-registry tutorial-secret --docker-server=$REGISTRY_SERVER --docker-username=$REGISTRY_USER --docker-password=$REGISTRY_PASSWORD  --docker-email=me@here.com
```

_Note_: For more information about authentication, please refer to the related [docs](../development/authentication.md).

## Create the strategy

Ensure all strategies are in place, see the [Try It!](../../README.md#try-it) section for more information.

```sh
$ kubectl get cbs
NAME              AGE
buildah           2m
buildpacks-v3     2m
kaniko            2m
ko                2m
source-to-image   2m
```

Verify that the strategy exists in the cluster:

```sh
$ kubectl get cbs
NAME            AGE
buildpacks-v3   23m
```

_Note_: For more information about strategies, please refer to the related [docs](../buildstrategies.md).

## Creating a Build

For the Build definition, we will require the following:

- A GitHub repository containing a Ruby [application](https://github.com/shipwright-io/sample-ruby).
- The `tutorial-secret` we just created.
- The `buildpacks-v3` ClusterBuildStrategy.

Let's apply our Build and wait for it to be ready:

```bash
$ export REGISTRY_ORG=<your_registry_org>
$ cat <<EOF | kubectl apply -f -
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: ruby-tutorial
spec:
  source:
    git:
      url: https://github.com/shipwright-io/sample-ruby
    contextDir: source-build
  strategy:
    name: buildpacks-v3
    kind: ClusterBuildStrategy
  output:
    image: docker.io/${REGISTRY_ORG}/ruby-tutorial:latest
    pushSecret: tutorial-secret
EOF
```

Verify that the `go-tutorial` Build was created successfully:

```sh
kubectl get build
NAME            REGISTERED   REASON      BUILDSTRATEGYKIND      BUILDSTRATEGYNAME   CREATIONTIME
ruby-tutorial   True         Succeeded   ClusterBuildStrategy   buildpacks-v3       22s
```

_Note_: For more information about Build resources, please refer to the related [docs](../build.md).

## Creating a BuildRun

Second, we will create a `BuildRun` resource that references our previous `go-tutorial` Build:

```sh
$ cat <<EOF | kubectl create -f -
apiVersion: shipwright.io/v1beta1
kind: BuildRun
metadata:
  name: ruby-tutorial-buildrun
spec:
  build:
    name: ruby-tutorial
EOF
```

Wait until your `go-tutorial-buildrun` buildrun is completed:

```sh
$ kubectl get buildrun
NAME                     SUCCEEDED   REASON      STARTTIME   COMPLETIONTIME
ruby-tutorial-buildrun   True        Succeeded   115s        51s
```

To know more about the state of the BuildRun, the `.status.conditions` fields provide more data:

```sh
$ kubectl get buildrun ruby-tutorial-buildrun -o json | jq '.status.conditions[]'
{
  "lastTransitionTime": "2021-03-24T15:02:38Z",
  "message": "All Steps have completed executing",
  "reason": "Succeeded",
  "status": "True",
  "type": "Succeeded"
}
```

_Note_: A BuildRun is a resource that runs to completion. The `REASON` column reflects the state of the resource. If the BuildRun ran to completion successfully,
a `Succeeded` `REASON` is expected.

_Note_: For more information about BuildRun resources, please refer to the related [docs](../buildrun.md).

## Closing

Congratulations! You just created a container image from https://github.com/shipwright-io/sample-ruby using [Paketo Buildpacks](https://paketo.io/).

The new container image should now be available in your container registry.
