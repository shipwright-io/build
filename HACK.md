
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

The following is a list of environment variables you can use when running e2e tests, this will override specific paths under the **Build** CRD [examples](samples/build).

| Environment Variable               | Path                           | Description                                         |
|------------------------------------|--------------------------------|-----------------------------------------------------|
| `TEST_NAMESPACE`                   | _none_                         | Target namespace to execute tests upon              |
| `TEST_IMAGE_REPO`                  | `spec.output.image`            | Image repository for end-to-end tests               |
| `TEST_IMAGE_REPO_SECRET`           | `spec.output.credentials.name` | Container credentials secret name                   |
| `TEST_IMAGE_REPO_DOCKERCONFIGJSON` | _none_                         | JSON payload equivalent to `~/.docker/config.json`  |

The contents of `TEST_IMAGE_REPO_DOCKERCONFIGJSON` can be obtained from [quay.io](quay.io) using a [robot account](https://docs.quay.io/glossary/robot-accounts.html). The JSON payload is for example:

```json
{ "auths": { "quay.io": { "auth": "<secret-credentials>" } } }
```

When both `TEST_IMAGE_REPO_SECRET` and `TEST_IMAGE_REPO_DOCKERCONFIGJSON` are informed, a new secret is created for end-to-end tests, named by  `TEST_IMAGE_REPO_SECRET`. However, when `TEST_IMAGE_REPO_DOCKERCONFIGJSON` is empty, e2e tests are expecting to find a pre-existing one.

To execute the end-to-end tests, run:

```sh
make test-e2e \
  TEST_NAMESPACE="default" \
  TEST_IMAGE_REPO_DOCKERCONFIGJSON="<JSON>"
```

Currently the end-to-end tests are not run in parallel, and may take several minutes to complete.

### Private Git Repositories

End-to-end tests can also be executed with the context of private Git repositories, using the following environment variables to configure it.

| Environment Variable  | Path                           | Description                           |
|-----------------------|--------------------------------|---------------------------------------|
| `TEST_PRIVATE_REPO`   | _none_                         | Enable private repository e2e tests   |
| `TEST_PRIVATE_GITHUB` | `spec.source.url`              | Private URL, like `git@github.com`    |
| `TEST_PRIVATE_GITLAB` | `spec.source.url`              | Private URL, like `git@gitlab.com`    |
| `TEST_SOURCE_SECRET`  | `spec.source.credentials.name` | Private repository credentials        |

On using `TEST_SOURCE_SECRET`, the environment variable must contain the name of the Kubernetes Secret containing SSH private key, for given private Git repository. See [ssh-authentication](https://github.com/tektoncd/pipeline/blob/master/docs/auth.md#ssh-authentication-git). The secret definition must define the following annotations:
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
