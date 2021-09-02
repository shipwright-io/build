<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

---
title: webhook-validation
authors:
  - "@ImJasonH"
reviewers:
  - "@gmontero"
  - "@zhangtbj"
approvers:
  - "@qu1queee"
  - "@adamkaplan"
creation-date: 2020-03-19
last-updated: 2020-03-19
status: provisional
---

# Webhook Validation

**Build Enhancement Proposals have been moved into the Shipwright [Community](https://github.com/shipwright-io/community) repository. This document holds an obsolete Enhancement Proposal, please refer to the up-to-date [SHIP](https://github.com/shipwright-io/community/blob/main/ships/0012-webhook-validation.md) for more information.**

## Release Signoff Checklist

- [ ] Enhancement is `implementable`
- [ ] Design details are appropriately documented from clear requirements
- [ ] Test plan is defined
- [ ] Graduation criteria for dev preview, tech preview, GA
- [ ] User-facing documentation is created in [docs](/docs/)

## Open Questions [optional]

Do we want to require that operators run another job to synchronously validate client requests?

Do we want to rely on Knative's webhook packages, or write our own code to manage certs ourselves?
Is there anything already out there in controller-gen we should be using?

## Summary

Kubernetes provides facilities to inject synchronous validation of CRUD operations from clients for specified types, such as Shipwright's CRD types.

Webhooks can also mutate incoming requests, including setting default values and converting between versions of CRDs. These features should be considered later, and are not described in this proposal.

We should take advantage of these features and include a webhook validation deployment in the Shipwright installation to provide this functionality.

## Motivation

Today, when a user creates a resource (e.g., a Build, BuildRun, etc.), they are immediately stored in the cluster's etcd storage, and the Shipwright controller begins reconciling the object.

If the controller finds that the resource is invalid, it updates the resource's `.status` to indicate this state and stops reconciling it.

Examples of invalid resources include:

- a BuildRun referencing a Build that doesn't exist
- a BuildRun referencing a ServiceAccount that doesn't exist
- a Build referencing a (Cluster)BuildStrategy that doesn't exist
- a BuildStrategy containing zero steps

(This is a partial list, and new validation can be assumed to be added over time as the Shipwright API evolves.)

By allowing invalid resources to be accepted and stored, we require clients to watch or poll all resources to ensure their request was valid, and we add to the API surface of resources that otherwise would not need to report a status, like a Build.

Always accepting and storing resources, even invalid ones, can also lead to additional storage of known-invalid objects, which clutter the user experience and can lead to confusion, and in extreme cases, cluster resource exhaustion.

Mixing reconciliation and validation code in the controller also complicates our codebase, leading to PRs like https://github.com/shipwright-io/build/pull/641 which add new validation-centric statuses that the user must watch for.
As we add new validation scenarios, we currently add new `Reason` strings that clients must be aware of, which is cumbersome and won't scale indefinitely.

### Goals

Enforce synchronous validation via a new Deployment and a [ValidatingAdmissionWebhook](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#validatingadmissionwebhook), and remove validation-centric logic from the reconciler.

The reconciler should be able to assume in many cases that the resource it is reconciling has already been validated.

### Non-Goals

- Include a mutating or defaulting or conversion webhook.
  If we decide these are useful, we can add them in future proposals.

## Proposal

Rely on [kubebuilder's webhook support](https://book.kubebuilder.io/cronjob-tutorial/webhook-implementation.html) to add a new webhook validation controller, and include it in Shipwright's default installation as a K8s Deployment.


### Implementation Details/Notes/Constraints [optional]

#### Certificates 

The webhook deployment will also need to manage certificates since K8s webhook requests are sent over HTTPS.
Setting up these certificates securely can be cumbersome.

The [Knative](https://knative.dev) project has [some libraries to make this management easier](https://github.com/knative/pkg/tree/main/webhook) by taking care of cert generation, registration and refreshing, but mixing Knative-style components and existing `controller-gen`/KubeBuilder-style components might make complicate development.

#### Webhook Availability

Once registered, if the K8s API server can't reach the defined webhook service, _all_ requests will fail for registered CRD types.
The webhook deployment is stateless, so it can be easily replicated and scheduled across multiple nodes to ensure availability in the case of node failure.

### Risks and Mitigations

Clients that expect to have requests accepted and asynchronously validated might be surprised to find the API synchronously rejecting their invalid requests instead.
We should communicate this change to users and operators in release notes and in mailing list announcements, and be available to answer any questions.

## Design Details

### Test Plan

Our current tests to validate resources should be mostly reusable; validation logic itself isn't going anywhere, just moving.

Any tests that we have that assume an invalid resource will be accepted and eventually reconciled should be updated to expect immediate failure.

### Version Skew Strategy

A user who has previously installed a version of Shipwright that did not synchronously validate might have invalid resources in their etcd storage, and our controller may be asked to reconcile them.
Basic validation logic should remain in the reconciler, but either update a status to indicate invalidity, or merely log and ignore invalid resources.

During installation or upgrade of Shipwright components, the webhook job and the reconciling controller job might briefly run two versions of validation code, so we should be careful to consider this as we add (or remove, or refactor) validation support over time.
In practice, Tekton hasn't experienced significant issues with this, but I wanted to call it out because it's possible.

## Implementation History

Major milestones in the life cycle of a proposal should be tracked in `Implementation History`.

## Drawbacks

This requires us to build a new component in the Shipwright installation, and for the operator component to operate it.

## Alternatives

Continue to validate asynchronously, which probably requires investing more in scaling the addition and documentation of validation failures.
