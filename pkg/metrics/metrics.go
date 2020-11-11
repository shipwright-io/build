// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/shipwright-io/build/pkg/config"
)

const (
	BuildStrategyLabel string = "buildstrategy"
	NamespaceLabel     string = "namespace"
	BuildRunLabel      string = "buildrun"
)

var (
	buildCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "build_builds_registered_total",
			Help: "Number of total registered Builds.",
		},
		[]string{BuildStrategyLabel})

	buildRunCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "build_buildruns_completed_total",
			Help: "Number of total completed BuildRuns.",
		},
		[]string{BuildStrategyLabel})

	buildRunEstablishDuration  *prometheus.HistogramVec
	buildRunCompletionDuration *prometheus.HistogramVec

	buildRunRampUpDuration   *prometheus.HistogramVec
	taskRunRampUpDuration    *prometheus.HistogramVec
	taskRunPodRampUpDuration *prometheus.HistogramVec

	histogramBuildStrategyLabelEnabled = false
	histogramNamespaceLabelEnabled     = false
	histogramBuildRunLabelEnabled      = false

	initialized = false
)

// InitPrometheus initializes the prometheus stuff
func InitPrometheus(config *config.Config) {
	if initialized {
		return
	}

	initialized = true

	var histogramLabels []string
	if contains(config.Prometheus.HistogramEnabledLabels, BuildStrategyLabel) {
		histogramLabels = append(histogramLabels, BuildStrategyLabel)
		histogramBuildStrategyLabelEnabled = true
	}
	if contains(config.Prometheus.HistogramEnabledLabels, NamespaceLabel) {
		histogramLabels = append(histogramLabels, NamespaceLabel)
		histogramNamespaceLabelEnabled = true
	}
	if contains(config.Prometheus.HistogramEnabledLabels, BuildRunLabel) {
		histogramLabels = append(histogramLabels, BuildRunLabel)
		histogramBuildRunLabelEnabled = true
	}

	buildRunEstablishDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "build_buildrun_establish_duration_seconds",
			Help:    "BuildRun establish duration in seconds.",
			Buckets: config.Prometheus.BuildRunEstablishDurationBuckets,
		},
		histogramLabels)

	buildRunCompletionDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "build_buildrun_completion_duration_seconds",
			Help:    "BuildRun completion duration in seconds.",
			Buckets: config.Prometheus.BuildRunCompletionDurationBuckets,
		},
		histogramLabels)

	buildRunRampUpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "build_buildrun_rampup_duration_seconds",
			Help:    "BuildRun ramp-up duration in seconds (time between buildrun creation and taskrun creation).",
			Buckets: config.Prometheus.BuildRunRampUpDurationBuckets,
		},
		histogramLabels)

	taskRunRampUpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "build_buildrun_taskrun_rampup_duration_seconds",
			Help:    "BuildRun taskrun ramp-up duration in seconds (time between taskrun creation and taskrun pod creation).",
			Buckets: config.Prometheus.BuildRunRampUpDurationBuckets,
		},
		histogramLabels)

	taskRunPodRampUpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "build_buildrun_taskrun_pod_rampup_duration_seconds",
			Help:    "BuildRun taskrun pod ramp-up duration in seconds (time between pod creation and last init container completion).",
			Buckets: config.Prometheus.BuildRunRampUpDurationBuckets,
		},
		histogramLabels)

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

func contains(slice []string, element string) bool {
	for _, candidate := range slice {
		if candidate == element {
			return true
		}
	}
	return false
}

func createHistogramLabels(buildStrategy string, namespace string, buildRun string) prometheus.Labels {
	labels := prometheus.Labels{}

	if histogramBuildStrategyLabelEnabled {
		labels[BuildStrategyLabel] = buildStrategy
	}
	if histogramNamespaceLabelEnabled {
		labels[NamespaceLabel] = namespace
	}
	if histogramBuildRunLabelEnabled {
		labels[BuildRunLabel] = buildRun
	}

	return labels
}

// BuildCountInc increases a number of the existing build total count
func BuildCountInc(buildStrategy string) {
	buildCount.WithLabelValues(buildStrategy).Inc()
}

// BuildRunCountInc increases a number of the existing build run total count
func BuildRunCountInc(buildStrategy string) {
	if buildRunCount != nil {
		buildRunCount.WithLabelValues(buildStrategy).Inc()
	}
}

// BuildRunEstablishObserve sets the build run establish time
func BuildRunEstablishObserve(buildStrategy string, namespace string, buildRun string, duration time.Duration) {
	if buildRunEstablishDuration != nil {
		buildRunEstablishDuration.With(createHistogramLabels(buildStrategy, namespace, buildRun)).Observe(duration.Seconds())
	}
}

// BuildRunCompletionObserve sets the build run completion time
func BuildRunCompletionObserve(buildStrategy string, namespace string, buildRun string, duration time.Duration) {
	if buildRunCompletionDuration != nil {
		buildRunCompletionDuration.With(createHistogramLabels(buildStrategy, namespace, buildRun)).Observe(duration.Seconds())
	}
}

// BuildRunRampUpDurationObserve processes the observation of a new buildrun ramp-up duration
func BuildRunRampUpDurationObserve(buildStrategy string, namespace string, buildRun string, duration time.Duration) {
	if buildRunRampUpDuration != nil {
		buildRunRampUpDuration.With(createHistogramLabels(buildStrategy, namespace, buildRun)).Observe(duration.Seconds())
	}
}

// TaskRunRampUpDurationObserve processes the observation of a new taskrun ramp-up duration
func TaskRunRampUpDurationObserve(buildStrategy string, namespace string, buildRun string, duration time.Duration) {
	if taskRunRampUpDuration != nil {
		taskRunRampUpDuration.With(createHistogramLabels(buildStrategy, namespace, buildRun)).Observe(duration.Seconds())
	}
}

// TaskRunPodRampUpDurationObserve processes the observation of a new taskrun pod ramp-up duration
func TaskRunPodRampUpDurationObserve(buildStrategy string, namespace string, buildRun string, duration time.Duration) {
	if taskRunPodRampUpDuration != nil {
		taskRunPodRampUpDuration.With(createHistogramLabels(buildStrategy, namespace, buildRun)).Observe(duration.Seconds())
	}
}
