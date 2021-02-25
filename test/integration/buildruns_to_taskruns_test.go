// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

var _ = Describe("Integration tests BuildRuns and TaskRuns", func() {
	var (
		cbsObject      *v1alpha1.ClusterBuildStrategy
		buildObject    *v1alpha1.Build
		buildRunObject *v1alpha1.BuildRun
		buildSample    []byte
		buildRunSample []byte
	)

	var setupBuildAndBuildRun = func(buildDef []byte, buildRunDef []byte, strategy ...string) (watch.Interface, *v1alpha1.Build, *v1alpha1.BuildRun) {

		var strategyName = STRATEGY + tb.Namespace
		if len(strategy) > 0 {
			strategyName = strategy[0]
		}

		buildRunWitcher, err := tb.BuildClientSet.BuildV1alpha1().BuildRuns(tb.Namespace).Watch(context.TODO(), metav1.ListOptions{})
		Expect(err).To(BeNil())

		buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(BUILD+tb.Namespace, strategyName, buildDef)
		Expect(err).To(BeNil())
		Expect(tb.CreateBuild(buildObject)).To(BeNil())

		buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
		Expect(err).To(BeNil())

		buildRunObject, err = tb.Catalog.LoadBRWithNameAndRef(BUILDRUN+tb.Namespace, BUILD+tb.Namespace, buildRunDef)
		Expect(err).To(BeNil())
		Expect(tb.CreateBR(buildRunObject)).To(BeNil())

		//TODO: consider how to deal with buildObject or buildRunObject
		return buildRunWitcher, buildObject, buildRunObject
	}

	var WithCustomClusterBuildStrategy = func(data []byte, f func()) {
		customClusterBuildStrategy, err := tb.Catalog.LoadCBSWithName(STRATEGY+tb.Namespace+"custom", data)
		Expect(err).To(BeNil())

		Expect(tb.CreateClusterBuildStrategy(customClusterBuildStrategy)).To(BeNil())
		f()
		Expect(tb.DeleteClusterBuildStrategy(customClusterBuildStrategy.Name)).To(BeNil())
	}

	// Load the ClusterBuildStrategies before each test case
	BeforeEach(func() {
		cbsObject, err = tb.Catalog.LoadCBSWithName(STRATEGY+tb.Namespace, []byte(test.ClusterBuildStrategySingleStepKaniko))
		Expect(err).To(BeNil())

		err = tb.CreateClusterBuildStrategy(cbsObject)
		Expect(err).To(BeNil())
	})

	// Delete the ClusterBuildStrategies after each test case
	AfterEach(func() {

		_, err = tb.GetBuild(buildObject.Name)
		if err == nil {
			Expect(tb.DeleteBuild(buildObject.Name)).To(BeNil())
		}

		err := tb.DeleteClusterBuildStrategy(cbsObject.Name)
		Expect(err).To(BeNil())
	})

	// Override the Builds and BuildRuns CRDs instances to use
	// before an It() statement is executed
	JustBeforeEach(func() {
		if buildSample != nil {
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(BUILD+tb.Namespace, STRATEGY+tb.Namespace, buildSample)
			Expect(err).To(BeNil())
		}

		if buildRunSample != nil {
			buildRunObject, err = tb.Catalog.LoadBRWithNameAndRef(BUILDRUN+tb.Namespace, BUILD+tb.Namespace, buildRunSample)
			Expect(err).To(BeNil())
		}
	})

	Context("when buildrun uses conditions", func() {
		Context("when condition status unknown", func() {
			It("reflects a change from pending to running reason", func() {
				buildRunWitcher, _, _ := setupBuildAndBuildRun([]byte(test.BuildCBSMinimal), []byte(test.MinimalBuildRun))

				// use a fakeTime to simplify tests
				fakeTime := time.Date(1989, 05, 15, 00, 01, 01, 651387237, time.UTC)

				var timeout = time.After(tb.TimeOut)
				go func() {
					<-timeout
					buildRunWitcher.Stop()
				}()

				var seq = []*v1alpha1.Condition{}
				for event := range buildRunWitcher.ResultChan() {
					condition := event.Object.(*v1alpha1.BuildRun).Status.GetCondition(v1alpha1.Succeeded)
					if condition != nil {
						condition.LastTransitionTime = metav1.Time{Time: fakeTime}
						seq = append(seq, condition)
					}

					// Pending -> Running
					if condition != nil && condition.Reason == "Running" {
						buildRunWitcher.Stop()
					}
				}
				// consider a longer sequence, for events where the cluster have
				// insufficient resources, where the Reason will be ExceededNodeResources.
				Expect(len(seq) >= 2).To(Equal(true))

				// ensure the conditions move eventually from unknown into running
				Expect(seq).Should(ContainElement(&v1alpha1.Condition{
					Type:               v1alpha1.Succeeded,
					Status:             corev1.ConditionUnknown,
					LastTransitionTime: metav1.Time{Time: fakeTime},
					Reason:             "Pending",
					Message:            "Pending",
				}))
				Expect(seq).Should(ContainElement(&v1alpha1.Condition{
					Type:               v1alpha1.Succeeded,
					Status:             corev1.ConditionUnknown,
					LastTransitionTime: metav1.Time{Time: fakeTime},
					Reason:             "Running",
					Message:            "Not all Steps in the Task have finished executing",
				}))
			})
		})

		Context("when condition status is false", func() {
			It("reflects a timeout", func() {
				buildRunWitcher, build, buildRun := setupBuildAndBuildRun([]byte(test.BuildCBSWithShortTimeOut), []byte(test.MinimalBuildRun))

				var timeout = time.After(tb.TimeOut)
				go func() {
					<-timeout
					buildRunWitcher.Stop()
				}()

				var seq = []*v1alpha1.Condition{}
				for event := range buildRunWitcher.ResultChan() {
					condition := event.Object.(*v1alpha1.BuildRun).Status.GetCondition(v1alpha1.Succeeded)
					if condition != nil {
						seq = append(seq, condition)
					}

					// Pending -> Running
					if condition != nil && condition.Status == corev1.ConditionFalse {
						buildRunWitcher.Stop()
					}
				}

				lastIdx := len(seq) - 1
				Expect(lastIdx).To(BeNumerically(">", 0))
				Expect(seq[lastIdx].Type).To(Equal(v1alpha1.Succeeded))
				Expect(seq[lastIdx].Status).To(Equal(corev1.ConditionFalse))
				Expect(seq[lastIdx].Reason).To(Equal("BuildRunTimeout"))
				Expect(seq[lastIdx].Message).To(Equal(fmt.Sprintf("BuildRun %s failed to finish within %v", buildRun.Name, build.Spec.Timeout.Duration)))
			})

			It("reflects a failed reason", func() {
				WithCustomClusterBuildStrategy([]byte(test.ClusterBuildStrategySingleStepKanikoError), func() {
					buildRunWitcher, _, buildRun := setupBuildAndBuildRun([]byte(test.BuildCBSMinimal), []byte(test.MinimalBuildRun), STRATEGY+tb.Namespace+"custom")

					var timeout = time.After(tb.TimeOut)
					go func() {
						<-timeout
						buildRunWitcher.Stop()
					}()

					var seq = []*v1alpha1.Condition{}
					for event := range buildRunWitcher.ResultChan() {
						condition := event.Object.(*v1alpha1.BuildRun).Status.GetCondition(v1alpha1.Succeeded)
						if condition != nil {
							seq = append(seq, condition)
						}

						if condition != nil && condition.Status == corev1.ConditionFalse {
							buildRunWitcher.Stop()
						}
					}

					buildRun, err = tb.GetBR(buildRun.Name)
					Expect(err).ToNot(HaveOccurred())
					Expect(buildRun.Status.CompletionTime).ToNot(BeNil())

					taskRun, err := tb.GetTaskRunFromBuildRun(buildRun.Name)
					Expect(err).ToNot(HaveOccurred())

					Expect(buildRun.Status.FailedAt.Pod).To(Equal(taskRun.Status.PodName))
					Expect(buildRun.Status.FailedAt.Container).To(Equal("step-" + "step-build-and-push"))

					lastIdx := len(seq) - 1
					Expect(lastIdx).To(BeNumerically(">", 0))
					Expect(seq[lastIdx].Type).To(Equal(v1alpha1.Succeeded))
					Expect(seq[lastIdx].Status).To(Equal(corev1.ConditionFalse))
					Expect(seq[lastIdx].Reason).To(Equal("Failed"))
					Expect(seq[lastIdx].Message).To(ContainSubstring("buildrun step failed in pod %s", taskRun.Status.PodName))
				})
			})
		})

		Context("when condition status true", func() {
			It("should reflect the taskrun succeeded reason in the buildrun condition", func() {
				WithCustomClusterBuildStrategy([]byte(test.ClusterBuildStrategyNoOp), func() {
					buildRunWitcher, _, _ := setupBuildAndBuildRun([]byte(test.BuildCBSMinimal), []byte(test.MinimalBuildRun), STRATEGY+tb.Namespace+"custom")

					var timeout = time.After(tb.TimeOut)
					go func() {
						<-timeout
						buildRunWitcher.Stop()
					}()

					var seq = []*v1alpha1.Condition{}
					for event := range buildRunWitcher.ResultChan() {
						condition := event.Object.(*v1alpha1.BuildRun).Status.GetCondition(v1alpha1.Succeeded)
						if condition != nil {
							seq = append(seq, condition)
						}

						if condition != nil && condition.Status == corev1.ConditionTrue {
							buildRunWitcher.Stop()
						}
					}

					lastIdx := len(seq) - 1
					Expect(lastIdx).To(BeNumerically(">", 0))
					Expect(seq[lastIdx].Type).To(Equal(v1alpha1.Succeeded))
					Expect(seq[lastIdx].Status).To(Equal(corev1.ConditionTrue))
					Expect(seq[lastIdx].Reason).To(Equal("Succeeded"))
					Expect(seq[lastIdx].Message).To(ContainSubstring("All Steps have completed executing"))
				})
			})
		})
	})

	Context("when a buildrun is created", func() {
		It("should reflect a Pending and Running reason", func() {
			// use a custom strategy here that just sleeps 30 seconds to prevent it from completing too fast so that we do not get the Running state
			WithCustomClusterBuildStrategy([]byte(test.ClusterBuildStrategySleep30s), func() {
				_, _, buildRunObject := setupBuildAndBuildRun([]byte(test.BuildCBSMinimal), []byte(test.MinimalBuildRun), STRATEGY+tb.Namespace+"custom")

				_, err = tb.GetBRTillStartTime(buildRunObject.Name)
				Expect(err).To(BeNil())

				// Pending is an intermediate state where a certain amount of luck is needed to capture it with a polling interval of 3s.
				// Also, if the build-operator is not reconciling on this TaskRun status quick enough, a BuildRun might never be in Pending
				// but rather directly go to Running.
				/*
					expectedReason := "Pending"
					actualReason, err := tb.GetTRTillDesiredReason(buildRunObject.Name, expectedReason)
					Expect(err).To(BeNil(), fmt.Sprintf("failed to get desired reason; expected %s, got %s", expectedReason, actualReason))

					expectedReason = "Pending"
					actualReason, err = tb.GetBRTillDesiredReason(buildRunObject.Name, expectedReason)
					Expect(err).To(BeNil(), fmt.Sprintf("failed to get desired reason; expected %s, got %s", expectedReason, actualReason))
				*/

				expectedReason := "Running"
				actualReason, err := tb.GetTRTillDesiredReason(buildRunObject.Name, expectedReason)
				Expect(err).To(BeNil(), fmt.Sprintf("failed to get desired reason; expected %s, got %s", expectedReason, actualReason))

				expectedReason = "Running"
				actualReason, err = tb.GetBRTillDesiredReason(buildRunObject.Name, expectedReason)
				Expect(err).To(BeNil(), fmt.Sprintf("failed to get desired reason; expected %s, got %s", expectedReason, actualReason))
			})
		})
	})

	Context("when a buildrun is created but fails because of a timeout", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSWithShortTimeOut)
			buildRunSample = []byte(test.MinimalBuildRun)
		})

		It("should reflect a TaskRunTimeout reason and Completion time on timeout", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			_, err = tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())

			expectedReason := "TaskRunTimeout"
			actualReason, err := tb.GetTRTillDesiredReason(buildRunObject.Name, expectedReason)
			Expect(err).To(BeNil(), fmt.Sprintf("failed to get desired reason; expected %s, got %s", expectedReason, actualReason))

			tr, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())

			expectedReason = fmt.Sprintf("TaskRun \"%s\" failed to finish within \"5s\"", tr.Name)
			actualReason, err = tb.GetBRTillDesiredReason(buildRunObject.Name, expectedReason)
			Expect(err).To(BeNil(), fmt.Sprintf("failed to get desired reason; expected %s, got %s", expectedReason, actualReason))

			tr, err = tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(tr.Status.CompletionTime).ToNot(BeNil())
		})
	})

	Context("when a buildrun is created with a wrong url", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSWithWrongURL)
			buildRunSample = []byte(test.MinimalBuildRun)
		})

		It("should reflect a Failed reason and Completion on failure", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			b, err := tb.GetBuildTillRegistration(buildObject.Name, corev1.ConditionFalse)
			Expect(err).To(BeNil())
			Expect(b.Status.Registered).To(Equal(corev1.ConditionFalse))
			Expect(b.Status.Reason).To(Equal(v1alpha1.RemoteRepositoryUnreachable))
			Expect(b.Status.Message).To(ContainSubstring("no such host"))

			_, err = tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())

			reason, err := tb.GetBRReason(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(reason).To(ContainSubstring("the Build is not registered correctly"))

			_, err = tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).ToNot(BeNil())

		})
	})

	Context("when a buildrun is created and cancelled", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSMinimal)
			buildRunSample = []byte(test.MinimalBuildRun)
		})

		It("should reflect a TaskRunCancelled reason and no completionTime", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			_, err = tb.GetBRTillStartTime(buildRunObject.Name)
			Expect(err).To(BeNil())

			tr, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())

			tr.Spec.Status = "TaskRunCancelled"

			tr, err = tb.UpdateTaskRun(tr)
			Expect(err).To(BeNil())

			expectedReason := fmt.Sprintf("TaskRun \"%s\" was cancelled", tr.Name)
			actualReason, err := tb.GetBRTillDesiredReason(buildRunObject.Name, expectedReason)
			Expect(err).To(BeNil(), fmt.Sprintf("failed to get desired reason; expected %s, got %s", expectedReason, actualReason))

			expectedReason = "TaskRunCancelled"
			actualReason, err = tb.GetTRTillDesiredReason(buildRunObject.Name, expectedReason)
			Expect(err).To(BeNil(), fmt.Sprintf("failed to get desired reason; expected %s, got %s", expectedReason, actualReason))
		})
	})

	Context("when a buildrun is created and the taskrun deleted before completion", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSMinimal)
			buildRunSample = []byte(test.MinimalBuildRun)
		})

		It("should reflect a Failed reason", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			_, err = tb.GetBRTillStartTime(buildRunObject.Name)
			Expect(err).To(BeNil())

			tr, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())

			tb.DeleteTR(tr.Name)

			expectedReason := fmt.Sprintf("taskRun %s doesn't exist", tr.Name)
			actualReason, err := tb.GetBRTillDesiredReason(buildRunObject.Name, expectedReason)
			Expect(err).To(BeNil(), fmt.Sprintf("failed to get desired reason; expected %s, got %s", expectedReason, actualReason))
		})
	})

	Context("when a buildrun is created and the taskrun deleted after successful completion", func() {

		It("should reflect a Success reason", func() {
			WithCustomClusterBuildStrategy([]byte(test.ClusterBuildStrategyNoOp), func() {
				_, _, buildRunObject := setupBuildAndBuildRun([]byte(test.BuildCBSMinimal), []byte(test.MinimalBuildRun), STRATEGY+tb.Namespace+"custom")

				_, err = tb.GetBRTillCompletion(buildRunObject.Name)
				Expect(err).To(BeNil())

				reason, err := tb.GetBRReason(buildRunObject.Name)
				Expect(err).To(BeNil())
				Expect(reason).To(Equal("Succeeded"))

				tr, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
				Expect(err).To(BeNil())
				Expect(tr.Status.CompletionTime).NotTo(BeNil())

				tb.DeleteTR(tr.Name)

				// in a test case, it is hard to verify that something (marking the BuildRun failed) is not happening, we quickly check the TaskRun is gone and
				// check one more time that the BuildRun is still Succeeded
				_, err = tb.GetTaskRunFromBuildRun(buildRunObject.Name)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("failed to find an owned TaskRun"))

				reason, err = tb.GetBRReason(buildRunObject.Name)
				Expect(err).To(BeNil())
				Expect(reason).To(Equal("Succeeded"))
			})
		})
	})
})
