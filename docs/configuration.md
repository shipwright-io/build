<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

# Configuration

The controller is installed into Kubernetes with reasonable defaults. However, there are some settings that can be overridden using environment variables in [`controller.yaml`](../deploy/controller.yaml).

The following environment variables are available:

| Environment Variable | Description |
| --- | --- |
| `CTX_TIMEOUT` | Override the default context timeout used for all Custom Resource Definition reconciliation operations. |
| `KANIKO_CONTAINER_IMAGE` | Specify the Kaniko container image to be used for the runtime image build instead of the default, for example `gcr.io/kaniko-project/executor:v1.5.1`. |
| `BUILD_CONTROLLER_LEADER_ELECTION_NAMESPACE` |  Set the namespace to be used to store the `shipwright-build-controller` lock, by default it is in the same namespace as the controller itself. |
| `BUILD_CONTROLLER_LEASE_DURATION` |  Override the `LeaseDuration`, which is the duration that non-leader candidates will wait to force acquire leadership. |
| `BUILD_CONTROLLER_RENEW_DEADLINE` |  Override the `RenewDeadline`, which is the duration that the acting master will retry refreshing leadership before giving up. |
| `BUILD_CONTROLLER_RETRY_PERIOD` |  Override the `RetryPeriod`, which is the duration the LeaderElector clients should wait between tries of actions. |
