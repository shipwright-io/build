// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package metrics_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	io_prometheus_client "github.com/prometheus/client_model/go"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/shipwright-io/build/pkg/config"
	. "github.com/shipwright-io/build/pkg/metrics"
)

type buildLabels struct {
	buildStrategy string
	namespace     string
	build         string
	strategyKind  string
	sourceType    string
}

type buildRunLabels struct {
	buildStrategy string
	namespace     string
	build         string
	buildRun      string
	strategyKind  string
	executor      string
	sourceType    string
}

type buildRunResultLabels struct {
	buildStrategy string
	namespace     string
	build         string
	buildRun      string
	strategyKind  string
	executor      string
	sourceType    string
	result        string
}

type buildRunFailureReasonLabels struct {
	buildStrategy string
	namespace     string
	build         string
	buildRun      string
	strategyKind  string
	executor      string
	sourceType    string
	reason        string
}

var (
	buildCounterMetrics            map[string]map[buildLabels]float64
	buildRunHistogramMetrics       map[string]map[buildRunLabels]float64
	buildRunResultCounterMetrics   map[string]map[buildRunResultLabels]float64
	buildRunFailureReasonMetrics   map[string]map[buildRunFailureReasonLabels]float64
	buildRunsActiveGaugeMetrics    map[string]map[buildRunLabels]float64
	buildStrategyCountGaugeMetrics map[string]map[string]float64
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
		case StrategyKindLabel:
			result.strategyKind = *label.Value
		case SourceTypeLabel:
			result.sourceType = *label.Value
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
		case StrategyKindLabel:
			result.strategyKind = *label.Value
		case ExecutorLabel:
			result.executor = *label.Value
		case SourceTypeLabel:
			result.sourceType = *label.Value
		}
	}

	return result
}

func promLabelPairToBuildRunResultLabels(in []*io_prometheus_client.LabelPair) buildRunResultLabels {
	result := buildRunResultLabels{}
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
		case StrategyKindLabel:
			result.strategyKind = *label.Value
		case ExecutorLabel:
			result.executor = *label.Value
		case SourceTypeLabel:
			result.sourceType = *label.Value
		case ResultLabel:
			result.result = *label.Value
		}
	}

	return result
}

func promLabelPairToBuildRunFailureReasonLabels(in []*io_prometheus_client.LabelPair) buildRunFailureReasonLabels {
	result := buildRunFailureReasonLabels{}
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
		case StrategyKindLabel:
			result.strategyKind = *label.Value
		case ExecutorLabel:
			result.executor = *label.Value
		case SourceTypeLabel:
			result.sourceType = *label.Value
		case "reason":
			result.reason = *label.Value
		}
	}

	return result
}

var _ = BeforeSuite(func() {
	buildCounterMetrics = map[string]map[buildLabels]float64{}
	buildRunHistogramMetrics = map[string]map[buildRunLabels]float64{}
	buildRunResultCounterMetrics = map[string]map[buildRunResultLabels]float64{}
	buildRunFailureReasonMetrics = map[string]map[buildRunFailureReasonLabels]float64{}
	buildRunsActiveGaugeMetrics = map[string]map[buildRunLabels]float64{}
	buildStrategyCountGaugeMetrics = map[string]map[string]float64{}

	type testEntry struct {
		buildStrategy string
		namespace     string
		build         string
		buildRun      string
		strategyKind  string
		executor      string
		sourceType    string
	}

	var (
		testLabels = []testEntry{
			{buildStrategy: "kaniko", namespace: "default", build: "kaniko-build", buildRun: "kaniko-buildrun", strategyKind: "ClusterBuildStrategy", executor: "TaskRun", sourceType: "Git"},
			{buildStrategy: "buildpacks", namespace: "default", build: "buildpacks-build", buildRun: "buildpacks-buildrun", strategyKind: "BuildStrategy", executor: "PipelineRun", sourceType: "OCI"},
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
	buildRunResultCounterMetrics["build_buildrun_result_total"] = map[buildRunResultLabels]float64{}
	buildRunFailureReasonMetrics["build_buildrun_failure_reason_total"] = map[buildRunFailureReasonLabels]float64{}
	buildRunsActiveGaugeMetrics["build_buildruns_active"] = map[buildRunLabels]float64{}
	buildStrategyCountGaugeMetrics["build_buildstrategy_count"] = map[string]float64{}

	// initialize the histogram metrics result map with empty maps
	for _, name := range knownHistogramMetrics {
		buildRunHistogramMetrics[name] = map[buildRunLabels]float64{}
	}

	// initialize prometheus (second init should be no-op)
	config := config.NewDefaultConfig()
	config.Prometheus.EnabledLabels = []string{
		BuildStrategyLabel, NamespaceLabel, BuildLabel, BuildRunLabel,
		StrategyKindLabel, ExecutorLabel, ResultLabel, SourceTypeLabel,
	}
	InitPrometheus(config)

	// and fire some examples
	for _, entry := range testLabels {
		BuildCountInc(entry.buildStrategy, entry.namespace, entry.build, entry.strategyKind, entry.sourceType)
		BuildRunEstablishObserve(entry.buildStrategy, entry.namespace, entry.build, entry.buildRun, entry.strategyKind, entry.executor, entry.sourceType, time.Duration(1)*time.Second)
		BuildRunCompletionObserve(entry.buildStrategy, entry.namespace, entry.build, entry.buildRun, entry.strategyKind, entry.executor, entry.sourceType, time.Duration(200)*time.Second)
		BuildRunRampUpDurationObserve(entry.buildStrategy, entry.namespace, entry.build, entry.buildRun, entry.strategyKind, entry.executor, entry.sourceType, time.Duration(1)*time.Second)
		TaskRunRampUpDurationObserve(entry.buildStrategy, entry.namespace, entry.build, entry.buildRun, entry.strategyKind, entry.executor, entry.sourceType, time.Duration(2)*time.Second)
		TaskRunPodRampUpDurationObserve(entry.buildStrategy, entry.namespace, entry.build, entry.buildRun, entry.strategyKind, entry.executor, entry.sourceType, time.Duration(3)*time.Second)

		// Record result metrics
		BuildRunResultCountInc(entry.buildStrategy, entry.namespace, entry.build, entry.buildRun, entry.strategyKind, entry.executor, ResultSucceeded, entry.sourceType)

		// Record active gauge (inc and dec to simulate a completed run)
		BuildRunsActiveInc(entry.buildStrategy, entry.namespace, entry.build, entry.buildRun, entry.strategyKind, entry.executor, entry.sourceType)
	}

	// Record some failures
	BuildRunResultCountInc("kaniko", "default", "kaniko-build", "kaniko-buildrun-fail", "ClusterBuildStrategy", "TaskRun", ResultFailed, "Git")
	BuildRunFailureReasonCountInc("kaniko", "default", "kaniko-build", "kaniko-buildrun-fail", "ClusterBuildStrategy", "TaskRun", "Git", "StepOutOfMemory")

	BuildRunResultCountInc("buildpacks", "default", "buildpacks-build", "buildpacks-buildrun-cancel", "BuildStrategy", "PipelineRun", ResultCancelled, "OCI")
	BuildRunFailureReasonCountInc("buildpacks", "default", "buildpacks-build", "buildpacks-buildrun-cancel", "BuildStrategy", "PipelineRun", "OCI", "BuildRunCanceled")

	BuildRunResultCountInc("kaniko", "default", "kaniko-build", "kaniko-buildrun-timeout", "ClusterBuildStrategy", "TaskRun", ResultTimeout, "Git")
	BuildRunFailureReasonCountInc("kaniko", "default", "kaniko-build", "kaniko-buildrun-timeout", "ClusterBuildStrategy", "TaskRun", "Git", "BuildRunTimeout")

	// Decrement active for the first entry to test net gauge
	BuildRunsActiveDec(testLabels[0].buildStrategy, testLabels[0].namespace, testLabels[0].build, testLabels[0].buildRun, testLabels[0].strategyKind, testLabels[0].executor, testLabels[0].sourceType)

	// Set strategy counts
	BuildStrategyCountSet("BuildStrategy", 3)
	BuildStrategyCountSet("ClusterBuildStrategy", 5)

	// gather metrics from prometheus and fill the result maps
	metrics, err := crmetrics.Registry.Gather()
	if err != nil {
		Fail(err.Error())
	}

	for _, metricFamily := range metrics {
		switch metricFamily.GetName() {
		case "build_builds_registered_total":
			for _, metric := range metricFamily.GetMetric() {
				buildCounterMetrics[metricFamily.GetName()][promLabelPairToBuildLabels(metric.GetLabel())] = metric.GetCounter().GetValue()
			}

		case "build_buildrun_result_total":
			for _, metric := range metricFamily.GetMetric() {
				buildRunResultCounterMetrics[metricFamily.GetName()][promLabelPairToBuildRunResultLabels(metric.GetLabel())] = metric.GetCounter().GetValue()
			}

		case "build_buildrun_failure_reason_total":
			for _, metric := range metricFamily.GetMetric() {
				buildRunFailureReasonMetrics[metricFamily.GetName()][promLabelPairToBuildRunFailureReasonLabels(metric.GetLabel())] = metric.GetCounter().GetValue()
			}

		case "build_buildruns_active":
			for _, metric := range metricFamily.GetMetric() {
				buildRunsActiveGaugeMetrics[metricFamily.GetName()][promLabelPairToBuildRunLabels(metric.GetLabel())] = metric.GetGauge().GetValue()
			}

		case "build_buildstrategy_count":
			for _, metric := range metricFamily.GetMetric() {
				for _, label := range metric.GetLabel() {
					if *label.Name == StrategyKindLabel {
						buildStrategyCountGaugeMetrics[metricFamily.GetName()][*label.Value] = metric.GetGauge().GetValue()
					}
				}
			}

		default:
			if metricFamily.GetType() == io_prometheus_client.MetricType_HISTOGRAM {
				if _, ok := buildRunHistogramMetrics[metricFamily.GetName()]; ok {
					for _, metric := range metricFamily.GetMetric() {
						buildRunHistogramMetrics[metricFamily.GetName()][promLabelPairToBuildRunLabels(metric.GetLabel())] = metric.GetHistogram().GetSampleSum()
					}
				}
			}
		}
	}
})

var _ = Describe("Custom Metrics", func() {
	Context("when create a new kaniko buildrun", func() {
		It("should increase the kaniko build count", func() {
			Expect(buildCounterMetrics).To(HaveKey("build_builds_registered_total"))
			Expect(buildCounterMetrics["build_builds_registered_total"][buildLabels{"kaniko", "default", "kaniko-build", "ClusterBuildStrategy", "Git"}]).To(Equal(1.0))
		})

		It("should record the kaniko buildrun establish time", func() {
			Expect(buildRunHistogramMetrics).To(HaveKey("build_buildrun_establish_duration_seconds"))
			Expect(buildRunHistogramMetrics["build_buildrun_establish_duration_seconds"][buildRunLabels{"kaniko", "default", "kaniko-build", "kaniko-buildrun", "ClusterBuildStrategy", "TaskRun", "Git"}]).To(Equal(1.0))
		})

		It("should record the kaniko buildrun completion time", func() {
			Expect(buildRunHistogramMetrics).To(HaveKey("build_buildrun_completion_duration_seconds"))
			Expect(buildRunHistogramMetrics["build_buildrun_completion_duration_seconds"][buildRunLabels{"kaniko", "default", "kaniko-build", "kaniko-buildrun", "ClusterBuildStrategy", "TaskRun", "Git"}]).To(Equal(200.0))
		})

		It("should record the kaniko ramp-up durations", func() {
			Expect(buildRunHistogramMetrics).To(HaveKey("build_buildrun_rampup_duration_seconds"))
			Expect(buildRunHistogramMetrics).To(HaveKey("build_buildrun_taskrun_rampup_duration_seconds"))
			Expect(buildRunHistogramMetrics).To(HaveKey("build_buildrun_taskrun_pod_rampup_duration_seconds"))

			Expect(buildRunHistogramMetrics["build_buildrun_rampup_duration_seconds"][buildRunLabels{"kaniko", "default", "kaniko-build", "kaniko-buildrun", "ClusterBuildStrategy", "TaskRun", "Git"}]).To(BeNumerically(">", 0.0))
			Expect(buildRunHistogramMetrics["build_buildrun_taskrun_rampup_duration_seconds"][buildRunLabels{"kaniko", "default", "kaniko-build", "kaniko-buildrun", "ClusterBuildStrategy", "TaskRun", "Git"}]).To(BeNumerically(">", 0.0))
			Expect(buildRunHistogramMetrics["build_buildrun_taskrun_pod_rampup_duration_seconds"][buildRunLabels{"kaniko", "default", "kaniko-build", "kaniko-buildrun", "ClusterBuildStrategy", "TaskRun", "Git"}]).To(BeNumerically(">", 0.0))
		})
	})

	Context("when create a new buildpacks buildrun", func() {

		It("should increase the buildpacks build count", func() {
			Expect(buildCounterMetrics).To(HaveKey("build_builds_registered_total"))
			Expect(buildCounterMetrics["build_builds_registered_total"][buildLabels{"buildpacks", "default", "buildpacks-build", "BuildStrategy", "OCI"}]).To(Equal(1.0))
		})

		It("should record the buildpacks buildrun establish time", func() {
			Expect(buildRunHistogramMetrics).To(HaveKey("build_buildrun_establish_duration_seconds"))
			Expect(buildRunHistogramMetrics["build_buildrun_establish_duration_seconds"][buildRunLabels{"buildpacks", "default", "buildpacks-build", "buildpacks-buildrun", "BuildStrategy", "PipelineRun", "OCI"}]).To(Equal(1.0))
		})

		It("should record the buildpacks buildrun completion time", func() {
			Expect(buildRunHistogramMetrics).To(HaveKey("build_buildrun_completion_duration_seconds"))
			Expect(buildRunHistogramMetrics["build_buildrun_completion_duration_seconds"][buildRunLabels{"buildpacks", "default", "buildpacks-build", "buildpacks-buildrun", "BuildStrategy", "PipelineRun", "OCI"}]).To(Equal(200.0))
		})

		It("should record the buildpacks ramp-up durations", func() {
			Expect(buildRunHistogramMetrics).To(HaveKey("build_buildrun_rampup_duration_seconds"))
			Expect(buildRunHistogramMetrics).To(HaveKey("build_buildrun_taskrun_rampup_duration_seconds"))
			Expect(buildRunHistogramMetrics).To(HaveKey("build_buildrun_taskrun_pod_rampup_duration_seconds"))

			Expect(buildRunHistogramMetrics["build_buildrun_rampup_duration_seconds"][buildRunLabels{"buildpacks", "default", "buildpacks-build", "buildpacks-buildrun", "BuildStrategy", "PipelineRun", "OCI"}]).To(BeNumerically(">", 0.0))
			Expect(buildRunHistogramMetrics["build_buildrun_taskrun_rampup_duration_seconds"][buildRunLabels{"buildpacks", "default", "buildpacks-build", "buildpacks-buildrun", "BuildStrategy", "PipelineRun", "OCI"}]).To(BeNumerically(">", 0.0))
			Expect(buildRunHistogramMetrics["build_buildrun_taskrun_pod_rampup_duration_seconds"][buildRunLabels{"buildpacks", "default", "buildpacks-build", "buildpacks-buildrun", "BuildStrategy", "PipelineRun", "OCI"}]).To(BeNumerically(">", 0.0))
		})
	})

	Context("build_buildrun_result_total metric", func() {
		It("should record succeeded results", func() {
			Expect(buildRunResultCounterMetrics).To(HaveKey("build_buildrun_result_total"))
			Expect(buildRunResultCounterMetrics["build_buildrun_result_total"][buildRunResultLabels{
				"kaniko", "default", "kaniko-build", "kaniko-buildrun", "ClusterBuildStrategy", "TaskRun", "Git", ResultSucceeded,
			}]).To(Equal(1.0))
		})

		It("should record failed results", func() {
			Expect(buildRunResultCounterMetrics["build_buildrun_result_total"][buildRunResultLabels{
				"kaniko", "default", "kaniko-build", "kaniko-buildrun-fail", "ClusterBuildStrategy", "TaskRun", "Git", ResultFailed,
			}]).To(Equal(1.0))
		})

		It("should record cancelled results", func() {
			Expect(buildRunResultCounterMetrics["build_buildrun_result_total"][buildRunResultLabels{
				"buildpacks", "default", "buildpacks-build", "buildpacks-buildrun-cancel", "BuildStrategy", "PipelineRun", "OCI", ResultCancelled,
			}]).To(Equal(1.0))
		})

		It("should record timeout results", func() {
			Expect(buildRunResultCounterMetrics["build_buildrun_result_total"][buildRunResultLabels{
				"kaniko", "default", "kaniko-build", "kaniko-buildrun-timeout", "ClusterBuildStrategy", "TaskRun", "Git", ResultTimeout,
			}]).To(Equal(1.0))
		})
	})

	Context("build_buildrun_failure_reason_total metric", func() {
		It("should record OOM failure reason", func() {
			Expect(buildRunFailureReasonMetrics).To(HaveKey("build_buildrun_failure_reason_total"))
			Expect(buildRunFailureReasonMetrics["build_buildrun_failure_reason_total"][buildRunFailureReasonLabels{
				"kaniko", "default", "kaniko-build", "kaniko-buildrun-fail", "ClusterBuildStrategy", "TaskRun", "Git", "StepOutOfMemory",
			}]).To(Equal(1.0))
		})

		It("should record cancel failure reason", func() {
			Expect(buildRunFailureReasonMetrics["build_buildrun_failure_reason_total"][buildRunFailureReasonLabels{
				"buildpacks", "default", "buildpacks-build", "buildpacks-buildrun-cancel", "BuildStrategy", "PipelineRun", "OCI", "BuildRunCanceled",
			}]).To(Equal(1.0))
		})

		It("should record timeout failure reason", func() {
			Expect(buildRunFailureReasonMetrics["build_buildrun_failure_reason_total"][buildRunFailureReasonLabels{
				"kaniko", "default", "kaniko-build", "kaniko-buildrun-timeout", "ClusterBuildStrategy", "TaskRun", "Git", "BuildRunTimeout",
			}]).To(Equal(1.0))
		})
	})

	Context("build_buildruns_active gauge metric", func() {
		It("should show net active count after inc and dec", func() {
			Expect(buildRunsActiveGaugeMetrics).To(HaveKey("build_buildruns_active"))
			// kaniko was incremented then decremented, so should be 0
			Expect(buildRunsActiveGaugeMetrics["build_buildruns_active"][buildRunLabels{
				"kaniko", "default", "kaniko-build", "kaniko-buildrun", "ClusterBuildStrategy", "TaskRun", "Git",
			}]).To(Equal(0.0))
		})

		It("should show positive count when only incremented", func() {
			// buildpacks was only incremented, so should be 1
			Expect(buildRunsActiveGaugeMetrics["build_buildruns_active"][buildRunLabels{
				"buildpacks", "default", "buildpacks-build", "buildpacks-buildrun", "BuildStrategy", "PipelineRun", "OCI",
			}]).To(Equal(1.0))
		})
	})

	Context("build_buildstrategy_count gauge metric", func() {
		It("should show correct BuildStrategy count", func() {
			Expect(buildStrategyCountGaugeMetrics).To(HaveKey("build_buildstrategy_count"))
			Expect(buildStrategyCountGaugeMetrics["build_buildstrategy_count"]["BuildStrategy"]).To(Equal(3.0))
		})

		It("should show correct ClusterBuildStrategy count", func() {
			Expect(buildStrategyCountGaugeMetrics["build_buildstrategy_count"]["ClusterBuildStrategy"]).To(Equal(5.0))
		})
	})
})
