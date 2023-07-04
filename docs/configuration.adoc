<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

# Configuration

## Controller Settings

The controller is installed into Kubernetes with reasonable defaults. However, there are some settings that can be overridden using environment variables in [`controller.yaml`](../deploy/500-controller.yaml).

The following environment variables are available:

| Environment Variable | Description |
| --- | --- |
| `CTX_TIMEOUT` | Override the default context timeout used for all Custom Resource Definition reconciliation operations. Default is 5 (seconds). |
| `REMOTE_ARTIFACTS_CONTAINER_IMAGE` | Specify the container image used for the `.spec.sources` remote artifacts download, by default it uses `quay.io/quay/busybox:latest`. |
| `TERMINATION_LOG_PATH` | Path of the termination log. This is where controller application will write the reason of its termination. Default value is `/dev/termination-log`. |
| `GIT_ENABLE_REWRITE_RULE` | Enable Git wrapper to setup a URL `insteadOf` Git config rewrite rule for the respective source URL hostname. Default is `false`. |
| `GIT_CONTAINER_TEMPLATE` | JSON representation of a [Container] template that is used for steps that clone a Git repository. Default is `{"image": "ghcr.io/shipwright-io/build/git:latest", "command": ["/ko-app/git"], "env": [{"name": "HOME", "value": "/shared-home"}], "securityContext":{"allowPrivilegeEscalation": false, "capabilities": {"drop": ["ALL"]}, "runAsUser": 1000,"runAsGroup": 1000}}` [^1]. The following properties are ignored as they are set by the controller: `args`, `name`. |
| `GIT_CONTAINER_IMAGE` | Custom container image for Git clone steps. If `GIT_CONTAINER_TEMPLATE` is also specifying an image, then the value for `GIT_CONTAINER_IMAGE` has precedence. |
| `BUNDLE_IMAGE_CONTAINER_TEMPLATE` | JSON representation of a [Container] template that is used for steps that pulls a bundle image to obtain the packaged source code. Default is `{"image": "ghcr.io/shipwright-io/build/bundle:latest", "command": ["/ko-app/bundle"], "env": [{"name": "HOME","value": "/shared-home"}], "securityContext":{"allowPrivilegeEscalation": false, "capabilities": {"drop": ["ALL"]}, "runAsUser":1000,"runAsGroup":1000}}` [^1]. The following properties are ignored as they are set by the controller: `args`, `name`. |
| `BUNDLE_IMAGE_CONTAINER_IMAGE` | Custom container image that pulls a bundle image to obtain the packaged source code. If `BUNDLE_IMAGE_CONTAINER_TEMPLATE` is also specifying an image, then the value for `BUNDLE_IMAGE_CONTAINER_IMAGE` has precedence. |
| `IMAGE_PROCESSING_CONTAINER_TEMPLATE` | JSON representation of a [Container](https://pkg.go.dev/k8s.io/api/core/v1#Container) template that is used for steps that processes the image. Default is `{"image": "ghcr.io/shipwright-io/build/image-processing:latest", "command": ["/ko-app/image-processing"], "env": [{"name": "HOME","value": "/shared-home"}], "securityContext": {"allowPrivilegeEscalation": false, "capabilities": {"add": ["DAC_OVERRIDE"], "drop": ["ALL"]}, "runAsUser": 0, "runAsgGroup": 0}}`. The following properties are ignored as they are set by the controller: `args`, `name`. |
| `IMAGE_PROCESSING_CONTAINER_IMAGE` | Custom container image that is used for steps that processes the image. If `IMAGE_PROCESSING_CONTAINER_TEMPLATE` is also specifying an image, then the value for `IMAGE_PROCESSING_CONTAINER_IMAGE` has precedence. |
| `WAITER_IMAGE_CONTAINER_TEMPLATE` | JSON representation of a [Container] template that waits for local source code to be uploaded to it. Default is `{"image":"ghcr.io/shipwright-io/build/waiter:latest", "command": ["/ko-app/waiter"], "args": ["start"], "env": [{"name": "HOME","value": "/shared-home"}], "securityContext":{"allowPrivilegeEscalation": false, "capabilities": {"drop": ["ALL"]}, "runAsUser":1000,"runAsGroup":1000}}`. The following properties are ignored as they are set by the controller: `args`, `name`. |
| `WAITER_IMAGE_CONTAINER_IMAGE` | Custom container image that waits for local source code to be uploaded to it. If `WAITER_IMAGE_CONTAINER_TEMPLATE` is also specifying an image, then the value for `WAITER_IMAGE_CONTAINER_IMAGE` has precedence. |
| `BUILD_CONTROLLER_LEADER_ELECTION_NAMESPACE` |  Set the namespace to be used to store the `shipwright-build-controller` lock, by default it is in the same namespace as the controller itself. |
| `BUILD_CONTROLLER_LEASE_DURATION` |  Override the `LeaseDuration`, which is the duration that non-leader candidates will wait to force acquire leadership. |
| `BUILD_CONTROLLER_RENEW_DEADLINE` |  Override the `RenewDeadline`, which is the duration that the acting leader will retry refreshing leadership before giving up. |
| `BUILD_CONTROLLER_RETRY_PERIOD` |  Override the `RetryPeriod`, which is the duration the LeaderElector clients should wait between tries of actions. |
| `BUILD_MAX_CONCURRENT_RECONCILES` | The number of concurrent reconciles by the build controller. A value of 0 or lower will use the default from the [controller-runtime controller Options]. Default is 0. |
| `BUILDRUN_MAX_CONCURRENT_RECONCILES` | The number of concurrent reconciles by the BuildRun controller. A value of 0 or lower will use the default from the [controller-runtime controller Options]. Default is 0. |
| `BUILDSTRATEGY_MAX_CONCURRENT_RECONCILES` | The number of concurrent reconciles by the BuildStrategy controller. A value of 0 or lower will use the default from the [controller-runtime controller Options]. Default is 0. |
| `CLUSTERBUILDSTRATEGY_MAX_CONCURRENT_RECONCILES` | The number of concurrent reconciles by the ClusterBuildStrategy controller. A value of 0 or lower will use the default from the [controller-runtime controller Options]. Default is 0. |
| `KUBE_API_BURST` | Burst to use for the Kubernetes API client. See [Config.Burst]. A value of 0 or lower will use the default from client-go, which currently is 10. Default is 0. |
| `KUBE_API_QPS` | QPS to use for the Kubernetes API client. See [Config.QPS]. A value of 0 or lower will use the default from client-go, which currently is 5. Default is 0. |

[^1]: The `runAsUser` and `runAsGroup` are dynamically overwritten depending on the build strategy that is used. See [Security Contexts](buildstrategies.md#security-contexts) for more information.

## Role-based Access Control

The release deployment YAML file includes two cluster-wide roles for using Shipwright Build objects.
The following roles are installed:

- `shpwright-build-aggregate-view`: this role grants read access (get, list, watch) to most Shipwright Build objects.
  This includes `BuildStrategy`, `ClusterBuildStrategy`, `Build`, and `BuildRun` objects.
  This role is aggregated to the [Kubernetes "view" role].
- `shipwright-build-aggregate-edit`: this role grants write access (create, update, patch, delete) to Shipwright objects that are namespace-scoped.
  This includes `BuildStrategy`, `Builds`, and `BuildRuns`.
  Read access is granted to all `ClusterBuildStrategy` objects.
  This role is aggregated to the [Kubernetes "edit" and "admin" roles].

Only cluster administrators are granted write access to `ClusterBuildStrategy` objects.
This can be changed by creating a separate [Kubernetes `ClusterRole`] with these permissions and binding the role to appropriate users.

[Container]:https://pkg.go.dev/k8s.io/api/core/v1#Container
[controller-runtime controller Options]:https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/controller#Options
[Config.Burst]:https://pkg.go.dev/k8s.io/client-go/rest#Config.Burst
[Config.QPS]:https://pkg.go.dev/k8s.io/client-go/rest#Config.QPS
[Kubernetes "view" role]:https://kubernetes.io/docs/reference/access-authn-authz/rbac/#default-roles-and-role-bindings
[Kubernetes "edit" and "admin" roles]:https://kubernetes.io/docs/reference/access-authn-authz/rbac/#default-roles-and-role-bindings
[Kubernetes `ClusterRole`]:https://kubernetes.io/docs/reference/access-authn-authz/rbac/#role-and-clusterrole
