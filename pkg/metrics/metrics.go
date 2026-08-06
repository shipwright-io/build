// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/shipwright-io/build/pkg/config"
)

// Labels used in Prometheus metrics
const (
	BuildStrategyLabel string = "buildstrategy"
	NamespaceLabel     string = "namespace"
	BuildLabel         string = "build"
	BuildRunLabel      string = "buildrun"
	StrategyKindLabel  string = "strategy_kind"
	ExecutorLabel      string = "executor"
	ResultLabel        string = "result"
	SourceTypeLabel    string = "source_type"
)

// Result label values for build_buildrun_result_total
const (
	ResultSucceeded = "succeeded"
	ResultFailed    = "failed"
	ResultCancelled = "cancelled"
	ResultTimeout   = "timeout"
)

var (
	buildCount *prometheus.CounterVec

	buildRunEstablishDuration  *prometheus.HistogramVec
	buildRunCompletionDuration *prometheus.HistogramVec

	buildRunRampUpDuration   *prometheus.HistogramVec
	taskRunRampUpDuration    *prometheus.HistogramVec
	taskRunPodRampUpDuration *prometheus.HistogramVec

	buildRunResultCount        *prometheus.CounterVec
	buildRunFailureReasonCount *prometheus.CounterVec
	buildRunsActive            *prometheus.GaugeVec
	buildStrategyCount         *prometheus.GaugeVec

	buildStrategyLabelEnabled = false
	namespaceLabelEnabled     = false
	buildLabelEnabled         = false
	buildRunLabelEnabled      = false
	strategyKindLabelEnabled  = false
	executorLabelEnabled      = false
	resultLabelEnabled        = false
	sourceTypeLabelEnabled    = false

	initialized = false
)

// Optional additional metrics endpoint handlers to be configured
var metricsExtraHandlers = map[string]http.HandlerFunc{}

// InitPrometheus initializes the prometheus stuff
func InitPrometheus(config *config.Config) {
	if initialized {
		return
	}

	initialized = true

	var buildLabels []string
	var buildRunLabels []string
	if contains(config.Prometheus.EnabledLabels, BuildStrategyLabel) {
		buildLabels = append(buildLabels, BuildStrategyLabel)
		buildRunLabels = append(buildRunLabels, BuildStrategyLabel)
		buildStrategyLabelEnabled = true
	}
	if contains(config.Prometheus.EnabledLabels, NamespaceLabel) {
		buildLabels = append(buildLabels, NamespaceLabel)
		buildRunLabels = append(buildRunLabels, NamespaceLabel)
		namespaceLabelEnabled = true
	}
	if contains(config.Prometheus.EnabledLabels, BuildLabel) {
		buildLabels = append(buildLabels, BuildLabel)
		buildRunLabels = append(buildRunLabels, BuildLabel)
		buildLabelEnabled = true
	}
	if contains(config.Prometheus.EnabledLabels, BuildRunLabel) {
		buildRunLabels = append(buildRunLabels, BuildRunLabel)
		buildRunLabelEnabled = true
	}
	if contains(config.Prometheus.EnabledLabels, StrategyKindLabel) {
		buildLabels = append(buildLabels, StrategyKindLabel)
		buildRunLabels = append(buildRunLabels, StrategyKindLabel)
		strategyKindLabelEnabled = true
	}
	if contains(config.Prometheus.EnabledLabels, ExecutorLabel) {
		buildRunLabels = append(buildRunLabels, ExecutorLabel)
		executorLabelEnabled = true
	}
	if contains(config.Prometheus.EnabledLabels, SourceTypeLabel) {
		buildLabels = append(buildLabels, SourceTypeLabel)
		buildRunLabels = append(buildRunLabels, SourceTypeLabel)
		sourceTypeLabelEnabled = true
	}

	// buildRunResultLabels includes the result label for build_buildrun_result_total
	buildRunResultLabels := make([]string, len(buildRunLabels))
	copy(buildRunResultLabels, buildRunLabels)
	if contains(config.Prometheus.EnabledLabels, ResultLabel) {
		buildRunResultLabels = append(buildRunResultLabels, ResultLabel)
		resultLabelEnabled = true
	}

	// buildRunFailureReasonLabels always includes the reason label
	buildRunFailureReasonLabels := make([]string, len(buildRunLabels))
	copy(buildRunFailureReasonLabels, buildRunLabels)
	buildRunFailureReasonLabels = append(buildRunFailureReasonLabels, "reason")

	buildCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "build_builds_registered_total",
			Help: "Number of total registered Builds.",
		},
		buildLabels)

	buildRunEstablishDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "build_buildrun_establish_duration_seconds",
			Help:    "BuildRun establish duration in seconds.",
			Buckets: config.Prometheus.BuildRunEstablishDurationBuckets,
		},
		buildRunLabels)

	buildRunCompletionDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "build_buildrun_completion_duration_seconds",
			Help:    "BuildRun completion duration in seconds.",
			Buckets: config.Prometheus.BuildRunCompletionDurationBuckets,
		},
		buildRunLabels)

	buildRunRampUpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "build_buildrun_rampup_duration_seconds",
			Help:    "BuildRun ramp-up duration in seconds (time between buildrun creation and taskrun creation).",
			Buckets: config.Prometheus.BuildRunRampUpDurationBuckets,
		},
		buildRunLabels)

	taskRunRampUpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "build_buildrun_taskrun_rampup_duration_seconds",
			Help:    "BuildRun taskrun ramp-up duration in seconds (time between taskrun creation and taskrun pod creation).",
			Buckets: config.Prometheus.BuildRunRampUpDurationBuckets,
		},
		buildRunLabels)

	taskRunPodRampUpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "build_buildrun_taskrun_pod_rampup_duration_seconds",
			Help:    "BuildRun taskrun pod ramp-up duration in seconds (time between pod creation and last init container completion).",
			Buckets: config.Prometheus.BuildRunRampUpDurationBuckets,
		},
		buildRunLabels)

	buildRunResultCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "build_buildrun_result_total",
			Help: "Number of total completed BuildRuns by result.",
		},
		buildRunResultLabels)

	buildRunFailureReasonCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "build_buildrun_failure_reason_total",
			Help: "Number of total failed BuildRuns by failure reason.",
		},
		buildRunFailureReasonLabels)

	buildRunsActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "build_buildruns_active",
			Help: "Number of currently running BuildRuns.",
		},
		buildRunLabels)

	buildStrategyCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "build_buildstrategy_count",
			Help: "Number of BuildStrategy and ClusterBuildStrategy objects.",
		},
		[]string{StrategyKindLabel})

	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(
		buildCount,
		buildRunEstablishDuration,
		buildRunCompletionDuration,
		buildRunRampUpDuration,
		taskRunRampUpDuration,
		taskRunPodRampUpDuration,
		buildRunResultCount,
		buildRunFailureReasonCount,
		buildRunsActive,
		buildStrategyCount,
	)
}

// ExtraHandlers returns a mapping of paths and their respective
// additional HTTP handlers to be configured at the metrics listener
func ExtraHandlers() map[string]http.HandlerFunc {
	return metricsExtraHandlers
}

func contains(slice []string, element string) bool {
	for _, candidate := range slice {
		if candidate == element {
			return true
		}
	}
	return false
}

func createBuildLabels(buildStrategy, namespace, build, strategyKind, sourceType string) prometheus.Labels {
	labels := prometheus.Labels{}

	if buildStrategyLabelEnabled {
		labels[BuildStrategyLabel] = buildStrategy
	}
	if namespaceLabelEnabled {
		labels[NamespaceLabel] = namespace
	}
	if buildLabelEnabled {
		labels[BuildLabel] = build
	}
	if strategyKindLabelEnabled {
		labels[StrategyKindLabel] = strategyKind
	}
	if sourceTypeLabelEnabled {
		labels[SourceTypeLabel] = sourceType
	}

	return labels
}

func createBuildRunLabels(buildStrategy, namespace, build, buildRun, strategyKind, executor, sourceType string) prometheus.Labels {
	labels := prometheus.Labels{}

	if buildStrategyLabelEnabled {
		labels[BuildStrategyLabel] = buildStrategy
	}
	if namespaceLabelEnabled {
		labels[NamespaceLabel] = namespace
	}
	if buildLabelEnabled {
		labels[BuildLabel] = build
	}
	if buildRunLabelEnabled {
		labels[BuildRunLabel] = buildRun
	}
	if strategyKindLabelEnabled {
		labels[StrategyKindLabel] = strategyKind
	}
	if executorLabelEnabled {
		labels[ExecutorLabel] = executor
	}
	if sourceTypeLabelEnabled {
		labels[SourceTypeLabel] = sourceType
	}

	return labels
}

func createBuildRunResultLabels(buildStrategy, namespace, build, buildRun, strategyKind, executor, result, sourceType string) prometheus.Labels {
	labels := createBuildRunLabels(buildStrategy, namespace, build, buildRun, strategyKind, executor, sourceType)
	if resultLabelEnabled {
		labels[ResultLabel] = result
	}
	return labels
}

func createBuildRunFailureReasonLabels(buildStrategy, namespace, build, buildRun, strategyKind, executor, sourceType, reason string) prometheus.Labels {
	labels := createBuildRunLabels(buildStrategy, namespace, build, buildRun, strategyKind, executor, sourceType)
	labels["reason"] = reason
	return labels
}

// BuildCountInc increases the total count of registered Builds
func BuildCountInc(buildStrategy, namespace, build, strategyKind, sourceType string) {
	if buildCount != nil {
		buildCount.With(createBuildLabels(buildStrategy, namespace, build, strategyKind, sourceType)).Inc()
	}
}

// BuildRunEstablishObserve sets the build run establish time
func BuildRunEstablishObserve(buildStrategy, namespace, build, buildRun, strategyKind, executor, sourceType string, duration time.Duration) {
	if buildRunEstablishDuration != nil {
		buildRunEstablishDuration.With(createBuildRunLabels(buildStrategy, namespace, build, buildRun, strategyKind, executor, sourceType)).Observe(duration.Seconds())
	}
}

// BuildRunCompletionObserve sets the build run completion time
func BuildRunCompletionObserve(buildStrategy, namespace, build, buildRun, strategyKind, executor, sourceType string, duration time.Duration) {
	if buildRunCompletionDuration != nil {
		buildRunCompletionDuration.With(createBuildRunLabels(buildStrategy, namespace, build, buildRun, strategyKind, executor, sourceType)).Observe(duration.Seconds())
	}
}

// BuildRunRampUpDurationObserve processes the observation of a new buildrun ramp-up duration
func BuildRunRampUpDurationObserve(buildStrategy, namespace, build, buildRun, strategyKind, executor, sourceType string, duration time.Duration) {
	if buildRunRampUpDuration != nil {
		buildRunRampUpDuration.With(createBuildRunLabels(buildStrategy, namespace, build, buildRun, strategyKind, executor, sourceType)).Observe(duration.Seconds())
	}
}

// TaskRunRampUpDurationObserve processes the observation of a new taskrun ramp-up duration
func TaskRunRampUpDurationObserve(buildStrategy, namespace, build, buildRun, strategyKind, executor, sourceType string, duration time.Duration) {
	if taskRunRampUpDuration != nil {
		taskRunRampUpDuration.With(createBuildRunLabels(buildStrategy, namespace, build, buildRun, strategyKind, executor, sourceType)).Observe(duration.Seconds())
	}
}

// TaskRunPodRampUpDurationObserve processes the observation of a new taskrun pod ramp-up duration
func TaskRunPodRampUpDurationObserve(buildStrategy, namespace, build, buildRun, strategyKind, executor, sourceType string, duration time.Duration) {
	if taskRunPodRampUpDuration != nil {
		taskRunPodRampUpDuration.With(createBuildRunLabels(buildStrategy, namespace, build, buildRun, strategyKind, executor, sourceType)).Observe(duration.Seconds())
	}
}

// BuildRunResultCountInc increments the build_buildrun_result_total counter
func BuildRunResultCountInc(buildStrategy, namespace, build, buildRun, strategyKind, executor, result, sourceType string) {
	if buildRunResultCount != nil {
		buildRunResultCount.With(createBuildRunResultLabels(buildStrategy, namespace, build, buildRun, strategyKind, executor, result, sourceType)).Inc()
	}
}

// BuildRunFailureReasonCountInc increments the build_buildrun_failure_reason_total counter
func BuildRunFailureReasonCountInc(buildStrategy, namespace, build, buildRun, strategyKind, executor, sourceType, reason string) {
	if buildRunFailureReasonCount != nil {
		buildRunFailureReasonCount.With(createBuildRunFailureReasonLabels(buildStrategy, namespace, build, buildRun, strategyKind, executor, sourceType, reason)).Inc()
	}
}

// BuildRunsActiveInc increments the build_buildruns_active gauge
func BuildRunsActiveInc(buildStrategy, namespace, build, buildRun, strategyKind, executor, sourceType string) {
	if buildRunsActive != nil {
		buildRunsActive.With(createBuildRunLabels(buildStrategy, namespace, build, buildRun, strategyKind, executor, sourceType)).Inc()
	}
}

// BuildRunsActiveDec decrements the build_buildruns_active gauge
func BuildRunsActiveDec(buildStrategy, namespace, build, buildRun, strategyKind, executor, sourceType string) {
	if buildRunsActive != nil {
		buildRunsActive.With(createBuildRunLabels(buildStrategy, namespace, build, buildRun, strategyKind, executor, sourceType)).Dec()
	}
}

// BuildStrategyCountSet sets the build_buildstrategy_count gauge
func BuildStrategyCountSet(strategyKind string, count float64) {
	if buildStrategyCount != nil {
		buildStrategyCount.With(prometheus.Labels{StrategyKindLabel: strategyKind}).Set(count)
	}
}
