// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package sources_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources/sources"
)

var _ = Describe("LocalCopy", func() {
	cfg := config.NewDefaultConfig()

	Context("when LocalCopy source type is informed", func() {
		var taskSpec *tektonv1beta1.TaskSpec

		BeforeEach(func() {
			taskSpec = &tektonv1beta1.TaskSpec{}
			sources.AppendLocalCopyStep(cfg, taskSpec, &metav1.Duration{Duration: time.Minute})
		})

		It("produces a local-copy step", func() {
			Expect(len(taskSpec.Results)).To(Equal(0))
			Expect(len(taskSpec.Steps)).To(Equal(1))
			Expect(taskSpec.Steps[0].Name).To(Equal(sources.WaiterContainerName))
			Expect(taskSpec.Steps[0].Image).To(Equal(cfg.WaiterContainerTemplate.Image))
			Expect(taskSpec.Steps[0].Args).To(Equal([]string{"start", "--timeout=1m0s"}))
		})
	})
})
