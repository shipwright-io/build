<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

# Major Dependencies

The dependencies below are direct architectural dependencies in `go.mod`. Indirect modules are omitted; they are transitive implementation details of these libraries or of the Kubernetes and Tekton stacks.

| Dependency | What it provides | Why Shipwright uses it | Primary usage |
| --- | --- | --- | --- |
| Kubernetes API machinery and `client-go` | API types, runtime schemes, REST clients, discovery, watches, and generated client support. | Shipwright extends Kubernetes with custom resources and must read and write them through the Kubernetes API. | [`pkg/apis`](../../pkg/apis), [`pkg/client`](../../pkg/client), [`pkg/controller`](../../pkg/controller), and reconcilers. |
| `sigs.k8s.io/controller-runtime` | Managers, caches, clients, controller loops, reconciliation, webhooks, logging, and metrics integration. | It supplies the controller framework used to watch resources and coordinate the Shipwright controllers. | [`pkg/controller`](../../pkg/controller), [`pkg/reconciler`](../../pkg/reconciler), and [`pkg/webhook`](../../pkg/webhook). |
| Tekton Pipelines | `TaskRun` and `PipelineRun` APIs, results, conditions, and execution behavior. | Shipwright delegates the actual build steps to Tekton resources generated from Shipwright strategies. | [`pkg/reconciler/buildrun`](../../pkg/reconciler/buildrun), [`pkg/controller`](../../pkg/controller), and the controller command. |
| Knative packages | Kubernetes-style conditions and signals/lifecycle helpers. | Shipwright uses Knative conventions for resource status and process lifecycle behavior. | [`pkg/reconciler/buildrun`](../../pkg/reconciler/buildrun) and the webhook command. |
| `go-git` | Git repository access, revisions, transport authentication, ignore rules, and in-memory storage. | The Git helper must retrieve build sources with the supported authentication and revision features. | [`pkg/git`](../../pkg/git) and bundle source handling. |
| `go-containerregistry` | OCI image references, registry access, authentication, image mutation, layouts, tarballs, and remote images. | Shipwright packages source bundles and processes, annotates, and pushes build output images. | [`pkg/bundle`](../../pkg/bundle) and [`pkg/image`](../../pkg/image). |
| Docker CLI credential/config packages | Docker registry configuration and credential-file parsing. | Helper code and tests need to use standard Docker registry credentials. | [`pkg/image`](../../pkg/image) and E2E tests. |
| Prometheus client libraries | Metric types, registration, collection, and metric exposition support. | The controller exposes build and BuildRun health and duration metrics. | [`pkg/metrics`](../../pkg/metrics) and [`pkg/config`](../../pkg/config). |
| `logr` and `zap` | Structured logger interfaces and the Zap logging implementation. | Controller components need consistent, context-aware structured logs. | [`pkg/ctxlog`](../../pkg/ctxlog) and controller startup. |
| Cobra and pflag | Command and flag parsing for Go command-line programs. | Helper binaries and the controller expose consistent command-line options. | [`cmd/waiter`](../../cmd/waiter), other helper commands, and controller flags. |
| Ginkgo and Gomega | BDD-style test organization, execution, matchers, and assertions. | Unit, integration, and E2E tests share a common test style and runner. | Tests under [`pkg`](../../pkg), [`cmd`](../../cmd), and [`test`](../../test). |

The exact versions are maintained in [`go.mod`](../../go.mod). The repository also uses external command-line tools such as `ko`, `controller-gen`, `counterfeiter`, and `ginkgo`; their installation and Make targets are documented in [`DEVELOPMENT.md`](../../DEVELOPMENT.md), [`HACK.md`](../../HACK.md), and [`docs/development/testing.md`](testing.md).
