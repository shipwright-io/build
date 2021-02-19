<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

# Testing guide

- [Overview](#overview)
- [Ginkgo](#ginkgo)
- [Verifying your code](#verifying-your-code)
  - [Counterfeiter](#counterfeiter)
- [Unit Tests](#unit-tests)
- [Integration Tests](#integration-tests)
  - [Running integration tests](#running-integration-tests)
- [E2E Tests](#e2e-tests)
  - [General test parameters](#general-test-parameters)
  - [Build override parameters](#build-override-parameters)
  - [BuildRun override parameters](#buildrun-override-parameters)
  - [Private Git override parameters](#private-git-override-parameters)
  - [Running e2e tests](#running-e2e-tests)
  - [Running e2e tests with local registry](#running-e2e-tests-with-local-registry)
  - [Running e2e tests with private git repositories](#running-e2e-tests-with-private-git-repositories)

## Overview

Before opening a pull requests, please ensure that your changes are passing unit, integration, and verification tests.
In the following sections, the three levels of tests we cover are explained in detail.
Our testing implementation follows the Kubernetes community recommendations, see the [community docs](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/testing.md) for more information.

## Ginkgo

For all the testing levels, we rely on Gingko as the testing framework for defining test cases and executing them. Ginkgo is a go testing framework that use the Go´s **testing** package. The framework have many [features](https://github.com/onsi/ginkgo#feature-list) on top of Go´s built-in testing primitives.

## Verifying your code

Our Travis builds verify that your code conforms to project standards and that all generated code is up to date.
When adding files or updating APIs, please run `make generate` before submitting your code.

### Counterfeiter

Counterfeiter is used to generate and update fake implementations of objects. Currently only used for the `manager` and `client` package interface of the `sigs.k8s.io/controller-runtime`.

This allow us to use test doubles in the unit tests, from where we can instantiate the fakes and then stub return values. This is very useful, for example we can mock all **client** calls that happened during the k8s controllers reconciliation and stub the result. Same case for the **manager** when creating controllers.

Counterfeiter is required by the code generator scripts. Run `make install-counterfeiter` to add counterfeiter to your `GOPATH`.

### Static code analysis and linting

Run `make sanity-check` to run a collection of static code analyser and linters to check the code for issues, for example ineffective assignments, unused variables, missing comments, misspellings and so on. Each check also has an individual Make target to check:

- `make govet` examines Go source code and reports suspicious constructs
- `make ineffassign` checks Go source for variable assignments that are not used (i.e. overridden)
- `make golint` runs a linter against the Go source
- `make misspell` checks for TYPOs
- `make staticcheck` performs more complex static code analysis to find unused code and other issues

## Unit Tests

We use unit tests to provide coverage and ensure that our functions are behaving as expected, but also to assert the behaviour of the controllers during Reconciliations.

Unit tests are designed based on the following:

- They are fully hermetic, no calls to any k8s API.
- Required client mocks to simulate k8s API calls.
- All controller packages have a unit-test class.
- Unit tests must pass in different OS distributions( e.g. linux, macOS ).
- Unit tests should be run in parallel.

Because we use Ginkgo for this, each controller [package](https://github.com/shipwright-io/build/tree/master/pkg/reconciler) requires a `suite_test.go` file and a relative controller test file. You can generate a suite by running `ginkgo bootstrap` under the package directory. For testing an specific controller class, you can generate the testing class by running `ginkgo generate` under the package directory.

When building unit-tests, try to follow:

- Test DRY. Therefore we use the `catalog.go` helper class under the `test` directory, to avoid code repetition.
- Use counterfeiter to generate fakes.
- Tests happen on a separate `_test` file.
- Assert all errors.
- Assert that function invocations generate the expected values.

To run the unit tests, run `make test` from the command line.

## Integration Tests

We use integration tests to ensure all the interactions that happen between different resources are behaving as expected. These integrations go from Build Custom Resources instances, Tekton Custom Resources instances till Kubernetes primitive resources (e.g. secrets, service-accounts and pods.)

Integration tests are designed based on the following:

- All significant features should have an integration test.
- They require to have access to a Kubernetes cluster.
- Each test generates its own instance of the build controller, namespace and resources.
- After test are executed, all generated resources for the particular test are removed.
- They test all the interactions between components that have a relationship.
- They do not test an e2e flow.

### Running integration tests

Before running these tests, ensure you have:

- A running cluster. You can use Kind, see [installation](https://github.com/shipwright-io/build/blob/master/hack/install-kind.sh)
- Tekton controllers installed, see [installation](https://github.com/shipwright-io/build/blob/master/hack/install-tekton.sh)

```sh
make test-integration
```

## E2E Tests

We use e2e tests as the last signal to ensure the controllers behaviour in the cluster matches the developer specifications( _based on [e2e-tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/e2e-tests.md)_ ). During e2e tests execution, we don´t want to test any interaction between components but rather we want to simulate a normal user operation and ensure that images are successfully build. E2E tests should only cover:

- As the way to validate if the image was successfully build, only assert for a Succeeded Status on TaskRuns.
- Testing should be around building images with different supported strategies, and different runtimes inside the strategies.

### General test parameters

The following table contains a set of environment variables that control the behavior of the e2e tests.

| Environment Variable            | Default                                                                                          | Description                                                   |
|---------------------------------|--------------------------------------------------------------------------------------------------|---------------------------------------------------------------|
| `TEST_NAMESPACE`                | `default`                                                                                        | Target namespace to execute tests upon, default is `default`. |
| `TEST_E2E_FLAGS`                | `-failFast -flakeAttempts=2 -p -randomizeAllSpecs -slowSpecThreshold=300 -timeout=20m -trace -v` | Ginkgo flags. See all Ginkgo flags here: [The Ginkgo CLI](https://onsi.github.io/ginkgo/#the-ginkgo-cli). Especially of interest are `--focus` and `--skip` to run selective tests. |
| `TEST_E2E_CREATE_GLOBALOBJECTS` | `true`                                                                                           | Boolean, if `false`, the custom resource definitions and (cluster) build strategies are not created and deleted by the e2e test code |
| `TEST_E2E_OPERATOR`             | `start_local`                                                                                    | String with allowed values `start_local` (will start the local operator and print its logs add the end of the test run) and `managed_outside` (will assume that the operator is running through whatever means) |
| `TEST_E2E_TIMEOUT_MULTIPLIER`   | `1`                                                                                              | Multiplier for timeouts in the e2e tests to run them on slower systems. |
| `TEST_E2E_VERIFY_TEKTONOBJECTS` | `true`                                                                                           | Boolean, if false, the verification code will not try to verify the TaskRun object status |

### Build override parameters

The following table contains a list of environment variables that will override specific paths under the **Build** CRD.

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

### BuildRun override parameters

The following table contains a list of environment variables that will override specific paths under the **BuildRun** CRD.

| Environment Variable          | Path                   | Description |
|--------------------------------|-----------------------|-------------|
| `TEST_E2E_SERVICEACCOUNT_NAME` | `spec.serviceAccount` | The name of the service account used by the build runs, the code will try to create the service account but not fail if it already exists. Special value is `generated`, which will lead to using the auto-generation feature for each build run. |

### Private Git override parameters

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

### Running e2e tests

To execute the end-to-end tests, run:

```sh
make test-e2e \
  TEST_NAMESPACE="default" \
  TEST_IMAGE_REPO="<image-repository>" \
  TEST_IMAGE_REPO_DOCKERCONFIGJSON="<JSON>"
```

_Note_: Currently the end-to-end tests are not run in parallel, and may take several minutes to complete.

### Running e2e tests with local registry

When using [KinD](https://kind.sigs.k8s.io/) like jobs in Travis-CI, you can use a local container registry to store images created during end-to-end test execution. Run:

```sh
make kind
make test-e2e TEST_IMAGE_REPO="$(./hack/install-registry.sh show):5000/shipwright-io/build-e2e"
```

You only need to execute `make kind` once, `make test-e2e ...` can be repeated many times.

### Running e2e tests with private git repositories

To run end-to-end tests which also includes private Git repositories, run:

```sh
make test-e2e \
  TEST_NAMESPACE="default" \
  TEST_PRIVATE_REPO="true" \
  TEST_PRIVATE_GITHUB="git@github.com:<youruser>/<your-repo>.git" \
  TEST_PRIVATE_GITLAB="git@gitlab.com:<youruser>/<your-repo>.git" \
  TEST_SOURCE_SECRET="<secret-name>"
```
