
# Running the Operator

Assuming you are logged in to an OpenShift/Kubernetes cluster, run

```sh
make clean && make build && make local
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

In the near future, the above would be setup by the operator.


## Building it locally

```sh
make clean && make build
```


* This project uses Golang 1.13+ and operator-sdk 1.15.1.
* The controllers create/watch Tekton objects.

## Unit tests

```sh
make test
```

## End-to-end tests

To run all strategies except buildpacks-v3, execute

```sh
operator-sdk test local ./test/e2e --up-local --namespace 
build-examples
```

To run all strategies including buildpacks-v3, [setup your Quay credentials](samples/buildstrategy/buildpacks-v3#try-it-) and execute

```sh
TEST_IMAGE_REPO=quay.io/shbose/nodejs-ex:latest TEST_IMAGE_REPO_SECRET=regcred  operator-sdk test local ./test/e2e --up-local --namespace build-examples
```

The Build [examples](samples/build) uses by default a predefined `spec.output.image` endpoint. If a customize
container registry is desired, you can prepend the following environment variables when calling the tests:

* **REGISTRY_SECRET**: Name of the docker registry secret to use. This overrides the `spec.output.credentials.name` with your secret.

* **REGISTRY_ENDPOINT**: The registry endpoint. This overrides only the endpoint inside `spec.output.image`

* **REGISTRY_NAMESPACE** The registry namespace.  This overrides only the namespace inside `spec.output.image`

Example:

```sh
REGISTRY_SECRET=some-secret REGISTRY_NAMESPACE=some-namespace REGISTRY_ENDPOINT=us.icr.io  operator-sdk test local ./test/e2e --up-local --namespace build-examples
```
