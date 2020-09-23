<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

---
title: Introduce synthetic load generator tool<br/>
authors: @HeavyWombat<br/>
reviewers: TBD<br/>
approvers: TBD<br/>
creation-date: 2020-09-18<br/>
last-updated: 2020-09-18<br/>
status: provisional<br/>
see-also: _n/a_<br/>
replaces: _n/a_<br/>
superseded-by: _n/a_<br/>

---

# Neat Enhancement Idea

Introduce a tool to create synthetic load in a Kubernetes cluster using [shipwright-io/build] capabilities to help with performance tests or test plans.

## Release Signoff Checklist

- [X] Enhancement is `implementable`
- [ ] Design details are appropriately documented from clear requirements
- [ ] Test plan is defined
- [ ] Graduation criteria for dev preview, tech preview, GA
- [ ] User-facing documentation is created in [docs](/docs/)

## Open Questions [optional]

This is where to call out areas of the design that require closure before deciding to implement the
design. For instance:

> 1. Use dynamic client or `code-gen` created `clientset`?
> 1. What kind of default reports should be included, e.g. GraphJS, text, CSV?

## Summary

The performance of [shipwright-io/build] depends on the cluster capabilities and the actual application that is build into a container image. Using well-known synthetic applications for different cluster build strategies allows for reproducible execution times. In order to be able to detect performance regressions over time, it is useful to re-use the same test load all the time. Furthermore, the load generation should use the custom resrouce definition provided by `build` directly to avoid any additional overhead by a wrapper or convenienve layer.

## Motivation

We test our [shipwright-io/build] setup in different cluster setups using the same load to verify certain performance assumptions. This includes scenarions where a lot of concurrent builds are required. At this level, more simple load drivers such as Shell scripts become hard to maintain or impracticle.

### Goals

To have a simple command line tool to generate generic synthetic work load patterns in a Kubernetes cluster with [shipwright-io/build] CRDs.

### Non-Goals

The command line tool is written with [shipwright-io/build] and its CRDs in mind, it should not be a fully generic load driver tool.

## Proposal

### User Stories [optional]

#### Story 1

As a performance test, I want to quickly generate a lot of `build` load into a cluster to response how many builds went through and how fast they were.

#### Story 2

As a performance test automator, I want to re-run a well known test plan over and over again to produce a report. The test plan may contain entries like, run 100 Kaniko builds of this, and 200 BuildpacksV3 build of that and so on and so forth. The report would be a document (HTML, or PDF) to include all necessary details that can be compared with other versions of the report.

### Implementation Details/Notes/Constraints [optional]

The prototype in [homeport/build-load] uses a simple and direct approach. At the moment, it uses the dynamic client approach to apply Kubernetes resources like a `build`, or `buildrun`.

### Risks and Mitigations

No known risks, after all, it is a side project to test the main project.

## Design Details

### Test Plan

Section is not applicable at the moment.

### Graduation Criteria

Section is not applicable at the moment.

#### Examples

##### Dev Preview -> Tech Preview

Section is not applicable for this kind of tool.

##### Tech Preview -> GA

Section is not applicable for this kind of tool.

##### Removing a deprecated feature

Section is not applicable for this kind of tool.

### Upgrade / Downgrade Strategy

A straightforward release cycle using `semver` style version numbers would be best. Releases and the binaries could be stored in GitHub. Installation should be either a `brew` tap or convenience curl to shell installation script.

### Version Skew Strategy

The major version number should be aligned to the `build` API version and properly documented in the main README.

## Implementation History

No history.

## Drawbacks

It would be yet another piece of code that needs to be maintained and updated.

## Alternatives

One possible alternative is providing shell scripts to perform similar tasks.

## Infrastructure Needed [optional]

For the first version of the tool, only public infrastructure is required, i.e. GitHub, Travis CI, and Homebrew.

[shipwright-io/build]:https://github.com/shipwright-io/build
[homeport/build-load]:https://github.com/homeport/build-load
