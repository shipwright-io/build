// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shipwright-io/build/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// Labels used in Prometheus metrics
const (
	BuildStrategyLabel string = "buildstrategy"
	NamespaceLabel     string = "namespace"
	BuildLabel         string = "build"
	BuildRunLabel      string = "buildrun"
)

var (
	buildCount    *prometheus.CounterVec
	buildRunCount *prometheus.CounterVec

	buildRunEstablishDuration  *prometheus.HistogramVec
	buildRunCompletionDuration *prometheus.HistogramVec

	buildRunRampUpDuration   *prometheus.HistogramVec
	taskRunRampUpDuration    *prometheus.HistogramVec
	taskRunPodRampUpDuration *prometheus.HistogramVec

	buildStrategyLabelEnabled = false
	namespaceLabelEnabled     = false
	buildLabelEnabled         = false
	buildRunLabelEnabled      = false

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

	buildCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "build_builds_registered_total",
			Help: "Number of total registered Builds.",
		},
		buildLabels)

	buildRunCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "build_buildruns_completed_total",
			Help: "Number of total completed BuildRuns.",
		},
		buildRunLabels)

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

	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(
		buildCount,
		buildRunCount,
		buildRunEstablishDuration,
		buildRunCompletionDuration,
		buildRunRampUpDuration,
		taskRunRampUpDuration,
		taskRunPodRampUpDuration,
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

func createBuildLabels(buildStrategy string, namespace string, build string) prometheus.Labels {
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

	return labels
}

func createBuildRunLabels(buildStrategy string, namespace string, build string, buildRun string) prometheus.Labels {
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

	return labels
}

// BuildCountInc increases a number of the existing build total count
func BuildCountInc(buildStrategy string, namespace string, build string) {
	if buildCount != nil {
		buildCount.With(createBuildLabels(buildStrategy, namespace, build)).Inc()
	}
}

// BuildRunCountInc increases a number of the existing build run total count
func BuildRunCountInc(buildStrategy string, namespace string, build string, buildRun string) {
	if buildRunCount != nil {
		buildRunCount.With(createBuildRunLabels(buildStrategy, namespace, build, buildRun)).Inc()
	}
}

// BuildRunEstablishObserve sets the build run establish time
func BuildRunEstablishObserve(buildStrategy string, namespace string, build string, buildRun string, duration time.Duration) {
	if buildRunEstablishDuration != nil {
		buildRunEstablishDuration.With(createBuildRunLabels(buildStrategy, namespace, build, buildRun)).Observe(duration.Seconds())
	}
}

// BuildRunCompletionObserve sets the build run completion time
func BuildRunCompletionObserve(buildStrategy string, namespace string, build string, buildRun string, duration time.Duration) {
	if buildRunCompletionDuration != nil {
		buildRunCompletionDuration.With(createBuildRunLabels(buildStrategy, namespace, build, buildRun)).Observe(duration.Seconds())
	}
}

// BuildRunRampUpDurationObserve processes the observation of a new buildrun ramp-up duration
func BuildRunRampUpDurationObserve(buildStrategy string, namespace string, build string, buildRun string, duration time.Duration) {
	if buildRunRampUpDuration != nil {
		buildRunRampUpDuration.With(createBuildRunLabels(buildStrategy, namespace, build, buildRun)).Observe(duration.Seconds())
	}
}

// TaskRunRampUpDurationObserve processes the observation of a new taskrun ramp-up duration
func TaskRunRampUpDurationObserve(buildStrategy string, namespace string, build string, buildRun string, duration time.Duration) {
	if taskRunRampUpDuration != nil {
		taskRunRampUpDuration.With(createBuildRunLabels(buildStrategy, namespace, build, buildRun)).Observe(duration.Seconds())
	}
}

// TaskRunPodRampUpDurationObserve processes the observation of a new taskrun pod ramp-up duration
func TaskRunPodRampUpDurationObserve(buildStrategy string, namespace string, build string, buildRun string, duration time.Duration) {
	if taskRunPodRampUpDuration != nil {
		taskRunPodRampUpDuration.With(createBuildRunLabels(buildStrategy, namespace, build, buildRun)).Observe(duration.Seconds())
	}
}
