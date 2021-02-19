// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/shipwright-io/build/pkg/config"
)

var _ = Describe("Config", func() {
	Context("obtaining the configuration for build", func() {
		It("should create a default configuration with reasonable values", func() {
			config := NewDefaultConfig()
			Expect(config).ToNot(BeNil())
		})

		It("should allow for an override of the context timeout using an environment variable", func() {
			var overrides = map[string]string{"CTX_TIMEOUT": "600"}
			configWithEnvVariableOverrides(overrides, func(config *Config) {
				Expect(config.CtxTimeOut).To(Equal(600 * time.Second))
			})
		})

		It("should allow for an override of the default Kaniko project image using an environment variable", func() {
			var overrides = map[string]string{"KANIKO_CONTAINER_IMAGE": "gcr.io/kaniko-project/executor:v1.0.1"}
			configWithEnvVariableOverrides(overrides, func(config *Config) {
				Expect(config.KanikoContainerImage).To(Equal("gcr.io/kaniko-project/executor:v1.0.1"))
			})
		})

		It("should allow for an override of the Prometheus buckets settings using an environment variable", func() {
			var overrides = map[string]string{
				"PROMETHEUS_BR_COMP_DUR_BUCKETS":   "1,2,3,4",
				"PROMETHEUS_BR_EST_DUR_BUCKETS":    "10,20,30,40",
				"PROMETHEUS_BR_RAMPUP_DUR_BUCKETS": "1,2,3,5,8,12,20",
			}

			configWithEnvVariableOverrides(overrides, func(config *Config) {
				Expect(config.Prometheus.BuildRunCompletionDurationBuckets).To(Equal([]float64{1, 2, 3, 4}))
				Expect(config.Prometheus.BuildRunEstablishDurationBuckets).To(Equal([]float64{10, 20, 30, 40}))
				Expect(config.Prometheus.BuildRunRampUpDurationBuckets).To(Equal([]float64{1, 2, 3, 5, 8, 12, 20}))
			})
		})

		It("should allow for an override of the Prometheus enabled labels using an environment variable", func() {
			var overrides = map[string]string{"PROMETHEUS_ENABLED_LABELS": "namespace,strategy"}
			configWithEnvVariableOverrides(overrides, func(config *Config) {
				Expect(config.Prometheus.EnabledLabels).To(Equal([]string{"namespace", "strategy"}))
			})
		})

		It("should allow for an override of the operator leader election namespace using an environment variable", func() {
			var overrides = map[string]string{"BUILD_CONTROLLER_LEADER_ELECTION_NAMESPACE": "shipwright-build"}
			configWithEnvVariableOverrides(overrides, func(config *Config) {
				Expect(config.ManagerOptions.LeaderElectionNamespace).To(Equal("shipwright-build"))
			})
		})

		It("should allow for an override of the operator leader election times using environment variables", func() {
			var overrides = map[string]string{
				"BUILD_CONTROLLER_LEASE_DURATION": "42s",
				"BUILD_CONTROLLER_RENEW_DEADLINE": "32s",
				"BUILD_CONTROLLER_RETRY_PERIOD":   "10s",
			}

			configWithEnvVariableOverrides(overrides, func(config *Config) {
				Expect(*config.ManagerOptions.LeaseDuration).To(Equal(time.Duration(42 * time.Second)))
				Expect(*config.ManagerOptions.RenewDeadline).To(Equal(time.Duration(32 * time.Second)))
				Expect(*config.ManagerOptions.RetryPeriod).To(Equal(time.Duration(10 * time.Second)))
			})
		})
	})
})

func configWithEnvVariableOverrides(settings map[string]string, f func(config *Config)) {
	var backup = make(map[string]*string, len(settings))
	for k, v := range settings {
		value, ok := os.LookupEnv(k)
		switch ok {
		case true:
			backup[k] = &value

		case false:
			backup[k] = nil
		}

		os.Setenv(k, v)
	}

	config := NewDefaultConfig()
	Expect(config).ToNot(BeNil())

	err := config.SetConfigFromEnv()
	Expect(err).ToNot(HaveOccurred())

	f(config)

	for k, v := range backup {
		if v != nil {
			os.Setenv(k, *v)
		} else {
			os.Unsetenv(k)
		}
	}
}
