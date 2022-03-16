// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/pointer"

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

		It("should allow for an override of the Shipwright build controller leader election namespace using an environment variable", func() {
			var overrides = map[string]string{"BUILD_CONTROLLER_LEADER_ELECTION_NAMESPACE": "shipwright-build"}
			configWithEnvVariableOverrides(overrides, func(config *Config) {
				Expect(config.ManagerOptions.LeaderElectionNamespace).To(Equal("shipwright-build"))
			})
		})

		It("should allow for an override of the Shipwright build controller leader election times using environment variables", func() {
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

		It("should allow for an override of concurrent reconciles of the controllers", func() {
			var overrides = map[string]string{
				"BUILD_MAX_CONCURRENT_RECONCILES":                "2",
				"BUILDRUN_MAX_CONCURRENT_RECONCILES":             "3",
				"BUILDSTRATEGY_MAX_CONCURRENT_RECONCILES":        "4",
				"CLUSTERBUILDSTRATEGY_MAX_CONCURRENT_RECONCILES": "5",
			}

			configWithEnvVariableOverrides(overrides, func(config *Config) {
				Expect(config.Controllers.Build.MaxConcurrentReconciles).To(Equal(2))
				Expect(config.Controllers.BuildRun.MaxConcurrentReconciles).To(Equal(3))
				Expect(config.Controllers.BuildStrategy.MaxConcurrentReconciles).To(Equal(4))
				Expect(config.Controllers.ClusterBuildStrategy.MaxConcurrentReconciles).To(Equal(5))
			})
		})

		It("should allow for an override of kube API client configuration", func() {
			var overrides = map[string]string{
				"KUBE_API_BURST": "200",
				"KUBE_API_QPS":   "300",
			}

			configWithEnvVariableOverrides(overrides, func(config *Config) {
				Expect(config.KubeAPIOptions.Burst).To(Equal(200))
				Expect(config.KubeAPIOptions.QPS).To(Equal(300))
			})
		})

		It("should allow for an override of the Git container template", func() {
			var overrides = map[string]string{
				"GIT_CONTAINER_TEMPLATE": "{\"image\":\"myregistry/custom/git-image\",\"resources\":{\"requests\":{\"cpu\":\"0.5\",\"memory\":\"128Mi\"}}}",
			}

			configWithEnvVariableOverrides(overrides, func(config *Config) {
				Expect(config.GitContainerTemplate).To(Equal(corev1.Container{
					Image: "myregistry/custom/git-image",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("0.5"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
					},
				}))
			})
		})

		It("should allow for an override of the Git container image", func() {
			var overrides = map[string]string{
				"GIT_CONTAINER_IMAGE": "myregistry/custom/git-image",
			}

			configWithEnvVariableOverrides(overrides, func(config *Config) {
				nonRoot := pointer.Int64(1000)
				Expect(config.GitContainerTemplate).To(Equal(corev1.Container{
					Image: "myregistry/custom/git-image",
					Command: []string{
						"/ko-app/git",
					},
					SecurityContext: &corev1.SecurityContext{
						RunAsUser:  nonRoot,
						RunAsGroup: nonRoot,
					},
				}))
			})
		})

		It("should allow for an override of the Git container template and image", func() {
			var overrides = map[string]string{
				"GIT_CONTAINER_TEMPLATE": `{"image":"myregistry/custom/git-image","resources":{"requests":{"cpu":"0.5","memory":"128Mi"}}}`,
				"GIT_CONTAINER_IMAGE":    "myregistry/custom/git-image:override",
			}

			configWithEnvVariableOverrides(overrides, func(config *Config) {
				Expect(config.GitContainerTemplate).To(Equal(corev1.Container{
					Image: "myregistry/custom/git-image:override",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("0.5"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
					},
				}))
			})
		})

		It("should allow for an override of the Mutate-Image container template", func() {
			overrides := map[string]string{
				"MUTATE_IMAGE_CONTAINER_TEMPLATE": `{"image":"myregistry/custom/mutate-image","resources":{"requests":{"cpu":"0.5","memory":"128Mi"}}}`,
			}

			configWithEnvVariableOverrides(overrides, func(config *Config) {
				Expect(config.MutateImageContainerTemplate).To(Equal(corev1.Container{
					Image: "myregistry/custom/mutate-image",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("0.5"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
					},
				}))
			})
		})

		It("should allow for an override of the Waiter container template", func() {
			var overrides = map[string]string{
				"WAITER_CONTAINER_TEMPLATE": `{"image":"myregistry/custom/image","resources":{"requests":{"cpu":"0.5","memory":"128Mi"}}}`,
			}

			configWithEnvVariableOverrides(overrides, func(config *Config) {
				Expect(config.WaiterContainerTemplate).To(Equal(corev1.Container{
					Image: "myregistry/custom/image",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("0.5"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
					},
				}))
			})
		})

		It("should allow for an override of the Mutate-Image container template and image", func() {
			overrides := map[string]string{
				"MUTATE_IMAGE_CONTAINER_TEMPLATE": `{"image":"myregistry/custom/mutate-image","resources":{"requests":{"cpu":"0.5","memory":"128Mi"}}}`,
				"MUTATE_IMAGE_CONTAINER_IMAGE":    "myregistry/custom/mutate-image:override",
			}

			configWithEnvVariableOverrides(overrides, func(config *Config) {
				Expect(config.MutateImageContainerTemplate).To(Equal(corev1.Container{
					Image: "myregistry/custom/mutate-image:override",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("0.5"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
					},
				}))
			})

		})

		It("should allow for an override of the Waiter container image", func() {
			var overrides = map[string]string{
				"WAITER_CONTAINER_IMAGE": "myregistry/custom/image",
			}

			configWithEnvVariableOverrides(overrides, func(config *Config) {
				nonRoot := pointer.Int64(1000)
				Expect(config.WaiterContainerTemplate).To(Equal(corev1.Container{
					Image:   "myregistry/custom/image",
					Command: []string{"/ko-app/waiter"},
					Args:    []string{"start"},
					SecurityContext: &corev1.SecurityContext{
						RunAsUser:  nonRoot,
						RunAsGroup: nonRoot,
					},
				}))
			})
		})

		It("should allow for an override of the Waiter container template and image", func() {
			var overrides = map[string]string{
				"WAITER_CONTAINER_TEMPLATE": `{"image":"myregistry/custom/image","resources":{"requests":{"cpu":"0.5","memory":"128Mi"}}}`,
				"WAITER_CONTAINER_IMAGE":    "myregistry/custom/image:override",
			}

			configWithEnvVariableOverrides(overrides, func(config *Config) {
				Expect(config.WaiterContainerTemplate).To(Equal(corev1.Container{
					Image: "myregistry/custom/image:override",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("0.5"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
					},
				}))
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
