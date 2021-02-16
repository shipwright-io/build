<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

---
title: propagating-annotations-from-the-build-strategy-to-the-pod

authors:

- "@SaschaSchwarze0"

reviewers:

- "@zhangtbj"

approvers:

- "@adambkaplan"
- "@qu1queee"

creation-date: 2020-12-16

last-updated: 2020-01-20

status: implemented

see-also:

- "/docs/proposals/strategy.md"

---

# Propagating annotations from the build strategy to the pod

## Release Signoff Checklist

- [X] Enhancement is `implementable`
- [ ] Design details are appropriately documented from clear requirements
- [X] Test plan is defined
- [ ] Graduation criteria for dev preview, tech preview, GA
- [ ] User-facing documentation is created in [docs](/docs/)

## Open Questions [optional]

None.

## Summary

A build strategy administrator MUST only define very few details of how the `BuildRun`'s `Pod` looks at the end. This is intended and follows the principle from [The BuildStrategy API](buildstrategy.md):

> A slim BuildStrategy is one where the BuildStrategy author gets to accomplish more by specifying less.

But, beside the steps that are to be performed, the build strategy administrator already CAN define certain runtime behavior aspects like container resources and security context.

By enabling the administrator to also control the annotations of the `BuildRun` `Pod` IF necessary, more scenarios CAN be supported.

## Motivation

The runtime behavior of a `Pod` is not only specified in the containers. There are use cases where behavior of `Pod`s is described through annotations. This is especially the case for alpha and beta features of Kubernetes. Examples:

- The Kubernetes [Network Traffic Shaping](https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/network-plugins/#support-traffic-shaping) feature looks for the `kubernetes.io/ingress-bandwidth` and `kubernetes.io/egress-bandwidth` annotations to limit the network bandwidth the `Pod` is allowed to use.
- The [SecComp profile selection](https://kubernetes.io/docs/tutorials/clusters/seccomp/#create-a-pod-with-a-seccomp-profile-for-syscall-auditing) used to be done through the `seccomp.security.alpha.kubernetes.io/pod` annotation until Kubernetes 1.18 - 1.19 makes it a first-class property in the `Pod`'s security context.
- The [AppArmor profile of a container](https://kubernetes.io/docs/tutorials/clusters/apparmor/) is defined using the `container.apparmor.security.beta.kubernetes.io/<container_name>` annotation.

To use those features for `BuildRun` `Pod`s, control over the `Pod`'s annotations is necessary. As those features are clearly something that administrators should define rather then the users that define `Build`s and `BuildRun`s, it makes sense to restrict this feature to the `BuildStrategy` and `ClusterBuildStrategy` only.

### Goals

- Enable the build strategy administrator to define annotations that are copied to the `BuildRun`'s `Pod`.

### Non-Goals

- Enable users to define annotations on `Build` and `BuildRun` that are copied to the `BuildRun`'s `Pod`.
- Enable anybody to define labels on one of our custom resources and make them appear on the `BuildRun`'s `Pod`. This should be covered separately.

## Proposal

In the `BuildStrategy` and `ClusterBuildStrategy`, the build strategy administrator can define annotations in the metadata. This is possible in the same way as for all Kubernetes objects, see the [Annotations](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) topic in the Kubernetes documentation. Annotations are key/value pairs. The key consists of up to two parts: an optional prefix and a name. If both are defined, the `/` is used as separators. The prefix must be a DNS subdomain. Kubernetes reserves the `kubernetes.io` and `k8s.io` prefixes for its own core components. In build we already use annotations in the build controller using the `build.build.dev` prefix for two use cases:

- The `build.build.dev/build-run-deletion` annotation on builds controls whether its `BuildRun`s are deleted when the build is deleted.
- The `build.build.dev/referenced.secret` annotation tells the build controller that a secret is related to builds.

Based on that naming, the other three prefixes reserved by our controllers are: `buildstrategy.build.dev`, `clusterbuildstrategy.build.dev` and `buildrun.build.dev`.

When generating a Tekton `TaskRun`, the idea is to look at the annotations of the `BuildStrategy` or `ClusterBuildStrategy` and copy all annotations over to the `TaskRun`, except those that use one of our own prefixes because we today have no feature anyway where we look for one of our annotations on a `TaskRun` or `Pod`, and except the `kubectl.kubernetes.io/last-applied-configuration` annotation.

Tekton automatically copies all `TaskRun` annotations to the `Pod`, see [pod.go](https://github.com/tektoncd/pipeline/blob/v0.20.1/pkg/pod/pod.go#L257).

For example, this metadata of a cluster build strategy:

```yaml
apiVersion: build.dev/v1alpha1
kind: ClusterBuildStrategy
metadata:
  annotations:
    kubernetes.io/egress-bandwidth: 100M
    clusterbuildstrategy.build.dev/dummy: aValue
```

will lead to the following metadata on the `TaskRun` (and `Pod`):

```yaml
apiVersion: tekton.dev/v1beta1
kind: TaskRun
metadata:
  annotations:
    kubernetes.io/egress-bandwidth: 100M
```

### Implementation Details/Notes/Constraints [optional]

The implementation requires the [BuilderStrategy interface](../../pkg/apis/build/v1alpha1/buildstrategy.go) to be extended with a `GetAnnotations` functions that is implemented in the [BuildStrategy](../../pkg/apis/build/v1alpha1/buildstrategy_types.go) and [ClusterBuildStrategy](../../pkg/apis/build/v1alpha1/clusterbuildstrategy_types.go) types by returning the object's annotations.

The assignment of the `TaskRun` annotations needs to be done in the [generate_taskrun.go](../../pkg/reconciler/buildrun/resources/taskrun.go) file in the `GenerateTaskRun` function. The annotations from the build strategy need to be copied to the `TaskRun` except those using one of the four Shipwright Build owned prefixes mentioned under [Proposal](#proposal), and except the `kubectl.kubernetes.io/last-applied-configuration` annotation.

### Risks and Mitigations

A risk is that the build strategy administrator starts to use an annotation-controlled feature that the Kubernetes administrator does not want to be used. Third-party policy engines like [Open Policy Agent](https://www.openpolicyagent.org/) can be used by the Kubernetes administrator to prevent this without requiring anything from our operator - an [EP in Tekton](https://github.com/tektoncd/community/blob/master/teps/0035-document-tekton-position-around-policy-authentication-authorization.md#proposal) is suggesting the same. If this is considered not enough, then option (2) from the [alternatives](#alternatives) might be required.

## Design Details

### Test Plan

- The unit testing for the `TaskRun` generation must be extended.
- An integration test must be added to verify that annotations are copied over selectively from the `BuildStrategy` and `ClusterBuildStrategy` to the `TaskRun`.

### Upgrade / Downgrade Strategy

There is a behavior change that annotations on the `BuildStrategy` or `ClusterBuildStrategy` that a build strategy administrator has defined for whatever reason are now copied over to the `TaskRun` and `Pod`. These are either annotations without a behavioral change to the `Pod`, or annotations that the user already expected to be copied over which makes this proposal a fix for his scenario.

### Version Skew Strategy

N/A

## Implementation History

N/A

## Drawbacks

None

## Alternatives

(1) Instead of copying over the annotations from the `BuildStrategy` or `ClusterBuildStrategy` metadata, one could follow a similar pattern as the Kubernetes deployment with its PodTemplate where the annotations of the deployment are separated from the designated annotations for the `Pod`s created through the deployment (example: [here](https://github.com/kubernetes/kubernetes/issues/37666#issuecomment-283109237)). Translated into our use case, this would mean that the annotations for the `TaskRun` and `Pod` are then explicitly listed in the spec of the `BuildStrategy` or `ClusterBuildStrategy` rather than in the metadata:

```yaml
apiVersion: build.dev/v1alpha1
kind: ClusterBuildStrategy
metadata:
  name: a-cbs
  annotations:
    clusterbuildstrategy.build.dev/dummy: aValue
spec:
  podAnnotations:
    kubernetes.io/egress-bandwidth: 100M
```

This idea was not considered because it is unnecessary. The described filtering already eliminates annotations that are not of interest for the `Pod`.

(2) Instead of filtering out a hard-coded list of annotations by prefix (our Shipwright prefixes) or by full key (`kubectl.kubernetes.io/last-applied-configuration`), one could have introduced an extension to our [configuration](../../pkg/config/config.go) to allow the administrator of the build operator to configure which annotations are copied from the `BuildStrategy` and `ClusterBuildStrategy` to the `TaskRun`, using white- or black-listing.

This idea was not considered because we were not seeing a relevant use case for it.
