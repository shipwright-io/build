<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

---
title: local-registries
authors:
  - "@adambkaplan"
reviewers:
  - "@coreydaley"
  - "@otaviof"
approvers:
  - "@qu1queee"
  - "@sbose78"
creation-date: 2020-07-24
last-updated: 2020-08-10
status: provisional
see-also: []  
replaces: []
superseded-by: []
---

# Local Registry Image Specs

## Release Signoff Checklist

- [ ] Enhancement is `implementable`
- [ ] Design details are appropriately documented from clear requirements
- [ ] Test plan is defined
- [ ] Graduation criteria for dev preview, tech preview, GA
- [ ] User-facing documentation is created in [docs](/docs/)

## Summary

[KEP 1755](https://github.com/kubernetes/enhancements/tree/master/keps/sig-cluster-lifecycle/generic/1755-communicating-a-local-registry)
provides an official convention for clusters to declare the presence of an internal image registry.
This proposal aims to take advantage of declared local image registries, and let application
developers simplify how they declare image pull and push specs. This may also help `Build` objects
be portable across clusters.

## Motivation

The `Build` API in its present form requires that users provide a full image pull spec for image
references. This can be cumbersome when referencing an image on an internal registry, such as those
present in [KIND](https://kind.sigs.k8s.io/docs/user/local-registry/),
[Rancher](https://github.com/rancher/k3d/blob/master/docs/registries.md#using-a-local-registry),
[microk8s](https://microk8s.io/docs/registry-built-in), and
[OpenShift](https://docs.openshift.com/container-platform/4.5/registry/architecture-component-imageregistry.html).
This proposal will provide a means of declaring that an image spec references a local registry,
thereby allowing host information to be removed from the image reference.

### Goals

- Simplify how images on local image registries can be referenced in `Build` objects.
- Provide a means of declaring that an image references a cluster-local image registry.

### Non-Goals

- Automate how pull/push secrets can be added to a `Build` object.
- Support secure image pull/push if a local registry uses self-signed certificates.
- Configure image registry mirrors within builds.
- Alter the default image registry for image pull/push if no hostname is specified.

## Proposal

Application developers will be able to specify `local: true|false` if an image reference points
to a cluster-local image registry.

Per [KEP 1755](https://github.com/kubernetes/enhancements/tree/master/keps/sig-cluster-lifecycle/generic/1755-communicating-a-local-registry),
a cluster has a local registry if a `ConfigMap` named `local-registry-hosting`  is present in the
`kube-public` namespace. The data in this `ConfigMap` contains hosname information for the image
registry installed on the cluster. The build controller/operator will read this `ConfigMap` to check if
a local image registry is declared. If declared, it will infer the local registry host information
from what is provided in the `ConfigMap`.

When a `BuildRun` is instantiated, the image specs that declare `local: true` are mutated to
prepend the cluster's local image registry hostname.

### User Stories [optional]

As an application developer
I would like to declare that images in builds reference a local image registry
So that I do not have to provide host information in the pull spec

### Implementation Details/Notes/Constraints [optional]

#### Obtaining local registry information

Clusters which deploy a local image registry must provide the releavant host info by
publishing it in the `local-registry-hosting`
[ConfigMap](https://github.com/kubernetes/enhancements/tree/master/keps/sig-cluster-lifecycle/generic/1755-communicating-a-local-registry#the-local-registry-hosting-configmap).
The build operator/controller must have RBAC which allows it to read the `ConfigMap` and obtain the
registry host information. For `Build` and `BuildRun` objects, the `HostFromClusterNetwork` value
should be used as the hostname.

If the `local-registry-hosting` ConfigMap is created, updated, or deleted, the build controller
and/or operator need to react accordingly.

If the `local-registry-hosting` ConfigMap is not present, the build operator/controller must
be able to function normally. Absence of this information is not considered an error.

#### Declaring an image as local

`Image` references in the API will be updated to have the `local` field. An example using the 
`output` spec:

```yaml
spec:
  output:
    image: mynamespace/myapp:latest
    local: true
```

When an image is declared as `local`, the build controller prepends the local image registry
hostname to the provided image reference.

If the `local-registry-hosting` ConfigMap specified in KEP-1755 is not present, or does not contain
sufficient information to inject the registry hostname, `Build` and `BuildRun` objects should
present appropriate error messages in there statuses:

- `Build` objects will set the `Registered` status to `false`.
- `BuildRun` objects will present an error message in their status conditions. Note that this
  standard status conditions are not present in the `BuildRun` API at present. 

### Risks and Mitigations

**Risk**: App developers use `local: true` on clusters that don't have a local registry defined,
In this event build could inadvertently pull or push images to `docker.io` or an image registry in
the container runtime's unqualified search path.

*Mitigation*: if a `Build` references a local registry with no registry defined, it's `Registered`
status should be set to `false`. For `BuildRun`, the run should fail with a clear error message in
its status conditions.

**Risk**: Local image registry configurations can be altered while a build is running.

*Mitigation*: Image spec resolution should only happen when the `BuildRun` is transformed into an
appropriate Tekton `TaskRun`. The `BuildRun` will likely fail, but that can be addressed by re-
running the same build via a new `BuildRun` object.

The build controller itself will need to be restarted by a separate operator and reload new
configuration via a deployment rollout. This is a separate matter that is identified in
[#310](https://github.com/redhat-developer/build/issues/310).

## Design Details

### Test Plan

1. Deploy the build operator/controller on a cluster that populates the `local-registry-hosting`
   ConfigMap OR as cluster-admin, manually populate the `local-registry-hosting` ConfigMap.
2. Create a `Build` and/or `BuildRun` which references a local image in the following mannter:
   1. As output for a build.
   2. As the `builder` image - ex. for use in the source-to-image `BuildStrategy`.
   3. As the `runtime` base image for builds that create lean runtime images.
3. Ensure that a build can push an image referencing the local image spec.

### Graduation Criteria

#### Dev Preview

As an initial implementation, we can define an environment variable to set the local registry
hostname for the build operator. This can be changed by alterting the deployment for the build
operator, which would force a rollout with the new value.

#### Tech Preview and GA

The operator must fully read the `local-registry-hosting` ConfigMap, with the ability to update
the build controller if this ConfigMap changes.

#### Examples

### Upgrade / Downgrade Strategy

Upgrades add a new field to the API. On downgrade the `local` attribute should be ignored.

### Version Skew Strategy

N/A

## Implementation History

2020-07-24: Initial proposal

## Drawbacks

1. The KEP is relatively new, and not many *KS providers will have a need to implement it.
2. In production environments, may users push images to registries that reside outside of the
   cluster. Therefore this feature may not prove valuable.
3. `Build` and `BuildRun` definitions will not be portable across namespaces with this
   implementation - they will only be portable across clusters.

## Alternatives

### okd ImageStreams

OpenShift/okd [Imagestreams](https://docs.okd.io/latest/openshift_images/images-understand.html#images-imagestream-use_images-understand)
are a primary motivation for this enhancement proposal. Shortened image pull specs which resolve to
a local image registry is one of many features provided by this API.

Upstreaming Imagestreams would require significant effort and may be beyond the scope of this
project.

### APIs for Default Search Paths

Most container image build tools inject `docker.io` as the host if an image spec does not have a
domain. This can often be overrode to reference another "default" registry:

- Kaniko - the `--registry-mirror` option lets you override `docker.io` as the default. [1]
- Buildah - uses the file `/etc/containers/registries.conf` to configure unqualified search
  paths. [2]
- Buildpacks - no known means of changing `docker.io` as the default path for pulling images.

To reference a local image registry, the following would be needed:

1. An API needs to be exposed so that the default image registry can be changed to a local registry
   on the cluster. This would not need to be published via the mechanisms in KEP-1755.
2. Builds would need to expose this configuration setting to every build (ex: a `ConfigMap` volume
   mount).
3. Build strategies would need to provide the appropriate configuration option to the commands that
   execute the build.

The main downside of this approach is that every build strategy would need to opt into this
feature. Using `docker.io` as the default image registry is also assumed for the general k8s
ecosystem - alterting this has proved to be a source of bugs, unexpected behavior, and difficult
debug situations.

Changing the default image registry is also a blunt configuration - it is applied to all image pull
and push actions for the duration of the build. You cannot use the local registry for some image
references, and `docker.io` for others.

[1] https://github.com/GoogleContainerTools/kaniko#--registry-mirror
[2] https://www.mankier.com/5/containers-registries.conf#Description-Global_Settings


## Infrastructure Needed [optional]

- A cluster which populates the `local-registry-hosting` ConfigMap

## Open Questions [optional]

1. How can we make `Build` objects with local image pull specs portable across namespaces?
