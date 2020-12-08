<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

---
title: package-olm-bundle-image
authors:
  - @adambkaplan
reviewers:
  - @SaschaSchwarze0
  - @otaviof
  - @gabemontero
approvers:
  - @qu1queee
  - @sbose78
creation-date: 2020-12-08
last-updated: 2021-01-22
status: implementable
see-also: []
replaces: []
superseded-by: []
---

# Package as an OLM Bundle Image

## Release Signoff Checklist

- [x] Enhancement is `implementable`
- [x] Design details are appropriately documented from clear requirements
- [x] Test plan is defined
- [ ] Graduation criteria for dev preview, tech preview, GA
- [ ] User-facing documentation is created in [docs](/docs/)

## Open Questions

None.

## Summary

This proposal will facilitate the release of Shipwright Builds as an operator on [Operator Hub](https://operatorhub.io) by packaging Shipwright's deployment in an OLM bundle image.
This will be accomplished by migrating `operator-sdk` to v1.3 or higher, which contains utitlites to create OLM bundle images and package manifests.
This proposal details a the requirements needed to migrate to Operator SDK v1, and how the bundle image will be built, tested, and deployed.

## Motivation

Operator SDK v1.0 and higher facilitates developemnt of operators that can be installed via the Operator Lifecycle Manager (OLM).
Version 1.x includes commands that help developers create and verify OLM bundle images [1], which is the preferred means that operators are installed via OLM.

Operator SDK v1.0 introduced substantial changes to the way sdk-enabled repositories are structured.
The changes are so drastic for legacy projects that the Operator SDK maintainers recommend a "lift and shift" approach to migration, versus an in-place update [2].
The scope and reach of these changes are significant, and it is important to identify key decisions that may cause breaking changes.

[1] https://github.com/operator-framework/operator-registry/blob/v1.12.6/docs/design/operator-bundle.md
[2] https://sdk.operatorframework.io/docs/building-operators/golang/migration/#migration-steps

### Goals

* Package Shipwright Builds as an OLM bundle image by using Operator SDK v1.
* Identify key decisions that need to be made in the Operator SDK migration
* Plan the implementation of the Operator SDK migration

### Non-Goals

* Define requirements for a migration tool if we introduce breaking API changes
* Define a strategy for presenting install/upgrades via OperatorHub
* Move the Build APIs to the `shipwright.io` domain.

## Proposal

### Project initialization

Operator SDK initializes CRDs with two key flags - `--group` and `--domain`.
`domain` refers to the DNS segment that all APIs for the operator share, and Operator SDK uses this to boostrap the entire project [3].
We will keep the current domain of `build.dev`.

The project does not require multi-group APIs at present, and moving forward this will likely remain the case [4].

[3] https://sdk.operatorframework.io/docs/building-operators/golang/migration/#create-a-new-project
[4] https://sdk.operatorframework.io/docs/building-operators/golang/migration/#check-if-your-project-is-multi-group


### API Migration

Operator SDK v1.x changed the package layout of APIs from `pkg/apis/<group>/<version>/<kind>_types.go` to `/api/version/<kind>_types.go`.
Other notable changes to API generation include [5]:

1. Replacing `+k8s:deepcopy-gen:interfaces=...` markers with `+kubebuilder:object:root=true`.
2. Removing `// +k8s:openapi-gen=true` and other related openapi markers. We do not use this at
   present, and therefore do not need to restore these markers.

We currently use `deepcopy-gen` markers to generate our deep copy code, and use `k8s.io/code-generator` to create our OpenAPI generated code under the covers.
Presumably using the `kubebuilder` marker will let us retain our generated deep copies.

[5] https://sdk.operatorframework.io/docs/building-operators/golang/migration/#apis

### Controllers

Controller code can more or less be migrated as is from `pkg/controller/<kind>/<kind>_controller.go` to `/controllers/<kind>_controller.go` [5].
There are a few subtleties with generated types and a change to the manager interface, but in general the shift is straightforward.

[6] https://sdk.operatorframework.io/docs/building-operators/golang/migration/#controllers

### RBAC

RBAC is maintained via kubebuilder RBAC Markers, rather than separate YAML files.
The RBAC markers are placed on the `Reconcile` method of each controller [7], alongside implementing code.
SDK projects can then generate RBAC for the controllers via a make target (`make manifests`).
RBAC YAML has been moved to `config/rbac/` directory, rather than living alongside deployment manifests in `deploy`.

New controllers are cluster-scoped by default [8].
We believe this is the right scoping for the build controllers.

[7] https://sdk.operatorframework.io/docs/building-operators/golang/migration/#set-the-rbac-permissions
[8] https://sdk.operatorframework.io/docs/building-operators/golang/operator-scope/

### Migrate main.go

The main invocation has changed its default generated code with respect to leader election.
Operator SDK has not deprecated `leader.Become` (yet), but it has moved new code to use the leader election libraries from controller-runtime [8].
We have already migrated to controller-runtime to handle leader election and controller initialization - therefore this is not a concern.

Our operator allows the leader election namespace to be customized via the `BUILD_OPERATOR_LEADER_ELECTION_NAMESPACE` environment variable.
This is ultimately wired through to controller-runtime's leader election logic - this should remain stable post migration.

[8] https://sdk.operatorframework.io/docs/building-operators/golang/migration/#migrate-maingo

### Migrate tests

`operator-sdk test` and its related packages are removed in v1, instead opting for the envtest library alongside ginkgo and gomega [9].
The test framework aligns with the upstream kubebuilder project, which now allows integration tests to run without an actual cluster [10].
Tests are run via `make` targets, making it easy to modify at a later date.
They use envtest by default to allow integration tests to run without a cluster [11].

We do not need to migrate our existing integration test suite, since it does not rely on `operator-sdk test` or `operator-sdk/pkg/test`.
Moving to envtest may be beneficial long-term, but is not necessary.

Our end to end tests will need setup changes.
The current e2e suite runs the controller manager outside of the cluster via `operator-sdk run local`.
We can improve this by deploying the controller on the cluster, either via the generated `make deploy` target or via OLM installation.

[9] https://sdk.operatorframework.io/docs/building-operators/golang/migration/#migrate-your-tests
[10] https://book.kubebuilder.io/cronjob-tutorial/writing-tests.html
[11] https://book.kubebuilder.io/reference/envtest.html#configuring-envtest-for-integration-tests

### Operator Customizations

`BUILD_OPERATOR_LEADER_ELECTION_NAMESPACE` is the only customization that is readily declared as an environment variable.
The migration may provide an opportunity to include other customizations that can be declared.

### Migrate metrics

The ServiceMonitor for exporting metrics is via kustomize, rather than in code [12].
This eliminates the boilerplate generated from the `addMetrics` function in `main.go`.

We should still be able to export the default metrics for our custom resources, as well as the detailed metrics that capture build performance.

Note too that with operator-sdk v1 (at least v1.2) the way metrics are exported has changed significantly.
In v0.18 and lower, the controller manager binary created the metrics service and service monitors on startup.
Two separate ports were opened to serve metrics - one for the baked in CustomResource metrics, and another to serve our application-specific metrics.

In v1 there is only one metrics port.
The application exposes metrics on port 8080 by default.
The generator then adds a kube-rbac-proxy sidecar to the controller manager, which acts as a reverse HTTP proxy for the metrics.
With the rbac proxy, metrics are served over TLS and authorized via subject access reviews [13].
Baked in CustomResource metrics can be turned on via code.

The generated services can all be tuned and adjusted via generated YAML manifests.
The manifests are managed via kustomize to render the final deployments.
Items like the rbac-proxy can be removed by adjusting the kustomize settings.

It is unclear if OLM would tear down the old metrics service on our behalf - the assumption is yes.
The new service montior should be capable of connecting to new metrics service.

[12] https://sdk.operatorframework.io/docs/building-operators/golang/migration/#export-metrics
[13] https://sdk.operatorframework.io/docs/building-operators/golang/migration/#use-handler-from-operator-lib

### Process migrations

A few of the common development commands have been moved out of operator-sdk:

1. `operator-sdk generate crds` has been replaced with the generated `make manifests` target.
2. `operator-sdk build` has likewise been replaced with a make target - `make docker-build` [13].
3. Deploying is likewise handled by a make targets - `make install` and `make deploy` [14].

The generated Makefile targets can be used by contributors to test Shipwright Builds.
We are free to add our own targets to generate an "all-in-one" YAML manifest.

[13] https://sdk.operatorframework.io/docs/building-operators/golang/migration/#generate-manifests-and-build-the-operator
[14] https://sdk.operatorframework.io/docs/building-operators/golang/quickstart/

### Build the Bundle Image

After migration, `operator-sdk` will boostrap a `make bundle` target that creates and validates the OLM bundle image for the operator.
When code merges to the main/master branch, or a new release tag is created, the bundle image needs to be pushed to quay.io.
Optionally, we should run the e2e suite against a cluster that installs the operator via OLM.
This can be taken on as a follow-up action, since operator-sdk does not currently have a simple means to test bundle images that are installed by OLM.

### Risks and Mitigations

**Risk**: Bug fixes and features are wiped out as we migrate to operator-sdk v1.

*Mitigation*: As the rebase PR nears completion, we hold outstanding pull requests until it merges.
Our CI needs to ensure that the rebase can merge without conflicts.

## Design Details

### Test Plan

After migration, our existing test suites should be restored via their respective Makefile targets.
The e2e suite will require configuration/setup changes such that the controllers run within a cluster.

Integration and e2e tests have been separated and run via separate CI jobs.

### Graduation Criteria

Not applicable

### Upgrade / Downgrade Strategy

Since this project is in the alpha state, support for in-place upgrade is not a hard requirement.
This proposal outlines specific areas where there is potential to introduce breaking changes that downstream distributions may need to consider.

### Version Skew Strategy

If the leader election ConfigMap is left unchanged, we can support operator skew.

## Implementation History

2020-12-08: Initial proposal
2020-12-09: Alternative using knative/pkg
2020-12-16: Removed integration test migration and API domain rename
2021-01-22: Update to emphasize creating bundle images

## Drawbacks

Migrating to operator-sdk v1 requires significant project restructuring and changes to how we set up the e2e test suite.
Bug fixes and other enhancements could easily get lost in the shuffle.

The primary benefit of operator-sdk is to facilitate the deployment of Shipwright Build as an operator managed by the Operator Lifecycle Manager.
`operator-sdk` has tooling to help create the necessary artifacts for OLM, but one does not need operator-sdk to create the components needed to deploy an operator via OLM.

## Alternatives

### Use Upstream Components

We can use the sub-components that operator-sdk packages together, without using operator-sdk directly:

1. Controller-runtime can be used to write the controllers and controller manager [15].
2. KubeBuilder can be used to create the custom resources [16].
3. Kustomize can be used to manage our YAML manifests [17].

[15] https://github.com/kubernetes-sigs/controller-runtime
[16] https://github.com/kubernetes-sigs/kubebuilder
[17] https://kustomize.io/

### Clean Break Migration

In this scenario, we declare the version running operator-sdk v1 a "fully breaking" update, and introduce changes that are not backwards compatible.
Such actions would include altering the leader lock `ConfigMap` and changing the domain for the Shipwright Build APIs.
Other substantial changes, such as updating the integration test suite to run via envtest, could also be considered.

This would explode the scope of the migration, risking our ability deliver other bug fixes and features.

### Switch to knative.dev/pkg

Tekton uses knative packages for their controller code and operator.
This is largely due to their community's experience and expertise, as Tekton was born out of the knative/build project.
They are currently shipping Tekton as an operator in OperatorHub.

The Shipwright community, on the other hand, does not have as much experience with knative libraries.
More significant refactorings will be required if we abandon operator-sdk and its dependent libraries (kubebuilder and controller-runtime).

## Infrastructure Needed [optional]

- New repository on quay.io to host the OLM bundle image
