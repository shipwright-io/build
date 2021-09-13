<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

---
title: buildstrategy
authors:
  - "@sbose78"
  - "@qu1queee"
status: design
---

# The BuildStrategy API

**Build Enhancement Proposals have been moved into the Shipwright [Community](https://github.com/shipwright-io/community) repository. This document holds an obsolete Enhancement Proposal, please refer to the up-to-date [SHIP](https://github.com/shipwright-io/community/blob/main/ships/0003-buildstrategy.md) for more information.**

## Goals

### User-defined build strategies

Users and enterprises have strong opinions on how to build container images from source code.
This project aims to enable admins to define build strategies for building container images in a Kubernetes cluster.

### Accomplish more by specifying less!

A slim BuildStrategy is one where the BuildStrategy author gets to accomplish more by specifying less. Without enabling authors to do so,
we would be inadvertently making it hard to define BuildStrategies, thereby undermining the very premise of simplicity that this project  aims to provide with the BuildStrategy CRD.

For example, the author of the build strategy should not have to specify how to push images to remote registries.

### Simplicity

This project uses the `TaskRun` API under-the-hood to execute an image build without leaking abstraction. The complexity of defining
a BuildStrategy must be less than that of defining a Tekton `Task`.

## Defining a BuildStrategy

A BuildStrategy CR is defined as a list of [corev1.Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.11/#container-v1-core)
execution steps.

### What must be specified

The author of the BuildStrategy CR may thereby specify the command(s) that need to be executed inside a container to convert source code
into a container image.

As an example, the build author might wish to express the following commands as container steps to do a [buildpacks-v3](https://github.com/buildpacks/lifecycle) image build:

* detector - chooses buildpacks (via /bin/detect)
* analyzer - restores launch layer metadata from the previous build
* builder - executes buildpacks (via /bin/build)

### What need not be specified

The Build controller should take care of executing build steps common to all strategies without making it mandatory
for users to specify 'well-known' steps.

The BuildStrategy author need not necessarily specify:

* How the image is to be pushed to a registry.
* Where the root CA certificates are to be picked up for communicating with secure registries.
* How to generate lean runtime images from the built image.
* Or, anything that can be classified as common or popular activity in an image build flow.

From an implementation perspective, well-known `BuildStrategy` steps are `Tekton` `Task` steps that are dynamically generated on-the-fly.

### Optional overrides

If the BuildStrategy author wishes to be explicit about the "how" of pushing an image to registry, the author should be able
express that.

The BuildStrategy author could convey the same to the controller by annotating the `BuildStrategy` with:

 ```sh
 buildstrategy.build.dev/contains-image-push: true
 buildstrategy.build.dev/contains-runtime-image: true
 ```

### Parameterization

Attributes from the `Build` CR could be used as parameters while defining a `BuildStrategy`.

Examples:

* `$(build.output.image)` implies that the `Build` CR's `spec.output.image` value be used.

* `$(build.parameters.skip_ssl_verify)` implies that the value corresponding to key `skip_ssl_verify` in `Build` CR's `spec.parameters` be used.

### Consequences

At the time of writing this proposal, we've already had `BuildStrategies` defined in `/samples` where the `BuildStrategy` author has expressed [corev1.Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.11/) steps to build images and push them to a registry.

Here's what it means to adopt this proposal:

1. Explore the possibility of removing the image push step in all `BuildStrategies`.  

2. Explore the possibility of generating lean runtime images if the user provides the runtime base image information in the `Build` CR.

3. If the above experiments succeed, ensure that **Optional overrides** as stated in the previous section, are supported.
