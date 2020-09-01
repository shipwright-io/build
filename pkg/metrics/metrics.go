// Copyright The Shipwright Contributors
// 
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shipwright-io/build/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	buildStrategyLabel string = "buildstrategy"
	namespaceLabel     string = "namespace"
)

var (
	buildCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "build_builds_registered_total",
			Help: "Number of total registered Builds.",
		},
		[]string{buildStrategyLabel})

	buildRunCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "build_buildruns_completed_total",
			Help: "Number of total completed BuildRuns.",
		},
		[]string{buildStrategyLabel})

	buildRunEstablishDuration  *prometheus.HistogramVec
	buildRunCompletionDuration *prometheus.HistogramVec

	buildRunRampUpDuration   *prometheus.HistogramVec
	taskRunRampUpDuration    *prometheus.HistogramVec
	taskRunPodRampUpDuration *prometheus.HistogramVec

	initialized = false
)

// InitPrometheus initializes the prometheus stuff
func InitPrometheus(config *config.Config) {
	if initialized {
		return
	}

	initialized = true

	buildRunEstablishDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "build_buildrun_establish_duration_seconds",
			Help:    "BuildRun establish duration in seconds.",
			Buckets: config.Prometheus.BuildRunEstablishDurationBuckets,
		},
		[]string{buildStrategyLabel, namespaceLabel})

	buildRunCompletionDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "build_buildrun_completion_duration_seconds",
			Help:    "BuildRun completion duration in seconds.",
			Buckets: config.Prometheus.BuildRunCompletionDurationBuckets,
		},
		[]string{buildStrategyLabel, namespaceLabel})

	buildRunRampUpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "build_buildrun_rampup_duration_seconds",
			Help:    "BuildRun ramp-up duration in seconds (time between buildrun creation and taskrun creation).",
			Buckets: config.Prometheus.BuildRunRampUpDurationBuckets,
		},
		[]string{buildStrategyLabel, namespaceLabel})

	taskRunRampUpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "build_buildrun_taskrun_rampup_duration_seconds",
			Help:    "BuildRun taskrun ramp-up duration in seconds (time between taskrun creation and taskrun pod creation).",
			Buckets: config.Prometheus.BuildRunRampUpDurationBuckets,
		},
		[]string{buildStrategyLabel, namespaceLabel})

	taskRunPodRampUpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "build_buildrun_taskrun_pod_rampup_duration_seconds",
			Help:    "BuildRun taskrun pod ramp-up duration in seconds (time between pod creation and last init container completion).",
			Buckets: config.Prometheus.BuildRunRampUpDurationBuckets,
		},
		[]string{buildStrategyLabel, namespaceLabel})

	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(
		buildCount,
		buildRunCount,
		buildRunEstablishDuration,
		buildRunCompletionDuration)
}

// BuildCountInc increases a number of the existing build total count
func BuildCountInc(buildStrategy string) {
	buildCount.WithLabelValues(buildStrategy).Inc()
}

// BuildRunCountInc increases a number of the existing build run total count
func BuildRunCountInc(buildStrategy string) {
	buildRunCount.WithLabelValues(buildStrategy).Inc()
}

// BuildRunEstablishObserve sets the build run establish time
func BuildRunEstablishObserve(buildStrategy, namespace string, duration time.Duration) {
	buildRunEstablishDuration.WithLabelValues(buildStrategy, namespace).Observe(duration.Seconds())
}

// BuildRunCompletionObserve sets the build run completion time
func BuildRunCompletionObserve(buildStrategy, namespace string, duration time.Duration) {
	buildRunCompletionDuration.WithLabelValues(buildStrategy, namespace).Observe(duration.Seconds())
}

// BuildRunRampUpDurationObserve processes the observation of a new buildrun ramp-up duration
func BuildRunRampUpDurationObserve(buildStrategy string, namespace string, duration time.Duration) {
	buildRunCompletionDuration.WithLabelValues(buildStrategy, namespace).Observe(duration.Seconds())
}

// TaskRunRampUpDurationObserve processes the observation of a new taskrun ramp-up duration
func TaskRunRampUpDurationObserve(buildStrategy string, namespace string, duration time.Duration) {
	buildRunCompletionDuration.WithLabelValues(buildStrategy, namespace).Observe(duration.Seconds())
}

// TaskRunPodRampUpDurationObserve processes the observation of a new taskrun pod ramp-up duration
func TaskRunPodRampUpDurationObserve(buildStrategy string, namespace string, duration time.Duration) {
	buildRunCompletionDuration.WithLabelValues(buildStrategy, namespace).Observe(duration.Seconds())
}
