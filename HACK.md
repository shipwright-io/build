<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->


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

### Counterfeiter

Counterfeiter is used to generate and update fake implementations of objects. Currently only used for the `manager` and `client` package interface of the `sigs.k8s.io/controller-runtime`.

This allow us to use test doubles in the unit tests, from where we can instantiate the fakes and then stub return values. This is very useful, for example we can mock all **client** calls that happened during the k8s controllers reconciliation and stub the result. Same case for the **manager** when creating controllers.

For installing the binary, refer to [installing counterfeiter](https://github.com/maxbrunsfeld/counterfeiter#installing-counterfeiter-to-gopathbin).

### Ginkgo

Ginkgo is a go testing framework that use the Go´s **testing** package. The framework have many [features](https://github.com/onsi/ginkgo#feature-list) on top of Go´s built-in testing primitives.

In ginkgo, every package needs a `suite_test.go`. Each controller package under this repository will have one. You can generate a suite by running `ginkgo bootstrap` under the package directory. For testing an specific controller class, you can generate the testing class by running `ginkgo generate` under the package directory.

When building unit-tests, try to follow:

* Test DRY. Therefore we use the `catalog.go` helper class under the `test` directory, to avoid code repetition.
* Use counterfeiter to generate fakes.
* Tests happen on a separate `_test` package.
* Assert all errors.
* Assert that function invocations generate the expected values.

### Running unit-tests

```sh
make test
```

## End-to-End Tests

The following table contains a set of environment variables that control the behavior of the e2e tests.

| Environment Variable            | Default                                                                                          | Description                                                   |
|---------------------------------|--------------------------------------------------------------------------------------------------|---------------------------------------------------------------|
| `TEST_NAMESPACE`                | `default`                                                                                        | Target namespace to execute tests upon, default is `default`. |
| `TEST_E2E_FLAGS`                | `-failFast -flakeAttempts=2 -p -randomizeAllSpecs -slowSpecThreshold=300 -timeout=20m -trace -v` | Ginkgo flags. See all Ginkgo flags here: [The Ginkgo CLI](https://onsi.github.io/ginkgo/#the-ginkgo-cli). Especially of interest are `--focus` and `--skip` to run selective tests. |
| `TEST_E2E_CREATE_GLOBALOBJECTS` | `true`                                                                                           | Boolean, if `false`, the custom resource definitions and (cluster) build strategies are not created and deleted by the e2e test code |
| `TEST_E2E_OPERATOR`             | `start_local`                                                                                    | String with allowed values `start_local` (will start the local operator and print its logs add the end of the test run) and `managed_outside` (will assume that the operator is running through whatever means) |
| `TEST_E2E_TIMEOUT_MULTIPLIER`   | `1`                                                                                              | Multiplier for timeouts in the e2e tests to run them on slower systems. |
| `TEST_E2E_VERIFY_TEKTONOBJECTS` | `true`                                                                                           | Boolean, if false, the verification code will not try to verify the TaskRun object status |

The following table contains a list of environment variables you that will override specific paths under the **Build** CRD.

| Environment Variable               | Path                           | Description                                                   |
|------------------------------------|--------------------------------|---------------------------------------------------------------|
| `TEST_IMAGE_REPO`                  | `spec.output.image`            | Image repository for end-to-end tests                         |
| `TEST_IMAGE_REPO_SECRET`           | `spec.output.credentials.name` | Container credentials secret name                             |
| `TEST_IMAGE_REPO_DOCKERCONFIGJSON` | _none_                         | JSON payload equivalent to `~/.docker/config.json`            |

The contents of `TEST_IMAGE_REPO_DOCKERCONFIGJSON` can be obtained from [quay.io](quay.io) using a [robot account](https://docs.quay.io/glossary/robot-accounts.html). The JSON payload is for example:

```json
{ "auths": { "quay.io": { "auth": "<secret-credentials>" } } }
```

When both `TEST_IMAGE_REPO_SECRET` and `TEST_IMAGE_REPO_DOCKERCONFIGJSON` are informed, a new secret is created for end-to-end tests, named by `TEST_IMAGE_REPO_SECRET`. However, when `TEST_IMAGE_REPO_DOCKERCONFIGJSON` is empty, e2e tests are expecting to find a pre-existing one.

The following table contains a list of environment variables that will override specific paths under the **BuildRun** CRD.

| Environment Variable          | Path                   | Description |
|--------------------------------|-----------------------|-------------|
| `TEST_E2E_SERVICEACCOUNT_NAME` | `spec.serviceAccount` | The name of the service account used by the build runs, the code will try to create the service account but not fail if it already exists. Special value is `generated`, which will lead to using the auto-generation feature for each build run. |

To execute the end-to-end tests, run:

```sh
make test-e2e \
  TEST_NAMESPACE="default" \
  TEST_IMAGE_REPO="<image-repository>" \
  TEST_IMAGE_REPO_DOCKERCONFIGJSON="<JSON>"
```

Currently the end-to-end tests are not run in parallel, and may take several minutes to complete.

### KinD

When using [KinD](https://kind.sigs.k8s.io/) like jobs in Travis-CI, you can use a local container registry to store images created during end-to-end test execution. Run:

```sh
make kind
make test-e2e TEST_IMAGE_REPO="$(./hack/install-registry.sh show):5000/shipwright-io/build-e2e"
```

You only need to execute `make kind` once, `make test-e2e ...` can be repeated many times.

### Private Git Repositories

End-to-end tests can also be executed with the context of private Git repositories, using the following environment variables to configure it.

| Environment Variable  | Path                           | Description                           |
|-----------------------|--------------------------------|---------------------------------------|
| `TEST_PRIVATE_REPO`   | _none_                         | Enable private repository e2e tests   |
| `TEST_PRIVATE_GITHUB` | `spec.source.url`              | Private URL, like `git@github.com`    |
| `TEST_PRIVATE_GITLAB` | `spec.source.url`              | Private URL, like `git@gitlab.com`    |
| `TEST_SOURCE_SECRET`  | `spec.source.credentials.name` | Private repository credentials        |

On using `TEST_SOURCE_SECRET`, the environment variable must contain the name of the Kubernetes Secret containing SSH private key, for given private Git repository. See the [docs](/docs/development/authentication.md) for more information about authentication methods in the Build.

The secret definition must define the following annotations:
- `tekton.dev/git-0: github.com`
- `tekton.dev/git-1: gitlab.com`

To run end-to-end tests which also includes private Git repositories, run:

```sh
make test-e2e \
  TEST_NAMESPACE="default" \
  TEST_PRIVATE_REPO="true" \
  TEST_PRIVATE_GITHUB="git@github.com:<youruser>/<your-repo>.git" \
  TEST_PRIVATE_GITLAB="git@gitlab.com:<youruser>/<your-repo>.git" \
  TEST_SOURCE_SECRET="<secret-name>"
```

For private Git repositories tests, a secret of the type `kubernetes.io/ssh-auth` is required, here is an example:

```yml
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
