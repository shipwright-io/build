// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package metrics_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/shipwright-io/build/pkg/metrics"

	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/shipwright-io/build/pkg/config"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var _ = Describe("Custom Metrics", func() {
	type buildRunLabels struct {
		namespace     string
		buildStrategy string
	}

	var (
		counterMetrics   = map[string]map[string]float64{}
		histogramMetrics = map[string]map[buildRunLabels]float64{}

		promLabelPairToBuildRunLabels = func(in []*io_prometheus_client.LabelPair) buildRunLabels {
			var result = buildRunLabels{}
			for _, label := range in {
				switch *label.Name {
				case "buildstrategy":
					result.buildStrategy = *label.Value

				case "namespace":
					result.namespace = *label.Value
				}
			}

			return result
		}
	)

	BeforeSuite(func() {
		var (
			testLabels = []buildRunLabels{
				{namespace: "default", buildStrategy: "kaniko"},
				{namespace: "default", buildStrategy: "buildpacks"},
			}

			knownCounterMetrics = []string{
				"build_buildruns_completed_total",
				"build_builds_registered_total",
			}

			knownHistrogramMetrics = []string{
				"build_buildrun_establish_duration_seconds",
				"build_buildrun_completion_duration_seconds",
				"build_buildrun_rampup_duration_seconds",
				"build_buildrun_taskrun_rampup_duration_seconds",
				"build_buildrun_taskrun_pod_rampup_duration_seconds",
			}
		)

		// initialise the counter metrics result map with empty maps
		for _, name := range knownCounterMetrics {
			counterMetrics[name] = map[string]float64{}
		}

		// initialise the histrogram metrics result map with empty maps
		for _, name := range knownHistrogramMetrics {
			histogramMetrics[name] = map[buildRunLabels]float64{}
		}

		// initialise prometheus (second init should be no-op)
		config := config.NewDefaultConfig()
		config.Prometheus.HistogramEnabledLabels = []string{"buildstrategy", "namespace"}
		InitPrometheus(config)
		InitPrometheus(config)

		// and fire some examples
		for _, entry := range testLabels {
			buildStrategy, namespace := entry.buildStrategy, entry.namespace

			// tell prometheus some things have happened
			BuildCountInc(buildStrategy)
			BuildRunCountInc(buildStrategy)
			BuildRunEstablishObserve(buildStrategy, namespace, time.Duration(1)*time.Second)
			BuildRunCompletionObserve(buildStrategy, namespace, time.Duration(200)*time.Second)
			BuildRunRampUpDurationObserve(buildStrategy, namespace, time.Duration(1)*time.Second)
			TaskRunRampUpDurationObserve(buildStrategy, namespace, time.Duration(2)*time.Second)
			TaskRunPodRampUpDurationObserve(buildStrategy, namespace, time.Duration(3)*time.Second)
		}

		// gather metrics from prometheus and fill the result maps
		metrics, err := crmetrics.Registry.Gather()
		if err != nil {
			Fail(err.Error())
		}

		for _, metricFamily := range metrics {
			switch metricFamily.GetType() {
			case io_prometheus_client.MetricType_COUNTER:
				for _, metric := range metricFamily.GetMetric() {
					for _, label := range metric.GetLabel() {
						counterMetrics[metricFamily.GetName()][*label.Value] = metric.GetCounter().GetValue()
					}
				}

			case io_prometheus_client.MetricType_HISTOGRAM:
				for _, metric := range metricFamily.GetMetric() {
					histogramMetrics[metricFamily.GetName()][promLabelPairToBuildRunLabels(metric.GetLabel())] = metric.GetHistogram().GetSampleSum()
				}
			}
		}
	})

	Context("when create a new kaniko buildrun", func() {

		It("should increase the kaniko build count", func() {
			Expect(counterMetrics).To(HaveKey("build_builds_registered_total"))
			Expect(counterMetrics["build_builds_registered_total"]["kaniko"]).To(Equal(1.0))
		})

		It("should increase the kaniko buildrun count", func() {
			Expect(counterMetrics).To(HaveKey("build_buildruns_completed_total"))
			Expect(counterMetrics["build_buildruns_completed_total"]["kaniko"]).To(Equal(1.0))
		})

		It("should record the kaniko buildrun establish time", func() {
			Expect(histogramMetrics).To(HaveKey("build_buildrun_establish_duration_seconds"))
			Expect(histogramMetrics["build_buildrun_establish_duration_seconds"][buildRunLabels{"default", "kaniko"}]).To(Equal(1.0))
		})

		It("should record the kaniko buildrun completion time", func() {
			Expect(histogramMetrics).To(HaveKey("build_buildrun_completion_duration_seconds"))
			Expect(histogramMetrics["build_buildrun_completion_duration_seconds"][buildRunLabels{"default", "kaniko"}]).To(Equal(200.0))
		})

		It("should record the kaniko ramp-up durations", func() {
			Expect(histogramMetrics).To(HaveKey("build_buildrun_rampup_duration_seconds"))
			Expect(histogramMetrics).To(HaveKey("build_buildrun_taskrun_rampup_duration_seconds"))
			Expect(histogramMetrics).To(HaveKey("build_buildrun_taskrun_pod_rampup_duration_seconds"))

			Expect(histogramMetrics["build_buildrun_rampup_duration_seconds"][buildRunLabels{"default", "kaniko"}]).To(BeNumerically(">", 0.0))
			Expect(histogramMetrics["build_buildrun_taskrun_rampup_duration_seconds"][buildRunLabels{"default", "kaniko"}]).To(BeNumerically(">", 0.0))
			Expect(histogramMetrics["build_buildrun_taskrun_pod_rampup_duration_seconds"][buildRunLabels{"default", "kaniko"}]).To(BeNumerically(">", 0.0))
		})
	})

	Context("when create a new buildpacks buildrun", func() {

		It("should increase the buildpacks build count", func() {
			Expect(counterMetrics).To(HaveKey("build_builds_registered_total"))
			Expect(counterMetrics["build_builds_registered_total"]["buildpacks"]).To(Equal(1.0))
		})

		It("should increase the buildpacks buildrun count", func() {
			Expect(counterMetrics).To(HaveKey("build_buildruns_completed_total"))
			Expect(counterMetrics["build_buildruns_completed_total"]["buildpacks"]).To(Equal(1.0))
		})

		It("should record the buildpacks buildrun establish time", func() {
			Expect(histogramMetrics).To(HaveKey("build_buildrun_establish_duration_seconds"))
			Expect(histogramMetrics["build_buildrun_establish_duration_seconds"][buildRunLabels{"default", "buildpacks"}]).To(Equal(1.0))
		})

		It("should record the buildpacks buildrun completion time", func() {
			Expect(histogramMetrics).To(HaveKey("build_buildrun_completion_duration_seconds"))
			Expect(histogramMetrics["build_buildrun_completion_duration_seconds"][buildRunLabels{"default", "buildpacks"}]).To(Equal(200.0))
		})

		It("should record the buildpacks ramp-up durations", func() {
			Expect(histogramMetrics).To(HaveKey("build_buildrun_rampup_duration_seconds"))
			Expect(histogramMetrics).To(HaveKey("build_buildrun_taskrun_rampup_duration_seconds"))
			Expect(histogramMetrics).To(HaveKey("build_buildrun_taskrun_pod_rampup_duration_seconds"))

			Expect(histogramMetrics["build_buildrun_rampup_duration_seconds"][buildRunLabels{"default", "buildpacks"}]).To(BeNumerically(">", 0.0))
			Expect(histogramMetrics["build_buildrun_taskrun_rampup_duration_seconds"][buildRunLabels{"default", "buildpacks"}]).To(BeNumerically(">", 0.0))
			Expect(histogramMetrics["build_buildrun_taskrun_pod_rampup_duration_seconds"][buildRunLabels{"default", "buildpacks"}]).To(BeNumerically(">", 0.0))
		})
	})
})
