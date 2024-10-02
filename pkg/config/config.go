// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

const (
	contextTimeout = 300 * time.Second
	// A number in seconds to define a context Timeout
	// E.g. if 5 seconds is wanted, the CTX_TIMEOUT=5
	contextTimeoutEnvVar = "CTX_TIMEOUT"

	remoteArtifactsDefaultImage = "quay.io/quay/busybox:latest"
	remoteArtifactsEnvVar       = "REMOTE_ARTIFACTS_CONTAINER_IMAGE"

	// the Git image is built using ko which can replace environment variable values in the deployment, so once we decide to move
	// from environment variables to a ConfigMap, then we should move the container template, but retain the environment variable
	// (or make it an argument like Tekton)
	gitDefaultImage            = "ghcr.io/shipwright-io/build/git:latest"
	gitImageEnvVar             = "GIT_CONTAINER_IMAGE"
	gitContainerTemplateEnvVar = "GIT_CONTAINER_TEMPLATE"

	imageProcessingDefaultImage            = "ghcr.io/shipwright-io/build/image-processing:latest"
	imageProcessingImageEnvVar             = "IMAGE_PROCESSING_CONTAINER_IMAGE"
	imageProcessingContainerTemplateEnvVar = "IMAGE_PROCESSING_CONTAINER_TEMPLATE"

	// Analog to the Git image, the bundle image is also created by ko
	bundleDefaultImage            = "ghcr.io/shipwright-io/build/bundle:latest"
	bundleImageEnvVar             = "BUNDLE_CONTAINER_IMAGE"
	bundleContainerTemplateEnvVar = "BUNDLE_CONTAINER_TEMPLATE"

	// environment variable to hold waiter's container image, created by ko
	waiterDefaultImage            = "ghcr.io/shipwright-io/build/waiter:latest"
	waiterImageEnvVar             = "WAITER_CONTAINER_IMAGE"
	waiterContainerTemplateEnvVar = "WAITER_CONTAINER_TEMPLATE"

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

	terminationLogPathDefault = "/dev/termination-log"
	terminationLogPathEnvVar  = "TERMINATION_LOG_PATH"

	// environment variable for the Git rewrite setting
	useGitRewriteRule = "GIT_ENABLE_REWRITE_RULE"

	// environment variable to hold vulnerability count limit
	VulnerabilityCountLimitEnvVar = "VULNERABILITY_COUNT_LIMIT"
)

var (
	// arrays are not possible as constants
	metricBuildRunCompletionDurationBuckets = prometheus.LinearBuckets(50, 50, 10)
	metricBuildRunEstablishDurationBuckets  = []float64{0, 1, 2, 3, 5, 7, 10, 15, 20, 30}
	metricBuildRunRampUpDurationBuckets     = prometheus.LinearBuckets(0, 1, 10)

	root    = ptr.To[int64](0)
	nonRoot = ptr.To[int64](1000)
)

// Config hosts different parameters that
// can be set to use on the Build controllers
type Config struct {
	CtxTimeOut                       time.Duration
	GitContainerTemplate             Step
	ImageProcessingContainerTemplate Step
	BundleContainerTemplate          Step
	WaiterContainerTemplate          Step
	RemoteArtifactsContainerImage    string
	TerminationLogPath               string
	Prometheus                       PrometheusConfig
	ManagerOptions                   ManagerOptions
	Controllers                      Controllers
	KubeAPIOptions                   KubeAPIOptions
	GitRewriteRule                   bool
	VulnerabilityCountLimit          int
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

// KubeAPIOptions contains configurable options for the kube API client
type KubeAPIOptions struct {
	QPS   int
	Burst int
}

type Step struct {
	Args            []string                    `json:"args,omitempty"`
	Command         []string                    `json:"command,omitempty"`
	Env             []corev1.EnvVar             `json:"env,omitempty"`
	Image           string                      `json:"image,omitempty"`
	ImagePullPolicy corev1.PullPolicy           `json:"imagePullPolicy,omitempty"`
	Resources       corev1.ResourceRequirements `json:"resources,omitempty"`
	SecurityContext *corev1.SecurityContext     `json:"securityContext,omitempty"`
	WorkingDir      string                      `json:"workingDir,omitempty"`
}

// NewDefaultConfig returns a new Config, with context timeout and default Kaniko image.
func NewDefaultConfig() *Config {
	return &Config{
		CtxTimeOut:                    contextTimeout,
		RemoteArtifactsContainerImage: remoteArtifactsDefaultImage,
		TerminationLogPath:            terminationLogPathDefault,
		GitRewriteRule:                false,
		VulnerabilityCountLimit:       50,

		GitContainerTemplate: Step{
			Image: gitDefaultImage,
			Command: []string{
				"/ko-app/git",
			},
			Env: []corev1.EnvVar{
				// This directory is created in the base image as writable for everybody
				{
					Name:  "HOME",
					Value: "/shared-home",
				},
			},
			SecurityContext: &corev1.SecurityContext{
				AllowPrivilegeEscalation: ptr.To(false),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{
						"ALL",
					},
				},
				RunAsUser:  nonRoot,
				RunAsGroup: nonRoot,
			},
		},

		BundleContainerTemplate: Step{
			Image: bundleDefaultImage,
			Command: []string{
				"/ko-app/bundle",
			},
			// This directory is created in the base image as writable for everybody
			Env: []corev1.EnvVar{
				{
					Name:  "HOME",
					Value: "/shared-home",
				},
			},
			SecurityContext: &corev1.SecurityContext{
				AllowPrivilegeEscalation: ptr.To(false),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{
						"ALL",
					},
				},
				RunAsUser:  nonRoot,
				RunAsGroup: nonRoot,
			},
		},

		ImageProcessingContainerTemplate: Step{
			Image: imageProcessingDefaultImage,
			Command: []string{
				"/ko-app/image-processing",
			},
			// This directory is created in the base image as writable for everybody
			Env: []corev1.EnvVar{
				{
					Name:  "HOME",
					Value: "/shared-home",
				},
			},
			// The image processing step runs after the build strategy steps where an arbitrary
			// user could have been used to write the result files for the image digest. The
			// image processing step will overwrite the image digest file. To be able to do this
			// in all possible scenarios, we run this step as root with DAC_OVERRIDE
			// capability.
			SecurityContext: &corev1.SecurityContext{
				AllowPrivilegeEscalation: ptr.To(false),
				RunAsUser:                root,
				RunAsGroup:               root,
				Capabilities: &corev1.Capabilities{
					Add: []corev1.Capability{
						"DAC_OVERRIDE",
					},
					Drop: []corev1.Capability{
						"ALL",
					},
				},
			},
		},

		WaiterContainerTemplate: Step{
			Image: waiterDefaultImage,
			Command: []string{
				"/ko-app/waiter",
			},
			Args: []string{
				"start",
			},
			// This directory is created in the base image as writable for everybody
			Env: []corev1.EnvVar{
				{
					Name:  "HOME",
					Value: "/shared-home",
				},
			},
			SecurityContext: &corev1.SecurityContext{
				AllowPrivilegeEscalation: ptr.To(false),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{
						"ALL",
					},
				},
				RunAsUser:  nonRoot,
				RunAsGroup: nonRoot,
			},
		},

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

	if gitContainerTemplate := os.Getenv(gitContainerTemplateEnvVar); gitContainerTemplate != "" {
		c.GitContainerTemplate = Step{}
		if err := json.Unmarshal([]byte(gitContainerTemplate), &c.GitContainerTemplate); err != nil {
			return err
		}
		if c.GitContainerTemplate.Image == "" {
			c.GitContainerTemplate.Image = gitDefaultImage
		}
	}

	// the dedicated environment variable for the image overwrites what is defined in the git container template
	if gitImage := os.Getenv(gitImageEnvVar); gitImage != "" {
		c.GitContainerTemplate.Image = gitImage
	}

	if imageProcessingContainerTemplate := os.Getenv(imageProcessingContainerTemplateEnvVar); imageProcessingContainerTemplate != "" {
		c.ImageProcessingContainerTemplate = Step{}
		if err := json.Unmarshal([]byte(imageProcessingContainerTemplate), &c.ImageProcessingContainerTemplate); err != nil {
			return err
		}
		if c.ImageProcessingContainerTemplate.Image == "" {
			c.ImageProcessingContainerTemplate.Image = imageProcessingDefaultImage
		}
	}

	// the dedicated environment variable for the image overwrites
	// what is defined in the image processing container template
	if imageProcessingImage := os.Getenv(imageProcessingImageEnvVar); imageProcessingImage != "" {
		c.ImageProcessingContainerTemplate.Image = imageProcessingImage
	}

	// set environment variable for vulnerability count limit
	if vcStr := os.Getenv(VulnerabilityCountLimitEnvVar); vcStr != "" {
		vc, err := strconv.Atoi(vcStr)
		if err != nil {
			return err
		}
		c.VulnerabilityCountLimit = vc
	}

	// Mark that the Git wrapper is suppose to use Git rewrite rule
	if useGitRewriteRule := os.Getenv(useGitRewriteRule); useGitRewriteRule != "" {
		c.GitRewriteRule = strings.ToLower(useGitRewriteRule) == "true"
	}

	if bundleContainerTemplate := os.Getenv(bundleContainerTemplateEnvVar); bundleContainerTemplate != "" {
		c.BundleContainerTemplate = Step{}
		if err := json.Unmarshal([]byte(bundleContainerTemplate), &c.BundleContainerTemplate); err != nil {
			return err
		}
		if c.BundleContainerTemplate.Image == "" {
			c.BundleContainerTemplate.Image = bundleDefaultImage
		}
	}

	// the dedicated environment variable for the image overwrites what is defined in the bundle container template
	if bundleImage := os.Getenv(bundleImageEnvVar); bundleImage != "" {
		c.BundleContainerTemplate.Image = bundleImage
	}

	if waiterContainerTemplate := os.Getenv(waiterContainerTemplateEnvVar); waiterContainerTemplate != "" {
		c.WaiterContainerTemplate = Step{}
		if err := json.Unmarshal([]byte(waiterContainerTemplate), &c.WaiterContainerTemplate); err != nil {
			return err
		}
		if c.WaiterContainerTemplate.Image == "" {
			c.WaiterContainerTemplate.Image = waiterDefaultImage
		}
	}

	if waiterImage := os.Getenv(waiterImageEnvVar); waiterImage != "" {
		c.WaiterContainerTemplate.Image = waiterImage
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

	if terminationLogPath := os.Getenv(terminationLogPathEnvVar); terminationLogPath != "" {
		c.TerminationLogPath = terminationLogPath
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
