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
		buildStrategy string
		namespace     string
		buildRun      string
	}

	var (
		counterMetrics   = map[string]map[string]float64{}
		histogramMetrics = map[string]map[buildRunLabels]float64{}

		promLabelPairToBuildRunLabels = func(in []*io_prometheus_client.LabelPair) buildRunLabels {
			var result = buildRunLabels{}
			for _, label := range in {
				switch *label.Name {
				case BuildStrategyLabel:
					result.buildStrategy = *label.Value
				case NamespaceLabel:
					result.namespace = *label.Value
				case BuildRunLabel:
					result.buildRun = *label.Value
				}
			}

			return result
		}
	)

	BeforeSuite(func() {
		var (
			testLabels = []buildRunLabels{
				{buildStrategy: "kaniko", namespace: "default", buildRun: "kaniko-buildrun"},
				{buildStrategy: "buildpacks", namespace: "default", buildRun: "buildpacks-buildrun"},
			}

			knownCounterMetrics = []string{
				"build_buildruns_completed_total",
				"build_builds_registered_total",
			}

			knownHistogramMetrics = []string{
				"build_buildrun_establish_duration_seconds",
				"build_buildrun_completion_duration_seconds",
				"build_buildrun_rampup_duration_seconds",
				"build_buildrun_taskrun_rampup_duration_seconds",
				"build_buildrun_taskrun_pod_rampup_duration_seconds",
			}
		)

		// initialize the counter metrics result map with empty maps
		for _, name := range knownCounterMetrics {
			counterMetrics[name] = map[string]float64{}
		}

		// initialize the histogram metrics result map with empty maps
		for _, name := range knownHistogramMetrics {
			histogramMetrics[name] = map[buildRunLabels]float64{}
		}

		// initialize prometheus (second init should be no-op)
		config := config.NewDefaultConfig()
		config.Prometheus.HistogramEnabledLabels = []string{BuildStrategyLabel, NamespaceLabel, BuildRunLabel}
		InitPrometheus(config)
		InitPrometheus(config)

		// and fire some examples
		for _, entry := range testLabels {
			buildStrategy, namespace, buildRun := entry.buildStrategy, entry.namespace, entry.buildRun

			// tell prometheus some things have happened
			BuildCountInc(buildStrategy)
			BuildRunCountInc(buildStrategy)
			BuildRunEstablishObserve(buildStrategy, namespace, buildRun, time.Duration(1)*time.Second)
			BuildRunCompletionObserve(buildStrategy, namespace, buildRun, time.Duration(200)*time.Second)
			BuildRunRampUpDurationObserve(buildStrategy, namespace, buildRun, time.Duration(1)*time.Second)
			TaskRunRampUpDurationObserve(buildStrategy, namespace, buildRun, time.Duration(2)*time.Second)
			TaskRunPodRampUpDurationObserve(buildStrategy, namespace, buildRun, time.Duration(3)*time.Second)
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
			Expect(histogramMetrics["build_buildrun_establish_duration_seconds"][buildRunLabels{"kaniko", "default", "kaniko-buildrun"}]).To(Equal(1.0))
		})

		It("should record the kaniko buildrun completion time", func() {
			Expect(histogramMetrics).To(HaveKey("build_buildrun_completion_duration_seconds"))
			Expect(histogramMetrics["build_buildrun_completion_duration_seconds"][buildRunLabels{"kaniko", "default", "kaniko-buildrun"}]).To(Equal(200.0))
		})

		It("should record the kaniko ramp-up durations", func() {
			Expect(histogramMetrics).To(HaveKey("build_buildrun_rampup_duration_seconds"))
			Expect(histogramMetrics).To(HaveKey("build_buildrun_taskrun_rampup_duration_seconds"))
			Expect(histogramMetrics).To(HaveKey("build_buildrun_taskrun_pod_rampup_duration_seconds"))

			Expect(histogramMetrics["build_buildrun_rampup_duration_seconds"][buildRunLabels{"kaniko", "default", "kaniko-buildrun"}]).To(BeNumerically(">", 0.0))
			Expect(histogramMetrics["build_buildrun_taskrun_rampup_duration_seconds"][buildRunLabels{"kaniko", "default", "kaniko-buildrun"}]).To(BeNumerically(">", 0.0))
			Expect(histogramMetrics["build_buildrun_taskrun_pod_rampup_duration_seconds"][buildRunLabels{"kaniko", "default", "kaniko-buildrun"}]).To(BeNumerically(">", 0.0))
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
			Expect(histogramMetrics["build_buildrun_establish_duration_seconds"][buildRunLabels{"buildpacks", "default", "buildpacks-buildrun"}]).To(Equal(1.0))
		})

		It("should record the buildpacks buildrun completion time", func() {
			Expect(histogramMetrics).To(HaveKey("build_buildrun_completion_duration_seconds"))
			Expect(histogramMetrics["build_buildrun_completion_duration_seconds"][buildRunLabels{"buildpacks", "default", "buildpacks-buildrun"}]).To(Equal(200.0))
		})

		It("should record the buildpacks ramp-up durations", func() {
			Expect(histogramMetrics).To(HaveKey("build_buildrun_rampup_duration_seconds"))
			Expect(histogramMetrics).To(HaveKey("build_buildrun_taskrun_rampup_duration_seconds"))
			Expect(histogramMetrics).To(HaveKey("build_buildrun_taskrun_pod_rampup_duration_seconds"))

			Expect(histogramMetrics["build_buildrun_rampup_duration_seconds"][buildRunLabels{"buildpacks", "default", "buildpacks-buildrun"}]).To(BeNumerically(">", 0.0))
			Expect(histogramMetrics["build_buildrun_taskrun_rampup_duration_seconds"][buildRunLabels{"buildpacks", "default", "buildpacks-buildrun"}]).To(BeNumerically(">", 0.0))
			Expect(histogramMetrics["build_buildrun_taskrun_pod_rampup_duration_seconds"][buildRunLabels{"buildpacks", "default", "buildpacks-buildrun"}]).To(BeNumerically(">", 0.0))
		})
	})
})
