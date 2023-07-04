<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

---

title: build-execution-using-build-run
authors:

- "@sbose78"
- "@zhangtbj"

status: design

---

# Build execution using the BuildRun API

**Build Enhancement Proposals have been moved into the Shipwright [Community](https://github.com/shipwright-io/community) repository. This document holds an obsolete Enhancement Proposal, please refer to the up-to-date [SHIP](https://github.com/shipwright-io/community/blob/main/ships/0002-build-execution-using-build-run.md) for more information.**

## About

A `BuildRun` is an immutable CR/object that represents a single execution of the `Build`.

The simplest `BuildRun` looks like this:

```yaml
apiVersion: build.dev/v1alpha1
kind: BuildRun
metadata:
  generateName: account-service-build-
spec:
  buildRef:
    name: kaniko-golang-build
```

The `BuildRun` API also provides a way to override/specify execution time information.

As an example, a `Build` configuration should not have to accurately specify the resource requirements - the information on resource requirements is valuable at build execution time which varies from environment to environment.

## Defining a BuildRun

The `BuildRun`, apart from triggering a build execution, also provides the user an opportunity to specify/override a subset of Build configuration which are likely to change at build execution time.

### What is deterministic

#### What are we building

- The source code that's being built into an image.
- The application binary that's being packaged into an image.

The above information wraps the integrity of the code
that we are building into an image.

#### How are we building

- The `BuildStrategy` being used to build the image.
- Inputs associated with the `BuildStrategy` : Dockerfile, builder image, etc.
- The runtime base image being used to build a lean image.

The above information wraps the know-how that drives the process of converting source code into an image.

### What is non-deterministic

- The Image being pushed to
  - Repository
  - Tag
- The Service Account used to execute the build.
- Execution-time resource requiements.

As an example, the source code [nodejs-rest-http-crud](https://github.com/nodeshift-starters/nodejs-rest-http-crud) might be pushed to different image repositories depending on who is executing the build.

Note:
In some scenarios, the source code 'revision' may not necessarily be deterministic as well, especially in case of builds triggered from PRs/forks.

### Deciding what should be overridden

- Could the value of attribute X be reasonably determined at the time of `Build` configuration specification ?
- Could the modification of attribute X reasonably compromise the integrity of the build ?
- Could the value of attribute X reasonably differ in the context of a clusters or a namespace ?

The above is a non-exhaustive list of questions that should help us modify the BuildRun API in future.

### Next steps / Consequences

Here's what it means to adopt this proposal:

1. Make `spec.output` optional in `Build` API's `spec` .
2. Introduce `spec.output` in the `BuildRun` API's `spec`.
3. Make the `BuildRun` immutable.
