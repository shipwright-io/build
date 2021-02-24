<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

# Configuration

The `build-operator` is installed into Kubernetes with reasonable defaults. However, there are some settings that can be overridden using environment variables in [`operator.yaml`](../deploy/operator.yaml).

The following environment variables are available:

| Environment Variable | Description |
| --- | --- |
| `CTX_TIMEOUT` | Override the default context timeout used for all Custom Resource Definition reconciliation operations. |
| `KANIKO_CONTAINER_IMAGE` | Specify the Kaniko container image to be used for the runtime image build instead of the default, for example `gcr.io/kaniko-project/executor:v1.5.1`. |
| `BUILD_OPERATOR_LEADER_ELECTION_NAMESPACE` |  Set the namespace to be used to store the `build-operator` lock, by default it is in the same namespace as the operator itself. |
| `BUILD_OPERATOR_LEASE_DURATION` |  Override the `LeaseDuration`, which is the duration that non-leader candidates will wait to force acquire leadership. |
| `BUILD_OPERATOR_RENEW_DEADLINE` |  Override the `RenewDeadline`, which is the duration that the acting master will retry refreshing leadership before giving up. |
| `BUILD_OPERATOR_RETRY_PERIOD` |  Override the `RetryPeriod`, which is the duration the LeaderElector clients should wait between tries of actions. |
