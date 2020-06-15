---
title: Build Strategies steps resource limitations
authors:
  - "@qu1queee"
  - "@xiujuan95"
  - "@SaschaSchwarze0"
  - "@zhangtbj"
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

## Release Signoff Checklist

- [x] Enhancement is `implementable`
- [x] Design details are appropriately documented from clear requirements
- [x] Test plan is defined
- [ ] Graduation criteria for dev preview, tech preview, GA
- [x] User-facing documentation is created in [docs](/docs/)

## Open Questions [optional]

> 1. This locks users to define different resource constrains for multiple Build Strategy steps. Build/ClusterBuildStrategy admins can still do this. Do we want this?

## Summary

The current implementation of how resources(_e.g. CPU, Memory_) are applied to the Build Strategy steps have a flaw in
the implementation. The flaw is that the resources numbers, apply to all of the steps(_containers_) listed on the related
Build/ClusterBuild Strategy, hoarding resources. This ends up with a higher number for resources consumption than what the user initially defined in
the `Build` instance. For example, if users define `500m` as CPU request, but the strategy consists of five steps, then the used
resources will be `500m` multiplied by `5`. This of course, can have implications for billing.

## Motivation

For strategies with multiple steps like Buildpacks, not all the steps required the same amount of resources. For this multi-step strategies we
want to identify the **Step** that will consume the highest resources and allow users to be able to modify that step(_container_) resources through the `Build`. In other words, the more resources you assign to that step, the **faster** the build will be.

### Goals

- Keep the `Build` API untouched. Basically the `spec.resources` remains the same, but this configuration will only apply to one
particular step in the related Build/ClusterBuildStrategy strategies.

- Keep the user away from the strategies insight. User should only know that by adding more resources for X strategy, the build will be faster.

- Allow users to modify one single Build/ClusterBuildStrategy strategies step(_the one that consumes the more resources_), by using the `Build` `spec.resources`.

- Allow Build/ClusterBuildStrategy **admins** to define which step on the strategy will get the resources, by setting up a boolean flag. But also admins currently could
  define particular resources for each step.

### Non-Goals

- Defining the concrete numbers for the resources(_e.g. CPU/memory_) used per steps in the different Build/ClusterBuildStrategy strategies.

## Proposal

Currently we assign the numbers inside the Build `spec.resources` into all the `steps` for the related strategy. We want to modify this
by only applying those resources to a single step(_container_). The mechanism for doing this will consist of three parts:

1. We will introduce a new field under the `BuildStep` struct, see `buildstrategy_types.go`. This field will be of the type `boolean`. We propose the following:
   ```go
    // BuildStep defines a partial step that needs to run in container for
    // building the image.
    type BuildStep struct {
    	ResourceEnabled  bool `yaml:"resourceEnable,omitempty"`
    	corev1.Container `yaml:",inline"`
    }
   ```
2. The Build/ClusterBuildStrategy admin will be able to set the above field via a key on the step he/she considers is the more resource consumer.
   ```yaml
    - name: step-build
      image: $(build.builder.image)
      resourceEnable: true
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
   ```
3. We need to use the above flag in the logic, inside the `generate_taskrun_test.go`, during the loop into all of the strategy steps.

### Risks and Mitigations

For the end-user the only concern is that only one container resources could be modified per strategy. This should be the container in the strategy
consuming the highest number of resources. Build/ClusterBuildStrategy admins can always modify the steps manually, modifying any step with the resources they want.

## Design Details

### Test Plan

**Note:** *Section not required until targeted at a release.*

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

End-Users are limited to only be able to define a single set of values for resources in N strategy step.
If the end-user would required to set different values for different strategy steps, he/she will not be able to do this.
However this implies that the user have good knowledge on the used Build/ClusterBuildStrategy strategy, which is not
supposed to be the case.

## Alternatives

None at the moment.

## Infrastructure Needed [optional]

N/A