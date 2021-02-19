// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	contextTimeout = 300 * time.Second
	// A number in seconds to define a context Timeout
	// E.g. if 5 seconds is wanted, the CTX_TIMEOUT=5
	contextTimeoutEnvVar = "CTX_TIMEOUT"

	kanikoDefaultImage = "gcr.io/kaniko-project/executor:v1.5.1"
	kanikoImageEnvVar  = "KANIKO_CONTAINER_IMAGE"

	// environment variable to override the buckets
	metricBuildRunCompletionDurationBucketsEnvVar = "PROMETHEUS_BR_COMP_DUR_BUCKETS"
	metricBuildRunEstablishDurationBucketsEnvVar  = "PROMETHEUS_BR_EST_DUR_BUCKETS"
	metricBuildRunRampUpDurationBucketsEnvVar     = "PROMETHEUS_BR_RAMPUP_DUR_BUCKETS"

	// environment variable to enable prometheus metric labels
	prometheusEnabledLabelsEnvVar = "PROMETHEUS_ENABLED_LABELS"

	leaderElectionNamespaceDefault = "default"
	leaderElectionNamespaceEnvVar  = "BUILD_CONTROLLER_LEADER_ELECTION_NAMESPACE"

	leaseDurationEnvVar = "BUILD_CONTROLLER_LEASE_DURATION"
	renewDeadlineEnvVar = "BUILD_CONTROLLER_RENEW_DEADLINE"
	retryPeriodEnvVar   = "BUILD_CONTROLLER_RETRY_PERIOD"
)

var (
	// arrays are not possible as constants
	metricBuildRunCompletionDurationBuckets = prometheus.LinearBuckets(50, 50, 10)
	metricBuildRunEstablishDurationBuckets  = []float64{0, 1, 2, 3, 5, 7, 10, 15, 20, 30}
	metricBuildRunRampUpDurationBuckets     = prometheus.LinearBuckets(0, 1, 10)
)

// Config hosts different parameters that
// can be set to use on the Build controllers
type Config struct {
	CtxTimeOut           time.Duration
	KanikoContainerImage string
	Prometheus           PrometheusConfig
	ManagerOptions       ManagerOptions
}

// PrometheusConfig contains the specific configuration for the
type PrometheusConfig struct {
	BuildRunCompletionDurationBuckets []float64
	BuildRunEstablishDurationBuckets  []float64
	BuildRunRampUpDurationBuckets     []float64
	EnabledLabels                     []string
}

// ManagerOptions contains configurable options for the build operator manager
type ManagerOptions struct {
	LeaderElectionNamespace string
	LeaseDuration           *time.Duration
	RenewDeadline           *time.Duration
	RetryPeriod             *time.Duration
}

// NewDefaultConfig returns a new Config, with context timeout and default Kaniko image.
func NewDefaultConfig() *Config {
	return &Config{
		CtxTimeOut:           contextTimeout,
		KanikoContainerImage: kanikoDefaultImage,
		Prometheus: PrometheusConfig{
			BuildRunCompletionDurationBuckets: metricBuildRunCompletionDurationBuckets,
			BuildRunEstablishDurationBuckets:  metricBuildRunEstablishDurationBuckets,
			BuildRunRampUpDurationBuckets:     metricBuildRunRampUpDurationBuckets,
		},
		ManagerOptions: ManagerOptions{
			LeaderElectionNamespace: leaderElectionNamespaceDefault,
		},
	}
}

// SetConfigFromEnv updates the configuration managed by environment variables.
func (c *Config) SetConfigFromEnv() error {
	if timeout := os.Getenv(contextTimeoutEnvVar); timeout != "" {
		i, err := strconv.Atoi(timeout)
		if err != nil {
			return err
		}
		c.CtxTimeOut = time.Duration(i) * time.Second
	}

	if kanikoImage := os.Getenv(kanikoImageEnvVar); kanikoImage != "" {
		c.KanikoContainerImage = kanikoImage
	}

	if err := updateBucketsConfig(&c.Prometheus.BuildRunCompletionDurationBuckets, metricBuildRunCompletionDurationBucketsEnvVar); err != nil {
		return err
	}

	if err := updateBucketsConfig(&c.Prometheus.BuildRunEstablishDurationBuckets, metricBuildRunEstablishDurationBucketsEnvVar); err != nil {
		return err
	}

	if err := updateBucketsConfig(&c.Prometheus.BuildRunRampUpDurationBuckets, metricBuildRunRampUpDurationBucketsEnvVar); err != nil {
		return err
	}

	c.Prometheus.EnabledLabels = strings.Split(os.Getenv(prometheusEnabledLabelsEnvVar), ",")

	if leaderElectionNamespace := os.Getenv(leaderElectionNamespaceEnvVar); leaderElectionNamespace != "" {
		c.ManagerOptions.LeaderElectionNamespace = leaderElectionNamespace
	}

	if err := updateBuildOperatorDurationOption(&c.ManagerOptions.LeaseDuration, leaseDurationEnvVar); err != nil {
		return err
	}

	if err := updateBuildOperatorDurationOption(&c.ManagerOptions.RenewDeadline, renewDeadlineEnvVar); err != nil {
		return err
	}

	if err := updateBuildOperatorDurationOption(&c.ManagerOptions.RetryPeriod, retryPeriodEnvVar); err != nil {
		return err
	}

	return nil
}

func stringToFloat64Array(strings []string) ([]float64, error) {
	floats := make([]float64, len(strings))

	for i, string := range strings {
		float, err := strconv.ParseFloat(string, 64)
		if err != nil {
			return nil, err
		}
		floats[i] = float
	}

	return floats, nil
}

func updateBucketsConfig(buckets *[]float64, envVarName string) error {
	if values, found := os.LookupEnv(envVarName); found {
		floats, err := stringToFloat64Array(strings.Split(values, ","))
		if err != nil {
			return err
		}

		*buckets = floats
	}

	return nil
}

func updateBuildOperatorDurationOption(d **time.Duration, envVarName string) error {
	if value := os.Getenv(envVarName); value != "" {
		valueDuration, err := time.ParseDuration(value)
		if err != nil {
			return err
		}

		*d = &valueDuration
	}

	return nil
}
