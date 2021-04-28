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

	kanikoDefaultImage = "gcr.io/kaniko-project/executor:v1.6.0"
	kanikoImageEnvVar  = "KANIKO_CONTAINER_IMAGE"

	remoteArtifactsDefaultImage = "quay.io/quay/busybox:latest"
	remoteArtifactsEnvVar       = "REMOTE_ARTIFACTS_CONTAINER_IMAGE"

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

	// environment variables for the controllers
	controllerBuildMaxConcurrentReconciles                = "BUILD_MAX_CONCURRENT_RECONCILES"
	controllerBuildRunMaxConcurrentReconciles             = "BUILDRUN_MAX_CONCURRENT_RECONCILES"
	controllerBuildStrategyMaxConcurrentReconciles        = "BUILDSTRATEGY_MAX_CONCURRENT_RECONCILES"
	controllerClusterBuildStrategyMaxConcurrentReconciles = "CLUSTERBUILDSTRATEGY_MAX_CONCURRENT_RECONCILES"

	// environment variables for the kube API
	kubeAPIBurst = "KUBE_API_BURST"
	kubeAPIQPS   = "KUBE_API_QPS"
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
	CtxTimeOut                    time.Duration
	KanikoContainerImage          string
	RemoteArtifactsContainerImage string
	Prometheus                    PrometheusConfig
	ManagerOptions                ManagerOptions
	Controllers                   Controllers
	KubeAPIOptions                KubeAPIOptions
}

// PrometheusConfig contains the specific configuration for the
type PrometheusConfig struct {
	BuildRunCompletionDurationBuckets []float64
	BuildRunEstablishDurationBuckets  []float64
	BuildRunRampUpDurationBuckets     []float64
	EnabledLabels                     []string
}

// ManagerOptions contains configurable options for the Shipwright build controller manager
type ManagerOptions struct {
	LeaderElectionNamespace string
	LeaseDuration           *time.Duration
	RenewDeadline           *time.Duration
	RetryPeriod             *time.Duration
}

// Controllers contains the options for the different controllers
type Controllers struct {
	Build                ControllerOptions
	BuildRun             ControllerOptions
	BuildStrategy        ControllerOptions
	ClusterBuildStrategy ControllerOptions
}

// ControllerOptions contains configurable options for a controller
type ControllerOptions struct {
	MaxConcurrentReconciles int
}

// KubeAPIOptions contains configrable options for the kube API client
type KubeAPIOptions struct {
	QPS   int
	Burst int
}

// NewDefaultConfig returns a new Config, with context timeout and default Kaniko image.
func NewDefaultConfig() *Config {
	return &Config{
		CtxTimeOut:                    contextTimeout,
		KanikoContainerImage:          kanikoDefaultImage,
		RemoteArtifactsContainerImage: remoteArtifactsDefaultImage,
		Prometheus: PrometheusConfig{
			BuildRunCompletionDurationBuckets: metricBuildRunCompletionDurationBuckets,
			BuildRunEstablishDurationBuckets:  metricBuildRunEstablishDurationBuckets,
			BuildRunRampUpDurationBuckets:     metricBuildRunRampUpDurationBuckets,
		},
		ManagerOptions: ManagerOptions{
			LeaderElectionNamespace: leaderElectionNamespaceDefault,
		},
		Controllers: Controllers{
			Build: ControllerOptions{
				MaxConcurrentReconciles: 0,
			},
			BuildRun: ControllerOptions{
				MaxConcurrentReconciles: 0,
			},
			BuildStrategy: ControllerOptions{
				MaxConcurrentReconciles: 0,
			},
			ClusterBuildStrategy: ControllerOptions{
				MaxConcurrentReconciles: 0,
			},
		},
		KubeAPIOptions: KubeAPIOptions{
			QPS:   0,
			Burst: 0,
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

	if remoteArtifactsImage := os.Getenv(remoteArtifactsEnvVar); remoteArtifactsImage != "" {
		c.RemoteArtifactsContainerImage = remoteArtifactsImage
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

	if err := updateBuildControllerDurationOption(&c.ManagerOptions.LeaseDuration, leaseDurationEnvVar); err != nil {
		return err
	}

	if err := updateBuildControllerDurationOption(&c.ManagerOptions.RenewDeadline, renewDeadlineEnvVar); err != nil {
		return err
	}

	if err := updateBuildControllerDurationOption(&c.ManagerOptions.RetryPeriod, retryPeriodEnvVar); err != nil {
		return err
	}

	// controller settings
	if err := updateIntOption(&c.Controllers.Build.MaxConcurrentReconciles, controllerBuildMaxConcurrentReconciles); err != nil {
		return err
	}
	if err := updateIntOption(&c.Controllers.BuildRun.MaxConcurrentReconciles, controllerBuildRunMaxConcurrentReconciles); err != nil {
		return err
	}
	if err := updateIntOption(&c.Controllers.BuildStrategy.MaxConcurrentReconciles, controllerBuildStrategyMaxConcurrentReconciles); err != nil {
		return err
	}
	if err := updateIntOption(&c.Controllers.ClusterBuildStrategy.MaxConcurrentReconciles, controllerClusterBuildStrategyMaxConcurrentReconciles); err != nil {
		return err
	}

	// kube API settings
	if err := updateIntOption(&c.KubeAPIOptions.Burst, kubeAPIBurst); err != nil {
		return err
	}
	if err := updateIntOption(&c.KubeAPIOptions.QPS, kubeAPIQPS); err != nil {
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

func updateBuildControllerDurationOption(d **time.Duration, envVarName string) error {
	if value := os.Getenv(envVarName); value != "" {
		valueDuration, err := time.ParseDuration(value)
		if err != nil {
			return err
		}

		*d = &valueDuration
	}

	return nil
}

func updateIntOption(i *int, envVarName string) error {
	if value := os.Getenv(envVarName); value != "" {
		intValue, err := strconv.ParseInt(value, 10, 0)
		if err != nil {
			return err
		}
		*i = int(intValue)
	}

	return nil
}
