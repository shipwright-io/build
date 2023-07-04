<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

---
title: dedicated-shipwright-operator
authors:
  - "@adambkaplan"
reviewers:
  - "@SaschaSchwarze0"
  - "@zhangtbj"
  - "@gabemontero"
approvers:
  - "@qu1queee"
  - "@sbose78"
creation-date: 2021-02-03
last-updated: 2021-02-09
status: implementable
---

# Dedicated Shipwright Operator

**Build Enhancement Proposals have been moved into the Shipwright [Community](https://github.com/shipwright-io/community) repository. This document holds an obsolete Enhancement Proposal, please refer to the up-to-date [SHIP](https://github.com/shipwright-io/community/blob/main/ships/0011-dedicated-shipwright-operator.md) for more information.**

## Release Signoff Checklist

- [x] Enhancement is `implementable`
- [ ] Design details are appropriately documented from clear requirements
- [ ] Test plan is defined
- [ ] Graduation criteria for dev preview, tech preview, GA
- [ ] User-facing documentation is created in [docs](/docs/)

## Open Questions [optional]

TBD

## Summary

Create a separate operator container image that configures and deploys Shipwright Builds.
The operator will consume a Custom Resource Definition (CRD) which defines how Shipwright Builds should be configured.
The operator then reconciles the CRD spec with Shipwright Build's controller deployment.

The design has been inspired by the upstream [Tekton operator](https://github.com/tektoncd/operator).
As with Tekton, the Shipwright operator will provide one of many potential ways Shipwright Builds can be installed on a Kubernetes cluster.

## Motivation

This proposal resolves issue [#310](https://github.com/shipwright-io/build/issues/310) by creating a separate operator container image, which is responsible for installing Shipwright Builds on the cluster.
This will be a true operator, initialized by operator-sdk to create a bundle image.
The bundle image will always include the `ShipwrightBuild` CRD that controls how Shipwright Builds are configured.
For simplicity, the bundle image will also include the Build API CRDs as defined in (shipwright-io/build)[https://github.com/shipwright-io/build].

The Shipwright Build controllers can have some of their behavior modified via environment variables.
Altering and configuring these environment variables is currently a manual process and is described in various portions of the Shipwright Build documentation.
For example, exposed metrics labels can be adjusted by setting the `PROMETHEUS_ENABLED_LABELS` environment variable.

A separate operator can also facilitate the packaging of future projects that will fall under the Shipwright umbrella.
For example, the operator can deploy an optional HTTP server which makes the `shp` command line available for download within cluster, along with associated `Service` and `Ingress` objects to provide network access.
This is useful for Kubernetes clusters that run in air-gapped environments.

Operator Lifecycle Manager (OLM) is the preferred way to install Kubernetes extensions on distributions like OpenShift.
To deploy an application via an OLM-managed operator, the configuration of said application must be declared via a Custom Resource object.
The custom resource serves as an API to the application (also known as _operand_) configuration.
The operator can then report the state of the operand via the CR's status subresource.

OLM has an optional web console that can be deployed locally or packaged in a Kubernetes distribution.
With appropriately annotated custom resources, Shipwright Builds can be configured without the kubectl command line.

### Goals

1. Allow Shipwright Builds to be installed via OLM using a bundle image.
2. Allow Shipwright Builds to be configured via Kubernetes-native interface.

### Non-Goals

1. Package Shipwright Builds itself as a bundle image.
2. Package Shipwright Builds as Helm chart.
3. Package Shipwright Builds as an all-in-one manifest.

## Proposal

### User Stories

As a Kubernetes cluster administrator
I want to be able to install Shipwright Builds via an operator
So that I can provide Shipwright Builds to my developers

As a Kubernetes cluster administrator
I want to be able to configure the metrics published by Shipwright Builds
So that I can get a detailed understanding of how users are experiencing Shipwright

As a distributor of Shipwright Builds
I want to be able to configure the base image for the Runtime Image feature
So that I can use my own base image for lean runtime images.

### Implementation Details/Notes/Constraints [optional]

#### API

Shipwright Builds will be configured via the `ShipwrightBuild` custom resource.
This custom resource is cluster-scoped, and has the canonical name `cluster`.
All other instances of this kind will be ignored.

```yaml
apiVersion: operator.shipwright.io/v1alpha1
kind: ShipwrightBuild
metadata:
  name: cluster # canonical name
spec:
  targetNamespace: shipwright-build
  prometheus:
    enabledLabels:
    - buildstrategy
    - namespace
    - build
    buildRun:
      completionDurationBuckets: []
      establishDurationBuckets: []
      rampupDurationBuckets: []
      taskRunRampupDurationBuckets: []
  kanikoContainerImage: gcr.io/kaniko:latest
status:
  conditions:
  - Type: Available
    Status: "True"
    Reason: AsExpected
    Message: "Build controller manager is available."
  ... # other conditions as needed
```

Fields in the specification will reconcile to appropriate environment variables on the deployment for Shipwright Builds.
As future configuration options for Shipwright Builds are added, this API will likewise be extended.

#### Bootstrap behavior

When the operator first starts, it should check for the presence of the cluster `ShipwrightBuild` object.
If this object does not exist, the operator should boostrap a default instance with empty/default `spec` values.
Deleting the `ShipwrightBuild` instance should remove the associated deployment objects.
However, the Build API CRDs will remain on the cluster, as these will be managed by OLM.

At present, OLM does not delete CRDs and CRD instances if the associated operator is removed.
This is by design to ensure user data is not accidentally deleted - see [operator-framework/operator-lifecycle-manager#1326](https://github.com/operator-framework/operator-lifecycle-manager/issues/1326).

#### Installed Custom Resource Definitions

The operator will install the `ShipwrightBuild` custom resource definition.
OLM tooling takes care of this when we produce an appropriately structured bundle image.
Because the operator will bootstrap Shipwright Builds, the Build API CRDs will also be included in the bundle image.
This ensures the operator runs with the minimum privileges needed to create the build controller manager deployment.

In the future, the Build API CRDs can be removed from the bundle image and managed by the operator directly.
The operator would need full permissions over custom resource definitions in this scenario.
This would be useful if Project Shipwright produces additional components and cluster admins wish to remove Shipwright Builds.

### Risks and Mitigations

**Risk**: The operator provides one of potentially multiple avenues admins can install Shipwright.

*Mitigation*: Documentation will need to provide instructions on supported installation methods.

**Risk**: Manifests uses to deploy Shipwright Builds in the `shipwright-io/operator` are not synchronized with content in `shipwright-io/build`.

*Mitigation*: The Shipwright build controller's CI should include a test suite that runs the e2e tests in `shipwright-io/build`.
Project maintainers should also ensure that deployment changes to `shipwright-io/build` carry across to the Shipwright build controller.

**Risk**: Operator will require permissions cluster admins will reject (ex - modify CRDs)

*Mitigation*: For an initial implementation, the Build API CRDs will be installed via OLM using bundle image content.
If we want to make the installation of the Build API CRDs optional, we can create RBAC such that the operator can only modify CRDs in the `shipwright.io` API group.
This would require us to move the Build APIs to the `shipwright.io` group - see [shipwright-io/build#563](https://github.com/shipwright-io/build/issues/563).

## Design Details

### Test Plan

Test suites will need to ensure the following:

1. When a `ShipwrightBuild` object is created, the corresponding build controllers are deployed and the Build APIs are added as custom resource definitions.
2. Changes to the `ShipwrightBuild` spec are correctly reflected in the subsequent deployment.
3. CI for the Shipwright build controller should include the e2e suite for `shipwright-io/build`, run against the controllers deployed by the operator.

### Graduation Criteria

##### Dev Preview -> Tech Preview

- Configuration API for `ShipwrightBuild` reaches v1beta1 stability.
- Installation instructions and configuration options are well documented.
- Support for basic installation

##### Tech Preview -> GA

- Configuration API for `ShipwrightBuild` reaches v1 stability.
- Support for over the air upgrades of the Shipwright build controller.
- [optional] allow version skews between the Shipwright build controller and deployed version of Shipwright Build.
- [optional] Shipwright build controller manages the Build API CRDs.

### Upgrade / Downgrade Strategy

The Shipwright build controller should use leader election to ensure that when a new version of the operator is installed, it does not conflict with the existing installation.
This is only required for Tech Preview - Dev Preview releases can assume that the operator is uninstalled before the new version is installed.

## Implementation History

- 2020-02-03: Proposal
- 2020-02-09: Marked implementable

## Drawbacks

A separate operator adds overhead to the project, particularly with respect to synchronizing deployment manifests.
This can also add confusion if `shipwright-io/build` is made available via a Helm chart or "all in one" Kubernetes manifest.

To avoid confusion, we will also need to rename components in `shipwright-io/build` to remove references to "operator."
For instance, what we call the `build-operator` today should be renamed the `build-controller-manager`.

## Alternatives

There are other ways to simply install a project like Shipwright:

1. An install script (current approach)
2. An "all in one" Kubernetes YAML manifest
3. A Helm chart

However, these mechanisms are one-way installations.
Even Helm does not include mechanisms to ensure the applied chart is healty and functioning as expected.
Helm also will not install/upgrade CRDs if they are already present on the cluster.
See https://helm.sh/docs/chart_best_practices/custom_resource_definitions/

OLM-managed operators have the advantage that:

1. CRDs are upgraded with operator upgrades.
2. OLM operators can report the state of their _operands_ and act accordingly.

That said, this proposal does not exclude adding a Helm chart or "all in one" YAML manifest to Shipwright Builds as a part of its release process.
The latter YAML manifest approach may prove most practical, since can be used by any administrator with `kubectl`.

## Infrastructure Needed

1. Create a new GitHub repo to host the operator (ex - github.com/shipwright-io/operator)
   This will need CI configured to ensure proper branch protection and prow labels.
2. quay.io repositories for the operator image and bundle image. (quay.io/shipwright-io/operator, quay.io/shipwright-io/operator-bundle)
3. Rename the existing published image for shipwright-operator (rename to build-controller-manager or equivalent).
4. Ensure shipwright-io/build generates CRD manifests that can be consumed by the operator repository.
