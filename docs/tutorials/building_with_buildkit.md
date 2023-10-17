<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

# Building with BuildKit

Before starting, make sure you have Tekton and Shipwright Build installed.

See the [Try It!](../../README.md#try-it) section for more information.

## Getting Started

### Registry Authentication

For this tutorial, we will require to create a `tutorial-secret` Kubernetes secret to access a [DockerHub](https://hub.docker.com/) registry, as follows:

```sh
$ REGISTRY_SERVER=https://index.docker.io/v1/ REGISTRY_USER=<your_registry_user> REGISTRY_PASSWORD=<your_registry_password>
$ kubectl create secret docker-registry tutorial-secret --docker-server=$REGISTRY_SERVER --docker-username=$REGISTRY_USER --docker-password=$REGISTRY_PASSWORD  --docker-email=me@here.com
```

_Note_: For more information about authentication, please refer to the related [document](../development/authentication.md).

## Create the strategy

Ensure all strategies are in place, see the [Try It!](../../README.md#try-it) section for more information.

```sh
$ kubectl get cbs
NAME              AGE
buildah           96s
buildkit          78s
buildkit-insecure 78s
buildpacks-v3     96s
kaniko            96s
ko                96s
source-to-image   96s
```

_Note_: For more information about strategies, please refer to the related [docs](../buildstrategies.md).

## Creating a Build

For the Build definition, we will require the following:

- A GitHub repository containing a Go [application](https://github.com/shipwright-io/sample-go/tree/main/docker-build) that requires a `Dockerfile`.
- The `tutorial-secret` we just created.
- The `buildkit` ClusterBuildStrategy.

Let's apply our Build and wait for it to be ready:

```bash
$ export REGISTRY_ORG=<your_registry_org>
$ cat <<EOF | kubectl apply -f -
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: go-tutorial-buildkit
spec:
  source:
    git:
      url: https://github.com/shipwright-io/sample-go
    contextDir: docker-build
  strategy:
    name: buildkit
    kind: ClusterBuildStrategy
  output:
    image: docker.io/${REGISTRY_ORG}/go-tutorial:latest
    pushSecret: tutorial-secret
EOF
```

Verify that the `go-tutorial` Build was created successfully:

```sh
$ kubectl get build
NAME                         REGISTERED   REASON      BUILDSTRATEGYKIND      BUILDSTRATEGYNAME   CREATIONTIME
go-tutorial-buildkit         True         Succeeded   ClusterBuildStrategy   buildkit            4s
```

_Note_: For more information about Build resources, please refer to the related [docs](../build.md).

## Creating a BuildRun

Second, we will create a `BuildRun` resource that references our previous `go-tutorial` Build:

```sh
$ cat <<EOF | kubectl create -f -
apiVersion: shipwright.io/v1beta1
kind: BuildRun
metadata:
  name: a-buildkit-buildrun
spec:
  build:
    name: go-tutorial-buildkit
EOF
```

Wait until your `go-tutorial-buildrun` buildrun is completed:

```sh
$ kubectl get buildrun
NAME                  SUCCEEDED   REASON    STARTTIME   COMPLETIONTIME
a-buildkit-buildrun   Unknown     Pending   4s
```

To know more about the state of the BuildRun, the `.status.conditions` fields provide more data:

```sh
$ kubectl get buildrun a-buildkit-buildrun -o json | jq '.status.conditions[]'
{
  "lastTransitionTime": "2021-04-01T09:15:13Z",
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

Congratulations! You just created a container image from https://github.com/shipwright-io/sample-go using [BuildKit](https://github.com/moby/buildkit).

The new container image should now be available in your container registry.
