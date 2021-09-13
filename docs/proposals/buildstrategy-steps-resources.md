<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

---
title: Build Strategies steps resource limitations
authors:
  - "@qu1queee"
  - "@xiujuan95"
  - "@SaschaSchwarze0"
  - "@zhangtbj"
  - "@gabemontero"
reviewers:
  - TBD
approvers:
  - TBD
creation-date: 2020-06-15
last-updated: 2020-06-15
status: implementable
see-also:
  - "/docs/proposals/buildstrategy-steps-resources.md"
---

# Build Steps Resource Limitations

**Build Enhancement Proposals have been moved into the Shipwright [Community](https://github.com/shipwright-io/community) repository. This document holds an obsolete Enhancement Proposal, please refer to the up-to-date [SHIP](https://github.com/shipwright-io/community/blob/main/ships/0004-buildstrategy-steps-resources.md) for more information.**

## Release Signoff Checklist

- [x] Enhancement is `implementable`
- [x] Design details are appropriately documented from clear requirements
- [x] Test plan is defined
- [ ] Graduation criteria for dev preview, tech preview, GA
- [x] User-facing documentation is created in [docs](/docs/)

## Open Questions [optional]

_**Note**_: Some notions around the role of an `end-user`. In development environments (_e.g. kind cluster, personal k8s cluster_) the end-user plays also the role of `strategy admin`, because he/she needs to deploy the controller and install the Build/ClusterBuildStrategy CRDs instances. For production environments(_e.g. IKS, Openshift_), an end-user will not play the role of the `strategy admin`, so he/she knows nothing about the internals of the Build/ClusterBuildStrategy CRDs instances.

> 1. Should end-users have fine-grained control on resource definitions for the different build steps (containers) of a build strategy?

## Summary

The current implementation of how resources (_e.g. CPU, Memory_) are applied to the Build Strategy steps have a flaw. This current implementation allows anyone to define resources via the `Build` or `BuildRun` CRD instance, by embedding them under the `spec.resources` path.
The flaw is that the resources numbers, apply to all of the steps(_containers_) listed on the related Build/ClusterBuild Strategy, hoarding resources. This ends up with a higher number for resources consumption than what the user initially defined in the `Build` instance. For example, if users define `500m` as CPU request, but the strategy consists of five steps, then the used
resources will be `500m` multiplied by `5`. This of course, can have implications for billing.

Also, with the current `spec.resources` API we have in the `Build`, we are constrain to only define a single set of values for resources (_e.g. memory, cpu, etc_). This is not flexible while
there could be situations where containers under the same strategy required different resource values (_not sharing the same values_). This raise the following questions:

- First, why do we have in the `Build`/`BuildRun` an API to define `resources` for containers, if the end-users neither know the number of containers for a particular strategy, nor have insights into what each of these containers is doing.

- Second, if a strategy consists of five steps(_containers_) and because of X reason I need to define five different values for each container CPU, how could I achieve this via the current `spec.build` API. (_not flexible_)

## Motivation

For strategies with multiple steps like Buildpacks, not all the steps (_containers_) will require the same set of resources (_e.g. memory, cpu, etc_) values.
In order to be able to manage situations where setting particular steps resources is required, we need a more flexible way to deal with N number of steps under the same strategy (_Build/ClusterBuildStrategy_). Also, we still need to help users to know that independently of the abstraction of the strategies, there are options for speeding up the builds. This options could be presented in the form of different `flavours` of the same strategy (_cluster or namespaced scope_), that differ between each other in terms of container resources values.

### Goals

- Allow end-users to focus only on the Build definition by abstracting the building mechanism from them. We can achieve this by keeping them away of the strategies content and the need of defining parameters(_like resources_) for them.

- Allow `strategy admins` to set different flavours of the **same** strategies. This is the responsibility of the `strategy admin`, and these flavours may vary depending on the Cloud provider. Recommendations of different flavours for each strategy could be made, based on the steps and the notion around, which of them consume higher numbers for cpu/memory.

- Set `default` strategies flavours in this repository, so that all strategy containers have limits on the resources they consume, avoiding OOM and CPU throttling when running in Travis.

### Non-Goals

None.

## Proposal

Ensure that strategies steps call out the resource requirements and then our internal TaksRun generation acknowledge it.

### Implementation

This proposal will consist of different steps:

1. Remove the `spec.resources` from the `Build` and `BuildRun` API.

2. Remove the logic inside the `generate_taskrun.go` and test classes, where we parse resources defined in the `Build/BuildRun` into the strategy containers. Ensure `resources` of the strategies will actually be processed and passed on runtime to the generated `TaskRun` from Tekton

3. For all the current strategies(_BuildStrategy or ClusterBuildStrategy_) in this repository (_samples_), we should define default values for the container resources limits/request so that our `travis` will not be affected by strategy containers consuming all the CPU and memory (_unlimited_). Also, update all of our sample `Build` YAMLs to use this strategies.

4. Properly document under the strategies docs, that all of the steps resources can be modified by the `strategy admin`. If a difference of values per strategy is needed, then multiple `ClusterBuildStrategy` will be required to exist on the cluster, with different names.

5. Document different flavours for each `ClusterBuildStrategy` (_e.g. kaniko, buildpacks_). Each flavour should carefully illustrate how it differentiate with the default `ClusterBuildStrategy` values, and which advantage one can have when choosing this flavour under their `Build` resource. Advantages should be for now `faster builds`.

### Example

To illustrate these changes from the perspective of the end-user, lets take a look on the default scenario:

```yaml
---
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: buildpack-nodejs-build
  annotations:
    build.build.dev/build-run-deletion: "false"
spec:
  source:
    url: https://github.com/sclorg/nodejs-ex
  strategy:
    name: buildpacks-v3-default <---- PAY ATTENTION TO THIS DEFAULT STRATEGY FLAVOUR
    kind: ClusterBuildStrategy
....
```

If an `end-user` will require a different flavour for `buildpacks` strategies because he/she wants faster builds, then the `Build` definition will look like:

```yaml
---
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: buildpack-nodejs-build
  annotations:
    build.build.dev/build-run-deletion: "false"
spec:
  source:
    url: https://github.com/sclorg/nodejs-ex
  strategy:
    name: buildpacks-v3-medium <---- PAY ATTENTION TO THIS MEDIUM STRATEGY FLAVOUR
    kind: ClusterBuildStrategy
....
```

If the same `end-user` is willing to get more resources in order to even go beyond the previous build times, he/she could select a different flavour for buildpacks, see the following:

```yaml
---
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: buildpack-nodejs-build
  annotations:
    build.build.dev/build-run-deletion: "false"
spec:
  source:
    url: https://github.com/sclorg/nodejs-ex
  strategy:
    name: buildpacks-v3-large <---- PAY ATTENTION TO THIS LARGE STRATEGY FLAVOUR
    kind: ClusterBuildStrategy
....
```

With the flavours approach from above, `strategy admins` have full control on what they offer to their customers in terms of resources tunning for each strategy.

In order to provide more insights on how the resources definition will look inside a strategy, see the following example. Inside the `buildpacks-v3-default` strategy, we define different resource values throughout three different steps (_step-prepare,step-detect,step-build_):

```yaml
---
apiVersion: build.dev/v1alpha1
kind: ClusterBuildStrategy
metadata:
  name: buildpacks-v3-default
spec:
  buildSteps:
    - name: prepare
      image: docker.io/paketobuildpacks/builder:full
      securityContext:
        runAsUser: 0
        capabilities:
          add: ["CHOWN"]
      command:
        - /bin/bash
      args:
        - -c
        - >
          chown -R "1000:1000" "/workspace/source" &&
          chown -R "1000:1000" "/tekton/home"
      resources:
        limits:
          cpu: "10m"
          memory: "128Mi"
        request:
          cpu: "10m"
          memory: "128Mi"
    - name: detect
      image: docker.io/paketobuildpacks/builder:full
      securityContext:
        runAsUser: 1000
      command:
        - /cnb/lifecycle/detector
      args:
        - -app=/workspace/source/$(build.source.contextDir)
        - -group=/layers/group.toml
        - -plan=/layers/plan.toml
      volumeMounts:
        - name: layers-dir
          mountPath: /layers
      resources:
        limits:
          cpu: "250m"
          memory: "50Mi"
        request:
          cpu: "250m"
          memory: "50Mi"
    - name: restore
      image: docker.io/paketobuildpacks/builder:full
      securityContext:
        runAsUser: 1000
      command:
        - /cnb/lifecycle/restorer
      args:
        - -layers=/layers
        - -cache-dir=/cache
        - -group=/layers/group.toml
      volumeMounts:
        - name: cache-dir
          mountPath: /cache
        - name: layers-dir
          mountPath: /layers
    - name: build
      image: docker.io/paketobuildpacks/builder:full
      securityContext:
        runAsUser: 1000
      command:
        - /cnb/lifecycle/builder
      args:
        - -app=/workspace/source/$(build.source.contextDir)
        - -layers=/layers
        - -group=/layers/group.toml
        - -plan=/layers/plan.toml
      volumeMounts:
        - name: layers-dir
          mountPath: /layers
      resources:
        limits:
          cpu: "500m"
          memory: "1Gi"
        request:
          cpu: "500m"
          memory: "1Gi"
    - name: export
      image: docker.io/paketobuildpacks/builder:full
      securityContext:
        runAsUser: 1000
      command:
        - /cnb/lifecycle/exporter
      args:
        - -app=/workspace/source/$(build.source.contextDir)
        - -layers=/layers
        - -cache-dir=/cache
        - -group=/layers/group.toml
        - $(build.output.image)
      volumeMounts:
        - name: cache-dir
          mountPath: /cache
        - name: layers-dir
          mountPath: /layers
```

### Risks and Mitigations

Proper documentation needs to be made to communicate the existence of `flavours` for strategies of the same type to users. When it comes to UI, is the decision of the UI team to decide
how they will expose(_interface_) this flavours to their end-users. Strategy admins have full responsibility on the flavours they decide to define, but recommendations of flavours should be
made for this repository.

## Design Details

### Test Plan

- Unit-tests require an update. We need to validate that resources defined in the strategies are propagated to the TaskRun steps resources.
- CI would run the same `test-unit` make target, for validation.

### Graduation Criteria

**Note:** *Section not required until targeted at a release.*

#### Examples

##### Dev Preview -> Tech Preview

- N/A

##### Tech Preview -> GA

- N/A

**For non-optional features moving to GA, the graduation criteria must include end to end tests.**

##### Removing a deprecated feature

- N/A

### Upgrade / Downgrade Strategy

- N/A

### Version Skew Strategy

- N/A

## Implementation History

- N/A

## Drawbacks

None. Only that we will remove a feature that is currently not performing properly.

## Alternatives

Create a separate CRD for the container resources settings and allow the strategy to have an optional reference to it. This provides two potential benefits:

- Incremental progress toward moving away from directly embedding corev1.Container in our strategies.
- Allow strategies to select if customize resources are desired (_e.g. expensive resources_)

This alternative approach still requires to answer the following questions:

- How to reference this new CRD on the existing strategies?
- How to define different resources for multiple steps(_same strategy_) on an instance of the new CRD?. Similar to the issue we have now with the `spec.resources` in the Build, where we cannot define customize values per step.
- Impact on the user experience? in terms of another CRD.
- Where to define default values for strategy steps?

## Infrastructure Needed [optional]

N/A
