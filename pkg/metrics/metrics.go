package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redhat-developer/build/pkg/config"
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

	buildRunEstablishDuration *prometheus.HistogramVec

	buildRunCompletionDuration *prometheus.HistogramVec

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
