// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/controller/fakes"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
	"github.com/shipwright-io/build/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/apis"
	crc "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Conditions", func() {

	var (
		ctl test.Catalog
	)

	Context("Operating on Conditions", func() {

		It("should be able to retrieve an existing condition message", func() {
			// BuildRun sample with an embedded condition of the type Succeeded
			br := ctl.BuildRunWithSucceededCondition()

			// BuildRun implements StatusConditions, therefore it can operate on
			// an existing Condition
			msg := br.Status.GetCondition(build.Succeeded).GetMessage()
			Expect(msg).To(Equal("foo is not bar"))
		})

		It("should be able to retrieve an existing condition reason", func() {
			// BuildRun sample with an embedded condition of the type Succeeded
			br := ctl.BuildRunWithSucceededCondition()

			reason := br.Status.GetCondition(build.Succeeded).GetReason()
			Expect(reason).To(Equal("foobar"))
		})

		It("should be able to retrieve an existing condition status", func() {
			// BuildRun sample with an embedded condition of the type Succeeded
			br := ctl.BuildRunWithSucceededCondition()

			status := br.Status.GetCondition(build.Succeeded).GetStatus()
			Expect(status).To(Equal(corev1.ConditionUnknown))
		})

		It("should return nil if a condition is not available when operating on it", func() {
			br := ctl.DefaultBuildRun("foo", "bar")

			// when getting a condition that does not exists on the BuildRun, do not
			// panic but rather return a nil
			cond := br.Status.GetCondition(build.Succeeded)
			Expect(cond).To(BeNil())
		})

		It("should be able to set a condition based on a type", func() {
			br := ctl.DefaultBuildRun("foo", "bar")
			// generate a condition of the type Succeeded
			tmpCond := &build.Condition{
				Type:               build.Succeeded,
				Status:             corev1.ConditionUnknown,
				Message:            "foobar",
				Reason:             "foo is bar",
				LastTransitionTime: metav1.Now(),
			}

			// set the condition on the BuildRun resource
			br.Status.SetCondition(tmpCond)

			condition := br.Status.GetCondition(build.Succeeded)
			Expect(condition).ToNot(BeNil())
			Expect(condition.Type).To(Equal(build.Succeeded))

			condMsg := br.Status.GetCondition(build.Succeeded).GetMessage()
			Expect(condMsg).To(Equal("foobar"))
		})

		It("should be able to update an existing condition based on a type", func() {
			// BuildRun sample with an embedded condition of the type Succeeded
			br := ctl.BuildRunWithSucceededCondition()

			reason := br.Status.GetCondition(build.Succeeded).GetReason()
			Expect(reason).To(Equal("foobar"))

			// generate a condition in order to update the existing one
			tmpCond := &build.Condition{
				Type:               build.Succeeded,
				Status:             corev1.ConditionUnknown,
				Message:            "foobar was updated",
				Reason:             "foo is bar",
				LastTransitionTime: metav1.Now(),
			}

			// update the condition on the BuildRun resource
			br.Status.SetCondition(tmpCond)

			condMsg := br.Status.GetCondition(build.Succeeded).GetMessage()
			Expect(condMsg).To(Equal("foobar was updated"))
		})

	})
	Context("Operating with TaskRun Conditions", func() {
		var (
			client *fakes.FakeClient
			ctl    test.Catalog
			br     *build.BuildRun
			tr     *v1beta1.TaskRun
		)

		tr = ctl.TaskRunWithStatus("foo", "bar")
		br = ctl.DefaultBuildRun("foo", "bar")
		client = &fakes.FakeClient{}

		It("updates BuildRun condition when TaskRun timeout", func() {

			fakeTRCondition := &apis.Condition{
				Type:    apis.ConditionSucceeded,
				Reason:  "TaskRunTimeout",
				Message: "not relevant",
			}

			Expect(resources.UpdateBuildRunUsingTaskRunCondition(
				context.TODO(),
				client,
				br,
				tr,
				fakeTRCondition,
			)).To(BeNil())
		})

		It("updates BuildRun condition when TaskRun fails and pod not found", func() {

			// stub a GET API call that fails with not found
			getClientStub := func(_ context.Context, nn types.NamespacedName, object crc.Object) error {
				switch object.(type) {
				case *corev1.Pod:
					return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
				}
				return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
			}
			// fake the calls with the above stub
			client.GetCalls(getClientStub)

			fakeTRCondition := &apis.Condition{
				Type:    apis.ConditionSucceeded,
				Reason:  "Failed",
				Message: "not relevant",
			}

			Expect(resources.UpdateBuildRunUsingTaskRunCondition(
				context.TODO(),
				client,
				br,
				tr,
				fakeTRCondition,
			)).To(BeNil())
		})

		It("updates a BuildRun condition when the related TaskRun fails and pod containers are available", func() {

			// generate a pod that have a single container and
			// one entry in the ContainerStatuses field, with
			// an exitCode
			taskRunGeneratedPod := corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foopod",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "foobar-container",
						},
					},
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name: "foobar-container",
							State: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason:   "foobar",
									ExitCode: 1,
								},
							},
						},
					},
				},
			}

			// stub a GET API call with taskRunGeneratedPod
			getClientStub := func(_ context.Context, nn types.NamespacedName, object crc.Object) error {
				switch object := object.(type) {
				case *corev1.Pod:
					taskRunGeneratedPod.DeepCopyInto(object)
					return nil
				}
				return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
			}

			// fake the calls with the above stub
			client.GetCalls(getClientStub)

			fakeTRCondition := &apis.Condition{
				Type:    apis.ConditionSucceeded,
				Reason:  "Failed",
				Message: "not relevant",
			}

			Expect(resources.UpdateBuildRunUsingTaskRunCondition(
				context.TODO(),
				client,
				br,
				tr,
				fakeTRCondition,
			)).To(BeNil())
		})

		It("updates BuildRun condition when TaskRun fails and pod is evicted", func() {
			// Generate a pod with the status to be evicted
			failedTaskRunEvictedPod := corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "evilpod",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "evilpod-container",
						},
					},
				},
				Status: corev1.PodStatus{
					Reason: "Evicted",
				},
			}

			// stub a GET API call with to pass the created pod
			getClientStub := func(_ context.Context, nn types.NamespacedName, object crc.Object) error {
				switch object := object.(type) {
				case *corev1.Pod:
					failedTaskRunEvictedPod.DeepCopyInto(object)
					return nil
				}
				return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
			}

			// fake the calls with the above stub
			client.GetCalls(getClientStub)

			// Now we need to create a fake failed taskrun so that it hits the code
			fakeTRCondition := &apis.Condition{
				Type:    apis.ConditionSucceeded,
				Reason:  "Failed",
				Message: "not relevant",
			}

			// We call the function with all the info
			Expect(resources.UpdateBuildRunUsingTaskRunCondition(
				context.TODO(),
				client,
				br,
				tr,
				fakeTRCondition,
			)).To(BeNil())

			// Finally, check the output of the buildRun
			Expect(br.Status.GetCondition(
				build.Succeeded).Reason,
			).To(Equal(build.BuildRunStatePodEvicted))
		})

		It("updates a BuildRun condition when the related TaskRun fails and pod containers are not available", func() {

			taskRunGeneratedPod := corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foobar",
				},
			}

			// stub a GET API call with the above taskRunGeneratedPod spec
			getClientStub := func(_ context.Context, nn types.NamespacedName, object crc.Object) error {
				switch object := object.(type) {
				case *corev1.Pod:
					taskRunGeneratedPod.DeepCopyInto(object)
					return nil
				}
				return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
			}
			// fake the calls with the above stub
			client.GetCalls(getClientStub)

			fakeTRCondition := &apis.Condition{
				Type:    apis.ConditionSucceeded,
				Reason:  "Failed",
				Message: "not relevant",
			}

			Expect(resources.UpdateBuildRunUsingTaskRunCondition(
				context.TODO(),
				client,
				br,
				tr,
				fakeTRCondition,
			)).To(BeNil())
		})
	})
})
