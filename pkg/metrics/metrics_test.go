// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package metrics_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/shipwright-io/build/pkg/metrics"

	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/shipwright-io/build/pkg/config"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

type buildLabels struct {
	buildStrategy string
	namespace     string
	build         string
}

type buildRunLabels struct {
	buildStrategy string
	namespace     string
	build         string
	buildRun      string
}

var (
	buildCounterMetrics      map[string]map[buildLabels]float64
	buildRunCounterMetrics   map[string]map[buildRunLabels]float64
	buildRunHistogramMetrics map[string]map[buildRunLabels]float64
)

func promLabelPairToBuildLabels(in []*io_prometheus_client.LabelPair) buildLabels {
	result := buildLabels{}
	for _, label := range in {
		switch *label.Name {
		case BuildStrategyLabel:
			result.buildStrategy = *label.Value
		case NamespaceLabel:
			result.namespace = *label.Value
		case BuildLabel:
			result.build = *label.Value
		}
	}

	return result
}

func promLabelPairToBuildRunLabels(in []*io_prometheus_client.LabelPair) buildRunLabels {
	result := buildRunLabels{}
	for _, label := range in {
		switch *label.Name {
		case BuildStrategyLabel:
			result.buildStrategy = *label.Value
		case NamespaceLabel:
			result.namespace = *label.Value
		case BuildLabel:
			result.build = *label.Value
		case BuildRunLabel:
			result.buildRun = *label.Value
		}
	}

	return result
}

var _ = BeforeSuite(func() {
	buildCounterMetrics = map[string]map[buildLabels]float64{}
	buildRunCounterMetrics = map[string]map[buildRunLabels]float64{}
	buildRunHistogramMetrics = map[string]map[buildRunLabels]float64{}

	var (
		testLabels = []buildRunLabels{
			{buildStrategy: "kaniko", namespace: "default", build: "kaniko-build", buildRun: "kaniko-buildrun"},
			{buildStrategy: "buildpacks", namespace: "default", build: "buildpacks-build", buildRun: "buildpacks-buildrun"},
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
	buildCounterMetrics["build_builds_registered_total"] = map[buildLabels]float64{}
	buildRunCounterMetrics["build_buildruns_completed_total"] = map[buildRunLabels]float64{}

	// initialize the histogram metrics result map with empty maps
	for _, name := range knownHistogramMetrics {
		buildRunHistogramMetrics[name] = map[buildRunLabels]float64{}
	}

	// initialize prometheus (second init should be no-op)
	config := config.NewDefaultConfig()
	config.Prometheus.EnabledLabels = []string{BuildStrategyLabel, NamespaceLabel, BuildLabel, BuildRunLabel}
	InitPrometheus(config)

	// and fire some examples
	for _, entry := range testLabels {
		buildStrategy, namespace, build, buildRun := entry.buildStrategy, entry.namespace, entry.build, entry.buildRun

		// tell prometheus some things have happened
		BuildCountInc(buildStrategy, namespace, build)
		BuildRunCountInc(buildStrategy, namespace, build, buildRun)
		BuildRunEstablishObserve(buildStrategy, namespace, build, buildRun, time.Duration(1)*time.Second)
		BuildRunCompletionObserve(buildStrategy, namespace, build, buildRun, time.Duration(200)*time.Second)
		BuildRunRampUpDurationObserve(buildStrategy, namespace, build, buildRun, time.Duration(1)*time.Second)
		TaskRunRampUpDurationObserve(buildStrategy, namespace, build, buildRun, time.Duration(2)*time.Second)
		TaskRunPodRampUpDurationObserve(buildStrategy, namespace, build, buildRun, time.Duration(3)*time.Second)
	}

	// gather metrics from prometheus and fill the result maps
	metrics, err := crmetrics.Registry.Gather()
	if err != nil {
		Fail(err.Error())
	}

	for _, metricFamily := range metrics {
		if metricFamily.GetType() == io_prometheus_client.MetricType_HISTOGRAM {
			for _, metric := range metricFamily.GetMetric() {
				buildRunHistogramMetrics[metricFamily.GetName()][promLabelPairToBuildRunLabels(metric.GetLabel())] = metric.GetHistogram().GetSampleSum()
			}
		} else {
			switch metricFamily.GetName() {
			case "build_builds_registered_total":
				for _, metric := range metricFamily.GetMetric() {
					buildCounterMetrics[metricFamily.GetName()][promLabelPairToBuildLabels(metric.GetLabel())] = metric.GetCounter().GetValue()
				}
			case "build_buildruns_completed_total":
				for _, metric := range metricFamily.GetMetric() {
					buildRunCounterMetrics[metricFamily.GetName()][promLabelPairToBuildRunLabels(metric.GetLabel())] = metric.GetCounter().GetValue()
				}
			}
		}
	}
})

var _ = Describe("Custom Metrics", func() {
	Context("when create a new kaniko buildrun", func() {
		It("should increase the kaniko build count", func() {
			Expect(buildCounterMetrics).To(HaveKey("build_builds_registered_total"))
			Expect(buildCounterMetrics["build_builds_registered_total"][buildLabels{"kaniko", "default", "kaniko-build"}]).To(Equal(1.0))
		})

		It("should increase the kaniko buildrun count", func() {
			Expect(buildRunCounterMetrics).To(HaveKey("build_buildruns_completed_total"))
			Expect(buildRunCounterMetrics["build_buildruns_completed_total"][buildRunLabels{"kaniko", "default", "kaniko-build", "kaniko-buildrun"}]).To(Equal(1.0))
		})

		It("should record the kaniko buildrun establish time", func() {
			Expect(buildRunHistogramMetrics).To(HaveKey("build_buildrun_establish_duration_seconds"))
			Expect(buildRunHistogramMetrics["build_buildrun_establish_duration_seconds"][buildRunLabels{"kaniko", "default", "kaniko-build", "kaniko-buildrun"}]).To(Equal(1.0))
		})

		It("should record the kaniko buildrun completion time", func() {
			Expect(buildRunHistogramMetrics).To(HaveKey("build_buildrun_completion_duration_seconds"))
			Expect(buildRunHistogramMetrics["build_buildrun_completion_duration_seconds"][buildRunLabels{"kaniko", "default", "kaniko-build", "kaniko-buildrun"}]).To(Equal(200.0))
		})

		It("should record the kaniko ramp-up durations", func() {
			Expect(buildRunHistogramMetrics).To(HaveKey("build_buildrun_rampup_duration_seconds"))
			Expect(buildRunHistogramMetrics).To(HaveKey("build_buildrun_taskrun_rampup_duration_seconds"))
			Expect(buildRunHistogramMetrics).To(HaveKey("build_buildrun_taskrun_pod_rampup_duration_seconds"))

			Expect(buildRunHistogramMetrics["build_buildrun_rampup_duration_seconds"][buildRunLabels{"kaniko", "default", "kaniko-build", "kaniko-buildrun"}]).To(BeNumerically(">", 0.0))
			Expect(buildRunHistogramMetrics["build_buildrun_taskrun_rampup_duration_seconds"][buildRunLabels{"kaniko", "default", "kaniko-build", "kaniko-buildrun"}]).To(BeNumerically(">", 0.0))
			Expect(buildRunHistogramMetrics["build_buildrun_taskrun_pod_rampup_duration_seconds"][buildRunLabels{"kaniko", "default", "kaniko-build", "kaniko-buildrun"}]).To(BeNumerically(">", 0.0))
		})
	})

	Context("when create a new buildpacks buildrun", func() {

		It("should increase the buildpacks build count", func() {
			Expect(buildCounterMetrics).To(HaveKey("build_builds_registered_total"))
			Expect(buildCounterMetrics["build_builds_registered_total"][buildLabels{"buildpacks", "default", "buildpacks-build"}]).To(Equal(1.0))
		})

		It("should increase the buildpacks buildrun count", func() {
			Expect(buildRunCounterMetrics).To(HaveKey("build_buildruns_completed_total"))
			Expect(buildRunCounterMetrics["build_buildruns_completed_total"][buildRunLabels{"buildpacks", "default", "buildpacks-build", "buildpacks-buildrun"}]).To(Equal(1.0))
		})

		It("should record the buildpacks buildrun establish time", func() {
			Expect(buildRunHistogramMetrics).To(HaveKey("build_buildrun_establish_duration_seconds"))
			Expect(buildRunHistogramMetrics["build_buildrun_establish_duration_seconds"][buildRunLabels{"buildpacks", "default", "buildpacks-build", "buildpacks-buildrun"}]).To(Equal(1.0))
		})

		It("should record the buildpacks buildrun completion time", func() {
			Expect(buildRunHistogramMetrics).To(HaveKey("build_buildrun_completion_duration_seconds"))
			Expect(buildRunHistogramMetrics["build_buildrun_completion_duration_seconds"][buildRunLabels{"buildpacks", "default", "buildpacks-build", "buildpacks-buildrun"}]).To(Equal(200.0))
		})

		It("should record the buildpacks ramp-up durations", func() {
			Expect(buildRunHistogramMetrics).To(HaveKey("build_buildrun_rampup_duration_seconds"))
			Expect(buildRunHistogramMetrics).To(HaveKey("build_buildrun_taskrun_rampup_duration_seconds"))
			Expect(buildRunHistogramMetrics).To(HaveKey("build_buildrun_taskrun_pod_rampup_duration_seconds"))

			Expect(buildRunHistogramMetrics["build_buildrun_rampup_duration_seconds"][buildRunLabels{"buildpacks", "default", "buildpacks-build", "buildpacks-buildrun"}]).To(BeNumerically(">", 0.0))
			Expect(buildRunHistogramMetrics["build_buildrun_taskrun_rampup_duration_seconds"][buildRunLabels{"buildpacks", "default", "buildpacks-build", "buildpacks-buildrun"}]).To(BeNumerically(">", 0.0))
			Expect(buildRunHistogramMetrics["build_buildrun_taskrun_pod_rampup_duration_seconds"][buildRunLabels{"buildpacks", "default", "buildpacks-build", "buildpacks-buildrun"}]).To(BeNumerically(">", 0.0))
		})
	})
})
