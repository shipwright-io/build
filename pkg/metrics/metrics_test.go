// Copyright The Shipwright Contributors
// 
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/shipwright-io/build/pkg/config"
)

var _ = Describe("Custom Metrics", func() {

	var (
		buildStrategy, namespace string
	)

	Context("when create a new kaniko buildrun", func() {
		buildStrategy = "kaniko"
		namespace = "default"

		InitPrometheus(config.NewDefaultConfig())

		BuildCountInc(buildStrategy)
		BuildRunCountInc(buildStrategy)
		BuildRunEstablishObserve(buildStrategy, namespace, time.Duration(1)*time.Second)
		BuildRunCompletionObserve(buildStrategy, namespace, time.Duration(200)*time.Second)
		BuildRunRampUpDurationObserve(buildStrategy, namespace, time.Duration(1)*time.Second)
		TaskRunRampUpDurationObserve(buildStrategy, namespace, time.Duration(2)*time.Second)
		TaskRunPodRampUpDurationObserve(buildStrategy, namespace, time.Duration(3)*time.Second)

		It("should increase the kaniko build count", func() {
			buildCount, _ := buildCount.GetMetricWithLabelValues(buildStrategy)
			Expect(testutil.ToFloat64(buildCount)).To(Equal(float64(1)))
		})

		It("should increase the kaniko buildrun count", func() {
			buildRunCount, _ := buildRunCount.GetMetricWithLabelValues(buildStrategy)
			Expect(testutil.ToFloat64(buildRunCount)).To(Equal(float64(1)))
		})

		It("should record the kaniko buildrun establish time", func() {
			buildRunEstablishDuration, err := buildRunEstablishDuration.GetMetricWithLabelValues(buildStrategy, namespace)
			Expect(buildRunEstablishDuration).NotTo(BeNil())
			Expect(err).To(BeNil())
		})

		It("should record the kaniko buildrun completion time", func() {
			buildRunCompletionDuration, err := buildRunCompletionDuration.GetMetricWithLabelValues(buildStrategy, namespace)
			Expect(buildRunCompletionDuration).NotTo(BeNil())
			Expect(err).To(BeNil())
		})

		It("should record the kaniko ramp-up durations", func() {
			for _, metric := range []*prometheus.HistogramVec{buildRunRampUpDuration, taskRunRampUpDuration, taskRunPodRampUpDuration} {
				metricRampUpDuration, err := metric.GetMetricWithLabelValues(buildStrategy, namespace)
				Expect(metricRampUpDuration).NotTo(BeNil())
				Expect(err).To(BeNil())
			}
		})
	})

	Context("when create a new buildpacks buildrun", func() {
		buildStrategy = "buildpacks"
		namespace = "test"

		InitPrometheus(config.NewDefaultConfig())

		BuildCountInc(buildStrategy)
		BuildRunCountInc(buildStrategy)
		BuildRunEstablishObserve(buildStrategy, namespace, time.Duration(1)*time.Second)
		BuildRunCompletionObserve(buildStrategy, namespace, time.Duration(100)*time.Second)
		BuildRunRampUpDurationObserve(buildStrategy, namespace, time.Duration(1)*time.Second)
		TaskRunRampUpDurationObserve(buildStrategy, namespace, time.Duration(2)*time.Second)
		TaskRunPodRampUpDurationObserve(buildStrategy, namespace, time.Duration(3)*time.Second)

		It("should increase the buildpacks build count", func() {
			buildCount, _ := buildCount.GetMetricWithLabelValues(buildStrategy)
			Expect(testutil.ToFloat64(buildCount)).To(Equal(float64(1)))
		})

		It("should increase the buildpacks buildrun count", func() {
			buildRunCount, _ := buildRunCount.GetMetricWithLabelValues(buildStrategy)
			Expect(testutil.ToFloat64(buildRunCount)).To(Equal(float64(1)))
		})

		It("should record the buildpacks buildrun establish time", func() {
			buildRunEstablishDuration, err := buildRunEstablishDuration.GetMetricWithLabelValues(buildStrategy, namespace)
			Expect(buildRunEstablishDuration).NotTo(BeNil())
			Expect(err).To(BeNil())
		})

		It("should record the buildpacks buildrun completion time", func() {
			buildRunCompletionDuration, err := buildRunCompletionDuration.GetMetricWithLabelValues(buildStrategy, namespace)
			Expect(buildRunCompletionDuration).NotTo(BeNil())
			Expect(err).To(BeNil())
		})

		It("should record the buildpacks ramp-up durations", func() {
			for _, metric := range []*prometheus.HistogramVec{buildRunRampUpDuration, taskRunRampUpDuration, taskRunPodRampUpDuration} {
				metricRampUpDuration, err := metric.GetMetricWithLabelValues(buildStrategy, namespace)
				Expect(metricRampUpDuration).NotTo(BeNil())
				Expect(err).To(BeNil())
			}
		})
	})
})
