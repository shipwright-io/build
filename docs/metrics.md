# Build Controller Metrics

The Build component exposes several metrics to help you monitor the health and behavior of your build resources.

Following build metrics are exposed at service `build-operator-metrics` on port `8383`.

| Name | Type | Description | Label | Status |
| ---- | ---- | ----------- | ----- | ------ |
| `build_builds_registered_total` | Counter | Number of total registered Builds. | buildstrategy=<build_buildstrategy_name> | experimental |
| `build_buildruns_completed_total` | Counter | Number of total completed BuildRuns. | buildstrategy=<build_buildstrategy_name> | experimental |
| `build_buildrun_establish_duration_seconds` | Histogram | BuildRun establish duration in seconds. | buildstrategy=<build_buildstrategy_name><br>namespace=<buildrun_namespace> | experimental |
| `build_buildrun_completion_duration_seconds` | Histogram | BuildRun completion duration in seconds. | buildstrategy=<build_buildstrategy_name><br>namespace=<buildrun_namespace> | experimental |
