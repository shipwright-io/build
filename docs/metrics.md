<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

# Build Controller Metrics

The Build component exposes several metrics to help you monitor the health and behavior of your build resources.

Following build metrics are exposed on port `8383`.

| Name                                                 | Type      | Description                                                          | Labels                          | Status       |
|:-----------------------------------------------------|:----------|:---------------------------------------------------------------------|:--------------------------------|:-------------|
| `build_builds_registered_total`                      | Counter   | Number of total registered Builds (both successful and failed).      | `buildstrategy` `namespace` `build` `strategy_kind` `source_type` <sup>1</sup> | experimental |
| `build_buildrun_result_total`                        | Counter   | Number of total completed BuildRuns by result.                       | `buildstrategy` `namespace` `build` `buildrun` `strategy_kind` `executor` `source_type` `result` <sup>1</sup> | experimental |
| `build_buildrun_failure_reason_total`                | Counter   | Number of total failed BuildRuns by failure reason.                  | `buildstrategy` `namespace` `build` `buildrun` `strategy_kind` `executor` `source_type` <sup>1</sup> `reason` <sup>2</sup> | experimental |
| `build_buildruns_active`                             | Gauge     | Number of currently running BuildRuns.                               | `buildstrategy` `namespace` `build` `buildrun` `strategy_kind` `executor` `source_type` <sup>1</sup> | experimental |
| `build_buildstrategy_count`                          | Gauge     | Number of BuildStrategy and ClusterBuildStrategy objects.            | `strategy_kind` <sup>3</sup>    | experimental |
| `build_buildrun_establish_duration_seconds`          | Histogram | BuildRun establish duration in seconds.                              | `buildstrategy` `namespace` `build` `buildrun` `strategy_kind` `executor` `source_type` <sup>1</sup> | experimental |
| `build_buildrun_completion_duration_seconds`         | Histogram | BuildRun completion duration in seconds.                             | `buildstrategy` `namespace` `build` `buildrun` `strategy_kind` `executor` `source_type` <sup>1</sup> | experimental |
| `build_buildrun_rampup_duration_seconds`             | Histogram | BuildRun ramp-up duration in seconds                                 | `buildstrategy` `namespace` `build` `buildrun` `strategy_kind` `executor` `source_type` <sup>1</sup> | experimental |
| `build_buildrun_taskrun_rampup_duration_seconds`     | Histogram | BuildRun taskrun ramp-up duration in seconds.                        | `buildstrategy` `namespace` `build` `buildrun` `strategy_kind` `executor` `source_type` <sup>1</sup> | experimental |
| `build_buildrun_taskrun_pod_rampup_duration_seconds` | Histogram | BuildRun taskrun pod ramp-up duration in seconds.                    | `buildstrategy` `namespace` `build` `buildrun` `strategy_kind` `executor` `source_type` <sup>1</sup> | experimental |

<sup>1</sup> Labels for metric are disabled by default. See [Configuration of metric labels](#configuration-of-metric-labels) to enable them.

<sup>2</sup> The `reason` label on `build_buildrun_failure_reason_total` is always present (not opt-in) because it is the defining dimension of this metric. Values are a bounded set of failure reasons such as `StepOutOfMemory`, `PodEvicted`, `BuildRunTimeout`, `BuildRunCanceled`, etc.

<sup>3</sup> The `strategy_kind` label on `build_buildstrategy_count` is always present (not opt-in). Values are `BuildStrategy` or `ClusterBuildStrategy`.

## Configuration of histogram buckets

Environment variables can be set to use custom buckets for the histogram metrics:

| Metric                                               | Environment variable               | Default                                  |
|------------------------------------------------------|------------------------------------|------------------------------------------|
| `build_buildrun_establish_duration_seconds`          | `PROMETHEUS_BR_EST_DUR_BUCKETS`    | `0,1,2,3,5,7,10,15,20,30`                |
| `build_buildrun_completion_duration_seconds`         | `PROMETHEUS_BR_COMP_DUR_BUCKETS`   | `50,100,150,200,250,300,350,400,450,500` |
| `build_buildrun_rampup_duration_seconds`             | `PROMETHEUS_BR_RAMPUP_DUR_BUCKETS` | `0,1,2,3,4,5,6,7,8,9,10`                 |
| `build_buildrun_taskrun_rampup_duration_seconds`     | `PROMETHEUS_BR_RAMPUP_DUR_BUCKETS` | `0,1,2,3,4,5,6,7,8,9,10`                 |
| `build_buildrun_taskrun_pod_rampup_duration_seconds` | `PROMETHEUS_BR_RAMPUP_DUR_BUCKETS` | `0,1,2,3,4,5,6,7,8,9,10`                 |

The values have to be a comma-separated list of numbers. You need to set the environment variable for the build controller for your customization to become active. When running locally, set the variable right before starting the controller:

```bash
export PROMETHEUS_BR_COMP_DUR_BUCKETS=30,60,90,120,180,240,300,360,420,480
make local
```

When you deploy the build controller in a Kubernetes cluster, you need to extend the `spec.containers[0].spec.env` section of the sample deployment file, [controller.yaml](../deploy/500-controller.yaml). Add another entry:

```yaml
[...]
  env:
  - name: PROMETHEUS_BR_COMP_DUR_BUCKETS
    value: "30,60,90,120,180,240,300,360,420,480"
[...]
```

## Configuration of metric labels

As the amount of buckets and labels has a direct impact on the number of Prometheus time series, you can selectively enable labels that you are interested in using the `PROMETHEUS_ENABLED_LABELS` environment variable. The supported labels are:

* `buildstrategy` - The build strategy name
* `namespace` - The Kubernetes namespace
* `build` - The Build resource name
* `buildrun` - The BuildRun resource name
* `strategy_kind` - The strategy kind (`BuildStrategy` or `ClusterBuildStrategy`)
* `executor` - The executor type (`TaskRun` or `PipelineRun`)
* `result` - The build result (`succeeded`, `failed`, `cancelled`, `timeout`) - only on `build_buildrun_result_total`
* `source_type` - The source code delivery method (`Git`, `OCI`, `Local`)

Use a comma-separated value to enable multiple labels. For example:

```bash
export PROMETHEUS_ENABLED_LABELS=namespace
make local
```

or

```bash
export PROMETHEUS_ENABLED_LABELS=buildstrategy,namespace,build,strategy_kind,result
make local
```

When you deploy the build controller in a Kubernetes cluster, you need to extend the `spec.containers[0].spec.env` section of the sample deployment file, [controller.yaml](../deploy/500-controller.yaml). Add another entry:

```yaml
[...]
  env:
  - name: PROMETHEUS_ENABLED_LABELS
    value: namespace,strategy_kind,result
[...]
```
