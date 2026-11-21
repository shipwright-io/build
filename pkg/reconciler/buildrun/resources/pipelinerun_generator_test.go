// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
)

var _ = Describe("PipelineRunGenerator", func() {
	var (
		build    *buildapi.Build
		buildRun *buildapi.BuildRun
		strategy *buildapi.BuildStrategy
	)

	BeforeEach(func() {
		build = &buildapi.Build{
			ObjectMeta: metav1.ObjectMeta{Name: "build", Namespace: "ns"},
		}
		buildRun = &buildapi.BuildRun{
			ObjectMeta: metav1.ObjectMeta{Name: "buildrun", Namespace: "ns"},
		}
		strategy = &buildapi.BuildStrategy{
			ObjectMeta: metav1.ObjectMeta{Name: "strategy", Namespace: "ns"},
		}
	})

	generate := func() *pipelineapi.PipelineRun {
		g := resources.NewPipelineRunGenerator(config.NewDefaultConfig(), build, buildRun, "sa", strategy)
		Expect(g.InitializeExecutor()).To(Succeed())
		Expect(g.ApplyInfrastructureConfiguration()).To(Succeed())

		pipelineRun, ok := g.GetExecutor().(*pipelineapi.PipelineRun)
		Expect(ok).To(BeTrue())
		return pipelineRun
	}

	Context("ApplyInfrastructureConfiguration with hostUsers", func() {
		It("should not set hostUsers when nothing specifies it", func() {
			pipelineRun := generate()

			Expect(pipelineRun.Spec.TaskRunTemplate.PodTemplate).ToNot(BeNil())
			Expect(pipelineRun.Spec.TaskRunTemplate.PodTemplate.HostUsers).To(BeNil())
		})

		It("should set hostUsers false when it is the only pod template setting", func() {
			buildRun.Spec.HostUsers = ptr.To(false)

			pipelineRun := generate()

			Expect(pipelineRun.Spec.TaskRunTemplate.PodTemplate).ToNot(BeNil())
			Expect(pipelineRun.Spec.TaskRunTemplate.PodTemplate.HostUsers).To(Equal(ptr.To(false)))
		})

		It("should set an explicit hostUsers true", func() {
			build.Spec.HostUsers = ptr.To(true)

			pipelineRun := generate()

			Expect(pipelineRun.Spec.TaskRunTemplate.PodTemplate).ToNot(BeNil())
			Expect(pipelineRun.Spec.TaskRunTemplate.PodTemplate.HostUsers).To(Equal(ptr.To(true)))
		})

		It("should not let Build or BuildRun opt out of a user namespace that the BuildStrategy requires", func() {
			strategy.Spec.HostUsers = ptr.To(false)
			build.Spec.HostUsers = ptr.To(true)
			buildRun.Spec.HostUsers = ptr.To(true)

			pipelineRun := generate()

			Expect(pipelineRun.Spec.TaskRunTemplate.PodTemplate).ToNot(BeNil())
			Expect(pipelineRun.Spec.TaskRunTemplate.PodTemplate.HostUsers).To(Equal(ptr.To(false)))
		})

		It("should keep other pod template settings when hostUsers is set", func() {
			build.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": "node-1"}
			buildRun.Spec.RuntimeClassName = ptr.To("kata-containers")
			buildRun.Spec.HostUsers = ptr.To(false)

			pipelineRun := generate()

			podTemplate := pipelineRun.Spec.TaskRunTemplate.PodTemplate
			Expect(podTemplate).ToNot(BeNil())
			Expect(podTemplate.HostUsers).To(Equal(ptr.To(false)))
			Expect(podTemplate.NodeSelector).To(HaveKeyWithValue("kubernetes.io/hostname", "node-1"))
			Expect(podTemplate.RuntimeClassName).To(Equal(ptr.To("kata-containers")))
			Expect(podTemplate.EnableServiceLinks).To(Equal(ptr.To(false)))
		})
	})
})
