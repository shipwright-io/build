<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

# Configuration

The controller is installed into Kubernetes with reasonable defaults. However, there are some settings that can be overridden using environment variables in [`controller.yaml`](../deploy/500-controller.yaml).

The following environment variables are available:

| Environment Variable | Description |
| --- | --- |
| `CTX_TIMEOUT` | Override the default context timeout used for all Custom Resource Definition reconciliation operations. |
| `KANIKO_CONTAINER_IMAGE` | Specify the Kaniko container image to be used for the runtime image build instead of the default, for example `gcr.io/kaniko-project/executor:v1.5.1`. |
| `BUILD_CONTROLLER_LEADER_ELECTION_NAMESPACE` |  Set the namespace to be used to store the `shipwright-build-controller` lock, by default it is in the same namespace as the controller itself. |
| `BUILD_CONTROLLER_LEASE_DURATION` |  Override the `LeaseDuration`, which is the duration that non-leader candidates will wait to force acquire leadership. |
| `BUILD_CONTROLLER_RENEW_DEADLINE` |  Override the `RenewDeadline`, which is the duration that the acting master will retry refreshing leadership before giving up. |
| `BUILD_CONTROLLER_RETRY_PERIOD` |  Override the `RetryPeriod`, which is the duration the LeaderElector clients should wait between tries of actions. |
| `BUILD_MAX_CONCURRENT_RECONCILES` | The number of concurrent reconciles by the build controller. A value of 0 or lower will use the default from the [controller-runtime controller Options](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/controller#Options). Default is 0. |
| `BUILDRUN_MAX_CONCURRENT_RECONCILES` | The number of concurrent reconciles by the buildrun controller. A value of 0 or lower will use the default from the [controller-runtime controller Options](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/controller#Options). Default is 0. |
| `BUILDSTRATEGY_MAX_CONCURRENT_RECONCILES` | The number of concurrent reconciles by the buildstrategy controller. A value of 0 or lower will use the default from the [controller-runtime controller Options](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/controller#Options). Default is 0. |
| `CLUSTERBUILDSTRATEGY_MAX_CONCURRENT_RECONCILES` | The number of concurrent reconciles by the clusterbuildstrategy controller. A value of 0 or lower will use the default from the [controller-runtime controller Options](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/controller#Options). Default is 0. |
| `KUBE_API_BURST` | Burst to use for the Kubernetes API client. See [Config.Burst](https://pkg.go.dev/k8s.io/client-go/rest#Config.Burst). A value of 0 or lower will use the default from client-go, which currently is 10. Default is 0. |
| `KUBE_API_QPS` | QPS to use for the Kubernetes API client. See [Config.QPS](https://pkg.go.dev/k8s.io/client-go/rest#Config.QPS). A value of 0 or lower will use the default from client-go, which currently is 5. Default is 0. |
