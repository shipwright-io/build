
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

The following is a list of environment variables you can use when running e2e tests, this will override specific paths under the **Build** CRD [examples](samples/build).

Env var | Path | Definition
--- | --- | --- |
**TEST_IMAGE_REPO** | **spec.output.image** | Registry endpoint to push images |
**TEST_IMAGE_REPO_SECRET** | **spec.output.credentials.name** | Registry endpoint secret(_usually of the type kubernetes.io/dockerconfigjson_) |

For running E2E tests for private repositories, the **TEST_WITH_PRIVATE_REPO** environment variable needs to be set to **true**.
If the Build private repositories [examples](test/data) contain references to private repositories you don´t have access, use
the following variables for any related modification in the examples.

Env var | Path | Definition
--- | --- | --- |
**TEST_WITH_PRIVATE_REPO** | _none_ | Enables running e2e tests for private repositories |
**BUILDAH_PRIV_GITHUB** | **spec.source.url** | Private URL for the Buildah samples of the form *git@github.com* |
**BUILDAH_PRIV_GITLAB** | **spec.source.url** | Private URL for the Buildah samples of the form *git@gitlab.com* |
**KANIKO_PRIV_GITHUB** | **spec.source.url** | Private URL  for the Kaniko samples of the form *git@github.com* |
**SRC_PRIV_GITHUB** | **spec.source.url** | Private URL for the src2img samples of the form *git@github.com* |
**BUILDPACKS_PRIV_GITHUB** | **spec.source.url** | Private URL for the buildpacks samples of the form *git@github.com* |
**TEST_SOURCE_SECRET** | **spec.source.credentials.name** | A secret containing the SSH private key for accessing the above private repository. See [ssh-authentication](https://github.com/tektoncd/pipeline/blob/master/docs/auth.md#ssh-authentication-git). The secret definition must define two annotations: `tekton.dev/git-0: github.com` and `tekton.dev/git-1: gitlab.com`  |

To run all strategies except buildpacks-v3 and none private git repositories tests, execute:

```sh
operator-sdk test local ./test/e2e --up-local --namespace build-examples
```

To run all strategies including buildpacks-v3 and none private git repositories tests, [setup your Quay credentials](samples/buildstrategy/buildpacks-v3#try-it-) and execute:

```sh
TEST_IMAGE_REPO=quay.io/shbose/nodejs-ex:latest TEST_IMAGE_REPO_SECRET=regcred  operator-sdk test local ./test/e2e --up-local --namespace build-examples
```

To run all strategies and also the private git repositories tests, execute:

```sh
export TEST_WITH_PRIVATE_REPO=true
export BUILDAH_PRIV_GITHUB=git@github.com:<youruser>/<your-repo>.git
export BUILDAH_PRIV_GITLAB=git@gitlab.com:<youruser>/<your-repo>.git
export KANIKO_PRIV_GITHUB=git@github.com:<youruser>/<your-repo>.git
export SRC_PRIV_GITHUB=git@github.com:<youruser>/<your-repo>.git
export BUILDPACKS_PRIV_GITHUB=git@github.com:<youruser>/<your-repo>.git
export TEST_SOURCE_SECRET=<your-github-ssh-all>
operator-sdk test local ./test/e2e --up-local --namespace build-examples --go-test-flags "-timeout=15m"
```

_Note:_ The e2e tests timeout defaults to 10min, when running with the private git repositories tests, more than 10 minutes is recommended.

For private git repositories test a secret of the type `kubernetes.io/ssh-auth` is required, here is an example of such a secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: github-ssh-all
  annotations:
    tekton.dev/git-0: github.com
    tekton.dev/git-1: gitlab.com
type: kubernetes.io/ssh-auth
data:
  ssh-privatekey: <cat ~/.ssh/id_rsa | base64>
```
