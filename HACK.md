<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

# Running the Controller

Assuming you are logged in to an OpenShift/Kubernetes cluster, run

```sh
make clean build local
```

If the `pipeline` service account isn't already created, here are the steps to create the same:

```sh
oc create serviceaccount pipeline
oc adm policy add-scc-to-user privileged -z pipeline
oc adm policy add-role-to-user edit -z pipeline
```

If your `Build`'s `outputImage` is to be pushed to the OpenShift internal registry, ensure the
`pipeline` service account has the required role:

```sh
oc policy add-role-to-user registry-editor pipeline
```

Or

```sh
oc policy add-role-to-user  system:image-builder  pipeline
```

In the near future, the above would be setup by the controller.

## Building it locally

```sh
make clean && make build
```

* This project uses Golang 1.15+ and operator-sdk v0.18.2.
* The controllers create/watch Tekton objects.

# Testing

Please refer to the [testing docs](/docs/development/testing.md) for more information about our test levels.

