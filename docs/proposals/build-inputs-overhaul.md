<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

---
title: build-inputs-overhaul
authors:
  - "@adambkaplan"
  - "@otaviof"
reviewers:
  - TBD
approvers:
  - TBD
creation-date: yyyy-mm-dd
last-updated: yyyy-mm-dd
status: provisional 
replaces:
  - "/docs/proposals/remote-artifacts.md"
---

# Build Inputs Overhaul

## Release Signoff Checklist

- [ ] Enhancement is `implementable`
- [ ] Design details are appropriately documented from clear requirements
- [ ] Test plan is defined
- [ ] Graduation criteria for dev preview, tech preview, GA
- [ ] User-facing documentation is created in [docs](/docs/)

## Open Questions [optional]

TBD

## Summary

Overhaul the Build API to separate build inputs from build execution and outputs.
This overhaul will provide a path to supporting the following:

1. Multiple build sources
2. Build volume mounts

## Motivation

Building a container image can require more than one type of source as code/input.
Builds may also rely on caches and other bits of information that are needed as input, but should not be present in the resulting container image.
The current build API does not satisfy these two needs, and the current structure is not amenable to future adaptation.

This proposal seeks to reorganize the Build API specification into three groups:

1. Input
2. Build (execution)
3. Output

Do so will allow Shipwright Builds to support multiple source inputs as well as mounts in builds.

### Goals

- Allow developers to assemble applications from multiple sources
- Allow developers to mount volumes within the build

### Non-Goals

- Provide specific implementations for a given volume mount.
- Provide additional source types beyond the ones supported today.

## Proposal

This is where we get down to the nitty gritty of what the proposal actually is.

### User Stories [optional]

Detail the things that people will be able to do if this is implemented. Include as much detail as
possible so that people can understand the "how" of the system. The goal here is to make this feel
real for users without getting bogged down.

#### Story 1

#### Story 2

### Implementation Details/Notes/Constraints [optional]

The Build API will serve as an abstraction layer for Tekton workspaces that are created in the associated TaskRun.
The new `input` section of the build spec will declare how the Tekton workspaces for the BuildRun are configured.
These workspaces will allow information to be shared across steps in the BuildRun.

#### New API

```yaml
spec:
  input:
    sources:
    - name: git
      type: Git
      git:
        url: https://github.com/shipwright-io/build.git
        ref: main
        credentials:
          name: git-creds
    - name: jars
      type: HTTP
      http:
        url: https://my-artifacts.corp/bin/base-lib.jar
      destination: bin/jars/ # Destination must be relative
    - name: dockerfile
      type: InlineFiles # This is a hypothetical type to let the Dockerfile be defined inline.
      files:
        Dockerfile: |
          FROM busybox
          RUN echo "Hello world!"
      overwrite: true # Option to allow source code to be overwritten
    mounts:
    - name: maven
      persistentVolumeClaim:
        name: shared-m2 # Consume a specific PVC
      volumeSubPath: .m2
      mountPath: "m2"
    - name: output-cache
      volumeClaimTemplate: # Create a PVC from a template
        spec:
          accessModes:
          - ReadWriteMany
          - ReadWriteOnce
          resources:
            requests:
              storage: 100Gi
      mountPath: /var/lib/containers/storage
    - name: trusted-ca
      configMap:
        name: trusted-ca
      mountPath: /etc/pki/ca-trust/extracted/pem
  build:
    contextDir: <context-dir> # moved from git source
    strategy:
    ...
    parameters:
    ...
    runtime: <>
    timeout: 5m
  ...
  output: # moved 
    image: quay.io/shipwright/controller:latest
    credentials:
      name: push-secret
  ...
```

#### Build Input Sources

The `sources` array defines a list of supported mechanisms to add source code or artifacts to the build.
Sources are code or artifacts that are needed to assemble the application and appear in the resulting container image.
All source code will be stored in the `source` workspace, mounted in `/workspaces/source` inside the BuildRun.
This will be backed by an `emptyDir` volume, which allows users to declare ephemeral storage requests/limits.

Build sources will be downloaded in the order presented in the array.
Code or artifacts will be saved to the `destination` directory, which must be a relative path.
If no destination is specified, `/workspace/source` is assumed to be the destination.
Source download should fail if the downloaded files/artifacts overwrite existing files, unless `overwrite: true` is set.

Entries in the `sources` array can be one of several types Shipwright supports for source code and artifacts.
Source types must be implement the `destination` and `overwrite` parameters.

The `contextDir` parameter that exists in the original Build `source` spec is moved to the `spec.build` portion

#### Build Input Mounts

Build mounts - unlike sources - are for content that is not meant to directly appear in the resulting container image.
Mounts can be used for container image caches, artifact caches (ex - maven), and runtime specific configuration (ex - certificate authorities).

Mounts must have a name and mountPath.
For simplicity, `mountPath` must be an absolute path (unlike Tekton, which allows relative paths).

Mounts must have a type that is supported by Tekton workspaces. Current supported types are:

- PersistentVolumeClaim
- VolumeClaimTemplate
- ConfigMap
- Secret
- EmptyDir

#### spec.build

`spec.build` is a new object that will contain the execution details of the build.
This consists of existing API that has been reorganized to clarify these settings control the execution of the build.
This will include:

- The build strategy
- The builder image used (if this is supported)
- Runtime parameters passed to the build strategy
- The build context directory - previously the `contextDir` for the single git source code
- The runtime image
- The build timeout

#### Build Output

`output` will remain a top-level specification, with the instructions on where the resulting image should be tagged and pushed.

### Risks and Mitigations

What are the risks of this proposal and how do we mitigate. Think broadly. For example, consider
both security and how this will impact the larger ecosystem.

How will security be reviewed and by whom? How will UX be reviewed and by whom?

Consider including folks that also work outside your immediate sub-project.

## Design Details

### Test Plan

**Note:** *Section not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:

- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy. Anything that would count as
tricky in the implementation and anything particularly challenging to test should be called out.

All code is expected to have adequate tests (eventually with coverage expectations).

### Graduation Criteria

**Note:** *Section not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. Initial proposal should keep
this high-level with a focus on what signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:

- Maturity levels - `Dev Preview`, `Tech Preview`, `GA`
- Deprecation

Clearly define what graduation means.

#### Examples

These are generalized examples to consider, in addition to the aforementioned [maturity
levels][maturity-levels].

##### Dev Preview -> Tech Preview

- Ability to utilize the enhancement end to end
- End user documentation, relative API stability
- Sufficient test coverage
- Gather feedback from users rather than just developers

##### Tech Preview -> GA

- More testing (upgrade, downgrade, scale)
- Sufficient time for feedback
- Available by default

**For non-optional features moving to GA, the graduation criteria must include end to end tests.**

##### Removing a deprecated feature

- Announce deprecation and support policy of the existing feature
- Deprecate the feature

### Upgrade / Downgrade Strategy

If applicable, how will the component be upgraded and downgraded? Make sure this is in the test
plan.

Consider the following in developing an upgrade/downgrade strategy for this enhancement:

- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to
  make on upgrade in order to keep previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to
  make on upgrade in order to make use of the enhancement?

### Version Skew Strategy

How will the component handle version skew with other components? What are the guarantees? Make sure
this is in the test plan.

Consider the following in developing a version skew strategy for this enhancement:

- During an upgrade, we will always have skew among components, how will this impact your work?
- Does this enhancement involve coordinating behavior in the control plane and in the kubelet? How
  does an n-2 kubelet without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI, CRI or CNI may require
  updating that component before the kubelet.

## Implementation History

Major milestones in the life cycle of a proposal should be tracked in `Implementation History`.

## Drawbacks

The idea is to find the best form of an argument why this enhancement should _not_ be implemented.

## Alternatives

Similar to the `Drawbacks` section the `Alternatives` section is used to highlight and record other
possible approaches to delivering the value proposed by an enhancement.

## Infrastructure Needed [optional]

Use this section if you need things from the project. Examples include a new subproject, repos
requested, github details, and/or testing infrastructure.

Listing these here allows the community to get the process for these resources started right away.
