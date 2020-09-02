<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

# Build Controller Metrics

The Build component exposes several metrics to help you monitor the health and behavior of your build resources.

Following build metrics are exposed at service `build-operator-metrics` on port `8383`.

| Name | Type | Description | Label | Status |
| ---- | ---- | ----------- | ----- | ------ |
| `build_builds_registered_total` | Counter | Number of total registered Builds. | buildstrategy=<build_buildstrategy_name> | experimental |
| `build_buildruns_completed_total` | Counter | Number of total completed BuildRuns. | buildstrategy=<build_buildstrategy_name> | experimental |
| `build_buildrun_establish_duration_seconds` | Histogram | BuildRun establish duration in seconds. | buildstrategy=<build_buildstrategy_name><br>namespace=<buildrun_namespace> | experimental |
| `build_buildrun_completion_duration_seconds` | Histogram | BuildRun completion duration in seconds. | buildstrategy=<build_buildstrategy_name><br>namespace=<buildrun_namespace> | experimental |
| `build_buildrun_rampup_duration_seconds` | Histogram | BuildRun ramp-up duration in seconds | buildstrategy=<build_buildstrategy_name><br>namespace=<buildrun_namespace> | experimental |
| `build_buildrun_taskrun_rampup_duration_seconds` | Histogram | BuildRun taskrun ramp-up duration in seconds. | buildstrategy=<build_buildstrategy_name><br>namespace=<buildrun_namespace> | experimental |
| `build_buildrun_taskrun_pod_rampup_duration_seconds` | Histogram | BuildRun taskrun pod ramp-up duration in seconds. | buildstrategy=<build_buildstrategy_name><br>namespace=<buildrun_namespace> | experimental |

Environment variables can be set to use custom buckets for the histogram metrics:

| Metric                                               | Environment variable               | Default                                  |
| ---------------------------------------------------- | ---------------------------------- | ---------------------------------------- |
| `build_buildrun_establish_duration_seconds`          | `PROMETHEUS_BR_EST_DUR_BUCKETS`    | `0,1,2,3,5,7,10,15,20,30`                |
| `build_buildrun_completion_duration_seconds`         | `PROMETHEUS_BR_COMP_DUR_BUCKETS`   | `50,100,150,200,250,300,350,400,450,500` |
| `build_buildrun_rampup_duration_seconds`             | `PROMETHEUS_BR_RAMPUP_DUR_BUCKETS` | `0,1,2,3,4,5,6,7,8,9,10`                 |
| `build_buildrun_taskrun_rampup_duration_seconds`     | `PROMETHEUS_BR_RAMPUP_DUR_BUCKETS` | `0,1,2,3,4,5,6,7,8,9,10`                 |
| `build_buildrun_taskrun_pod_rampup_duration_seconds` | `PROMETHEUS_BR_RAMPUP_DUR_BUCKETS` | `0,1,2,3,4,5,6,7,8,9,10`                 |

The values have to be a comma-separated list of numbers. You need to set the environment variable for the build operator for your customization to become active. When running locally, set the variable right before starting the operator:

```bash
export PROMETHEUS_BR_COMP_DUR_BUCKETS=30,60,90,120,180,240,300,360,420,480
make local
```

When you are deploying the build operator in your Kubernetes cluster, you need to extend the `spec.containers[0].spec.env` section of the sample deployment file, [operator.yaml](../deploy/operator.yaml), to add an additional entry:

```yaml
[...]
  env:
  - name: PROMETHEUS_BR_COMP_DUR_BUCKETS
    value: "30,60,90,120,180,240,300,360,420,480"
[...]
```
