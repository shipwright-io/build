// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package steps_test

import (
	"fmt"

	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources/steps"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("UpdateSecurityContext", func() {

	var buildStrategySecurityContext *buildapi.BuildStrategySecurityContext
	var buildStrategySteps []buildapi.Step
	var taskRunSpec *pipelineapi.TaskSpec
	var taskRunAnnotations map[string]string

	BeforeEach(func() {
		buildStrategySteps = []buildapi.Step{{
			Name: "first-step",
			SecurityContext: &corev1.SecurityContext{
				RunAsUser:  pointer.Int64(891),
				RunAsGroup: pointer.Int64(1210),
			},
		}, {
			Name: "second-step",
			SecurityContext: &corev1.SecurityContext{
				RunAsUser: pointer.Int64(891),
			},
		}, {
			Name: "third-step",
		}}

		taskRunSpec = &pipelineapi.TaskSpec{
			Steps: []pipelineapi.Step{{
				Name: "shp-source-default",
				SecurityContext: &corev1.SecurityContext{
					RunAsUser:  pointer.Int64(1000),
					RunAsGroup: pointer.Int64(1000),
				},
			}, {
				Name: "first-step",
				SecurityContext: &corev1.SecurityContext{
					RunAsUser:  pointer.Int64(891),
					RunAsGroup: pointer.Int64(1210),
				},
			}, {
				Name: "second-step",
				SecurityContext: &corev1.SecurityContext{
					RunAsUser: pointer.Int64(891),
				},
			}, {
				Name: "third-step",
			}},
		}
	})

	JustBeforeEach(func() {
		taskRunAnnotations = make(map[string]string)
		steps.UpdateSecurityContext(taskRunSpec, taskRunAnnotations, buildStrategySteps, buildStrategySecurityContext)
	})

	Context("for a build strategy without a securityContext", func() {

		BeforeEach(func() {
			buildStrategySecurityContext = nil
		})

		It("does not change the step's securityContext", func() {
			Expect(taskRunSpec.Steps[0].SecurityContext.RunAsUser).To(Equal(pointer.Int64(1000)))
			Expect(taskRunSpec.Steps[0].SecurityContext.RunAsGroup).To(Equal(pointer.Int64(1000)))
			Expect(taskRunSpec.Steps[1].SecurityContext.RunAsUser).To(Equal(pointer.Int64(891)))
			Expect(taskRunSpec.Steps[1].SecurityContext.RunAsGroup).To(Equal(pointer.Int64(1210)))
			Expect(taskRunSpec.Steps[2].SecurityContext.RunAsUser).To(Equal(pointer.Int64(891)))
			Expect(taskRunSpec.Steps[2].SecurityContext.RunAsGroup).To(BeNil())
			Expect(taskRunSpec.Steps[3].SecurityContext).To(BeNil())
		})

		It("does not modify annotations", func() {
			Expect(taskRunAnnotations).To(HaveLen(0))
		})

		It("does not introduce volumes", func() {
			Expect(taskRunSpec.Volumes).To(HaveLen(0))
			Expect(taskRunSpec.Steps[0].VolumeMounts).To(HaveLen(0))
			Expect(taskRunSpec.Steps[1].VolumeMounts).To(HaveLen(0))
			Expect(taskRunSpec.Steps[2].VolumeMounts).To(HaveLen(0))
			Expect(taskRunSpec.Steps[3].VolumeMounts).To(HaveLen(0))
		})
	})

	Context("for build strategy with a securityContext", func() {

		BeforeEach(func() {
			buildStrategySecurityContext = &buildapi.BuildStrategySecurityContext{
				RunAsUser:  123,
				RunAsGroup: 456,
			}
		})

		It("changes the securityContext of shipwright-managed steps", func() {
			Expect(taskRunSpec.Steps[0].SecurityContext.RunAsUser).To(Equal(pointer.Int64(123)))
			Expect(taskRunSpec.Steps[0].SecurityContext.RunAsGroup).To(Equal(pointer.Int64(456)))
		})

		It("does not change the securityContext of a strategy step that has runAsUser and runAsGroup set", func() {
			Expect(taskRunSpec.Steps[1].SecurityContext.RunAsUser).To(Equal(pointer.Int64(891)))
			Expect(taskRunSpec.Steps[1].SecurityContext.RunAsGroup).To(Equal(pointer.Int64(1210)))
		})

		It("changes the securityContext of a strategy step that does not have both runAsUser and runAsGroup set", func() {
			Expect(taskRunSpec.Steps[2].SecurityContext.RunAsUser).To(Equal(pointer.Int64(891)))
			Expect(taskRunSpec.Steps[2].SecurityContext.RunAsGroup).To(Equal(pointer.Int64(456)))
		})

		It("introduces a securityContext for a strategy step that does not have one", func() {
			Expect(taskRunSpec.Steps[3].SecurityContext).ToNot(BeNil())
			Expect(taskRunSpec.Steps[3].SecurityContext.RunAsUser).To(Equal(pointer.Int64(123)))
			Expect(taskRunSpec.Steps[3].SecurityContext.RunAsGroup).To(Equal(pointer.Int64(456)))
		})

		It("adds annotations", func() {
			Expect(taskRunAnnotations).To(HaveLen(2))
			Expect(taskRunAnnotations[steps.AnnotationSecurityContextGroup]).To(Equal("shp:x:456"))
			Expect(taskRunAnnotations[steps.AnnotationSecurityContextPasswd]).To(Equal("shp:x:123:456:shp:/shared-home:/sbin/nologin"))
		})

		It("does introduces a volume", func() {
			Expect(taskRunSpec.Volumes).To(HaveLen(1))
			Expect(taskRunSpec.Volumes[0]).To(BeEquivalentTo(corev1.Volume{
				Name: steps.VolumeNameSecurityContext,
				VolumeSource: corev1.VolumeSource{
					DownwardAPI: &corev1.DownwardAPIVolumeSource{
						DefaultMode: pointer.Int32(0444),

						Items: []corev1.DownwardAPIVolumeFile{{
							Path: "group",
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: fmt.Sprintf("metadata.annotations['%s']", steps.AnnotationSecurityContextGroup),
							},
						}, {
							Path: "passwd",
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: fmt.Sprintf("metadata.annotations['%s']", steps.AnnotationSecurityContextPasswd),
							},
						}},
					},
				},
			}))
		})

		It("introduces volume mounts to a shipwright-managed step", func() {
			Expect(taskRunSpec.Steps[0].VolumeMounts).To(HaveLen(2))
			Expect(taskRunSpec.Steps[0].VolumeMounts[0]).To(BeEquivalentTo(corev1.VolumeMount{
				Name:      steps.VolumeNameSecurityContext,
				MountPath: "/etc/group",
				SubPath:   "group",
			}))
			Expect(taskRunSpec.Steps[0].VolumeMounts[1]).To(BeEquivalentTo(corev1.VolumeMount{
				Name:      steps.VolumeNameSecurityContext,
				MountPath: "/etc/passwd",
				SubPath:   "passwd",
			}))

		})

		It("does not introduce volume mounts to strategy-defined steps", func() {
			Expect(taskRunSpec.Steps[1].VolumeMounts).To(HaveLen(0))
			Expect(taskRunSpec.Steps[2].VolumeMounts).To(HaveLen(0))
			Expect(taskRunSpec.Steps[3].VolumeMounts).To(HaveLen(0))
		})
	})
})
