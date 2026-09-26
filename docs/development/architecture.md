<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

# Repository Architecture

Shipwright Build is a Kubernetes controller that turns Shipwright custom resources into Tekton build resources. The main execution path is:

1. `cmd/shipwright-build-controller` starts the controller process.
2. `pkg/controller` creates the controller-runtime manager, registers schemes and caches, and installs the reconcilers.
3. `pkg/reconciler` observes `Build`, `BuildRun`, `BuildStrategy`, and `ClusterBuildStrategy` resources, validates them, and creates or updates Kubernetes and Tekton resources.
4. Kubernetes schedules the resulting pods and Tekton executes the generated `TaskRun` or `PipelineRun` that performs the image build.
5. Status and results are propagated back to the Shipwright resources.

## Repository directories

| Directory | Purpose |
| --- | --- |
| [`cmd/`](../../cmd) | Entrypoints for the controller, conversion webhook, and helper command-line programs for Git, bundles, image processing, and local-source waiting. |
| [`pkg/apis/`](../../pkg/apis) | Shipwright API types and registration for `v1alpha1` and `v1beta1` resources, including generated deepcopy code. |
| [`pkg/client/`](../../pkg/client) | Generated Kubernetes clients, informers, listers, and related scheme code for Shipwright resources. Regenerate it with the repository generation targets. |
| [`pkg/reconciler/`](../../pkg/reconciler) | Reconcilers for Builds, BuildRuns, BuildStrategies, ClusterBuildStrategies, and cleanup behavior. BuildRun resource builders also assemble the Tekton resources used for execution. |
| [`pkg/validate/`](../../pkg/validate) | Reusable validation for resource names, sources, strategies, parameters, secrets, volumes, triggers, scheduling, and related fields. |
| [`pkg/controller/`](../../pkg/controller) | Controller-runtime manager setup, scheme registration, cache configuration, and reconciler registration. |
| [`pkg/config/`](../../pkg/config) | Controller configuration and environment-variable handling, including concurrency, Kubernetes client, helper-image, and metrics settings. |
| [`pkg/webhook/`](../../pkg/webhook) | Conversion-webhook interfaces and TLS configuration used by the webhook process. |
| [`pkg/git/`](../../pkg/git) | Git source retrieval support, including revisions, credentials, submodules, and Git LFS behavior. |
| [`pkg/image/`](../../pkg/image) | Container-image loading, mutation, pushing, registry credentials, and vulnerability-scan integration. |
| [`pkg/bundle/`](../../pkg/bundle) | OCI bundle creation and retrieval for packaged source content. |
| [`pkg/util/`](../../pkg/util) | Shared utilities such as file listing and network helpers. |
| [`pkg/env/`](../../pkg/env) | Helpers for reading and parsing environment-variable configuration. |
| [`pkg/volumes/`](../../pkg/volumes) | Volume-related helpers used when resolving build and strategy volume declarations. |
| [`pkg/metrics/`](../../pkg/metrics) | Prometheus metrics and optional profiling handlers exposed by the controller. |
| [`pkg/ctxlog/`](../../pkg/ctxlog) | Context-aware structured logging setup used by controller processes. |
| [`images/`](../../images) | Dockerfiles and image-build configuration for the helper containers. |
| [`deploy/`](../../deploy) | Kubernetes deployments, services, service accounts, RBAC, and generated CRD manifests. |
| [`samples/`](../../samples) | Example Shipwright `Build`, `BuildRun`, and strategy resources for both supported API versions. See the [sample catalog](../../samples/README.md). |
| [`test/`](../../test) | Integration and end-to-end tests, test data, sample validation, and shared test utilities. |
| [`hack/`](../../hack) | Scripts for installation, code generation, generated-code verification, certificates, local infrastructure, releases, and image checks. |
| [`docs/`](..) | User, operational, tutorial, and contributor documentation. |
| [`version/`](../../version) | Runtime version value and setter used by the controller's version reporting. |

## Generated code and manifests

API definitions in `pkg/apis/build` are used with generated clients and informers under `pkg/client`. Code-generation support is also present in `pkg/kubecodegen`, and the `make generate` target updates generated clients, fakes, RBAC, and CRDs. The generated CRD manifests are stored under [`deploy/crds/`](../../deploy/crds).

When changing API types or controller markers, follow the generation and verification instructions in [`docs/development/testing.md`](testing.md) and [`DEVELOPMENT.md`](../../DEVELOPMENT.md).
