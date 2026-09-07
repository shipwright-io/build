// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	crc "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/shipwright-io/build/pkg/controller/fakes"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
)

var _ = Describe("Volumes", func() {
	var (
		ctx       context.Context
		client    *fakes.FakeClient
		namespace string
	)

	BeforeEach(func() {
		ctx = context.TODO()
		namespace = "test-ns"
		client = &fakes.FakeClient{}
	})

	Context("when checking TaskRun volumes", func() {
		It("succeeds when referenced Secret and ConfigMap exist", func() {
			client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object, _ ...crc.GetOption) error {
				if nn.Name == "my-secret" || nn.Name == "my-configmap" {
					return nil
				}
				return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
			})

			taskRun := &pipelineapi.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-taskrun",
					Namespace: namespace,
				},
				Spec: pipelineapi.TaskRunSpec{
					TaskSpec: &pipelineapi.TaskSpec{
						Volumes: []corev1.Volume{
							{
								Name: "sec-vol",
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										SecretName: "my-secret",
									},
								},
							},
							{
								Name: "cm-vol",
								VolumeSource: corev1.VolumeSource{
									ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "my-configmap",
										},
									},
								},
							},
						},
					},
				},
			}

			err := resources.CheckTaskRunVolumesExist(ctx, client, taskRun)
			Expect(err).ToNot(HaveOccurred())
		})

		It("fails when referenced Secret is missing", func() {
			client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object, _ ...crc.GetOption) error {
				return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
			})

			taskRun := &pipelineapi.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-taskrun",
					Namespace: namespace,
				},
				Spec: pipelineapi.TaskRunSpec{
					TaskSpec: &pipelineapi.TaskSpec{
						Volumes: []corev1.Volume{
							{
								Name: "sec-vol",
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										SecretName: "missing-secret",
									},
								},
							},
						},
					},
				},
			}

			err := resources.CheckTaskRunVolumesExist(ctx, client, taskRun)
			Expect(err).To(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})
	})

	Context("when checking PipelineRun volumes", func() {
		It("succeeds when all volumes referenced across Pipeline tasks exist", func() {
			client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object, _ ...crc.GetOption) error {
				if nn.Name == "my-secret" || nn.Name == "my-configmap" {
					return nil
				}
				return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
			})

			pipelineRun := &pipelineapi.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-pipelinerun",
					Namespace: namespace,
				},
				Spec: pipelineapi.PipelineRunSpec{
					PipelineSpec: &pipelineapi.PipelineSpec{
						Tasks: []pipelineapi.PipelineTask{
							{
								Name: "task-1",
								TaskSpec: &pipelineapi.EmbeddedTask{
									TaskSpec: pipelineapi.TaskSpec{
										Volumes: []corev1.Volume{
											{
												Name: "sec-vol",
												VolumeSource: corev1.VolumeSource{
													Secret: &corev1.SecretVolumeSource{
														SecretName: "my-secret",
													},
												},
											},
										},
									},
								},
							},
							{
								Name: "task-2",
								TaskSpec: &pipelineapi.EmbeddedTask{
									TaskSpec: pipelineapi.TaskSpec{
										Volumes: []corev1.Volume{
											{
												Name: "cm-vol",
												VolumeSource: corev1.VolumeSource{
													ConfigMap: &corev1.ConfigMapVolumeSource{
														LocalObjectReference: corev1.LocalObjectReference{
															Name: "my-configmap",
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			err := resources.CheckPipelineRunVolumesExist(ctx, client, pipelineRun)
			Expect(err).ToNot(HaveOccurred())
		})

		It("fails when a Secret referenced in a Pipeline task is missing", func() {
			client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object, _ ...crc.GetOption) error {
				return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
			})

			pipelineRun := &pipelineapi.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-pipelinerun",
					Namespace: namespace,
				},
				Spec: pipelineapi.PipelineRunSpec{
					PipelineSpec: &pipelineapi.PipelineSpec{
						Tasks: []pipelineapi.PipelineTask{
							{
								Name: "task-1",
								TaskSpec: &pipelineapi.EmbeddedTask{
									TaskSpec: pipelineapi.TaskSpec{
										Volumes: []corev1.Volume{
											{
												Name: "sec-vol",
												VolumeSource: corev1.VolumeSource{
													Secret: &corev1.SecretVolumeSource{
														SecretName: "missing-secret",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			err := resources.CheckPipelineRunVolumesExist(ctx, client, pipelineRun)
			Expect(err).To(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})
	})
})
