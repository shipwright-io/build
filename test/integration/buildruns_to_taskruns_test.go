// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
	"github.com/shipwright-io/build/test"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var _ = Describe("Integration tests BuildRuns and TaskRuns", func() {
	var (
		cbsObject      *v1alpha1.ClusterBuildStrategy
		buildObject    *v1alpha1.Build
		buildRunObject *v1alpha1.BuildRun
		buildSample    []byte
		buildRunSample []byte
	)

	var setupBuildAndBuildRun = func(buildDef []byte, buildRunDef []byte, strategy ...string) (*v1alpha1.Build, *v1alpha1.BuildRun) {

		var strategyName = STRATEGY + tb.Namespace
		if len(strategy) > 0 {
			strategyName = strategy[0]
		}

		buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(BUILD+tb.Namespace, strategyName, buildDef)
		Expect(err).To(BeNil())
		Expect(tb.CreateBuild(buildObject)).To(BeNil())

		buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
		Expect(err).To(BeNil())

		buildRunObject, err = tb.Catalog.LoadBRWithNameAndRef(BUILDRUN+tb.Namespace, BUILD+tb.Namespace, buildRunDef)
		Expect(err).To(BeNil())
		Expect(tb.CreateBR(buildRunObject)).To(BeNil())

		//TODO: consider how to deal with buildObject or buildRunObject
		return buildObject, buildRunObject
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
			It("eventually reports Running as the reason", func() {
				_, buildRun := setupBuildAndBuildRun([]byte(test.BuildCBSMinimal), []byte(test.MinimalBuildRun))
				fmt.Fprintf(GinkgoWriter, "Waiting for BuildRun to have success status %s and reason %s", corev1.ConditionUnknown, "Running")
				err := wait.PollImmediate(tb.Interval, tb.TimeOut, func() (bool, error) {
					updated, err := tb.GetBR(buildRun.Name)
					if err != nil {
						return false, err
					}
					succeeded := updated.Status.GetCondition(v1alpha1.Succeeded)
					if succeeded == nil {
						return false, nil
					}
					if succeeded.Status != corev1.ConditionUnknown {
						fmt.Fprintf(GinkgoWriter, "BuildRun success status is %s", succeeded.Status)
						return true, nil
					}
					if succeeded.Reason != "Running" {
						fmt.Fprintf(GinkgoWriter, "BuildRun success reason is %s", succeeded.Reason)
						return false, nil
					}
					return true, nil
				})
				// Error will either be a failure, or a timeout waiting for the "Running" reason to appear
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when condition status is false", func() {
			It("reflects a timeout", func() {
				_, buildRun := setupBuildAndBuildRun([]byte(test.BuildCBSWithShortTimeOut), []byte(test.MinimalBuildRun))
				fmt.Fprintf(GinkgoWriter, "Waiting for BuildRun to have success status %s", corev1.ConditionFalse)
				err := wait.PollImmediate(tb.Interval, tb.TimeOut, func() (bool, error) {
					buildRun, err := tb.GetBR(buildRun.Name)
					if err != nil {
						return false, err
					}
					succeeded := buildRun.Status.GetCondition(v1alpha1.Succeeded)
					if succeeded == nil {
						return false, nil
					}
					fmt.Fprintf(GinkgoWriter, "BuildRun success status is %s", succeeded.Status)
					return succeeded.Status == corev1.ConditionFalse, nil
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(buildRun).NotTo(BeNil())
				Expect(buildRun.Status.GetCondition(v1alpha1.Succeeded)).NotTo(BeNil())
				Expect(buildRun.Status.GetCondition(v1alpha1.Succeeded).Status).To(Equal(corev1.ConditionFalse))
				Expect(buildRun.Status.GetCondition(v1alpha1.Succeeded).Reason).To(Equal("BuildRunTimeout"))
			})

			It("reflects a failed reason", func() {
				WithCustomClusterBuildStrategy([]byte(test.ClusterBuildStrategySingleStepKanikoError), func() {
					_, buildRun := setupBuildAndBuildRun([]byte(test.BuildCBSMinimal), []byte(test.MinimalBuildRun), STRATEGY+tb.Namespace+"custom")
					fmt.Fprintf(GinkgoWriter, "Waiting for BuildRun to have success status %s", corev1.ConditionFalse)
					err := wait.PollImmediate(tb.Interval, tb.TimeOut, func() (bool, error) {
						buildRun, err := tb.GetBR(buildRun.Name)
						if err != nil {
							return false, err
						}
						succeeded := buildRun.Status.GetCondition(v1alpha1.Succeeded)
						if succeeded == nil {
							return false, nil
						}
						fmt.Fprintf(GinkgoWriter, "BuildRun success status is %s", succeeded.Status)
						return succeeded.Status == corev1.ConditionFalse, nil
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(buildRun.Status.CompletionTime).ToNot(BeNil())

					taskRun, err := tb.GetTaskRunFromBuildRun(buildRun.Name)
					Expect(err).NotTo(HaveOccurred())

					Expect(buildRun.Status.FailedAt.Pod).To(Equal(taskRun.Status.PodName))
					Expect(buildRun.Status.FailedAt.Container).To(Equal("step-" + "step-build-and-push"))

					succeeded := buildRun.Status.GetCondition(v1alpha1.Succeeded)
					Expect(succeeded).NotTo(BeNil())
					Expect(succeeded.Status).To(Equal(corev1.ConditionFalse))
					Expect(succeeded.Reason).To(Equal("Failed"))
					Expect(succeeded.Message).To(ContainSubstring("buildrun step %s failed in pod %s", "step-step-build-and-push", taskRun.Status.PodName))
				})
			})
		})

		Context("when condition status true", func() {
			It("should reflect the taskrun succeeded reason in the buildrun condition", func() {
				WithCustomClusterBuildStrategy([]byte(test.ClusterBuildStrategyNoOp), func() {
					_, buildRun := setupBuildAndBuildRun([]byte(test.BuildCBSMinimal), []byte(test.MinimalBuildRun), STRATEGY+tb.Namespace+"custom")
					fmt.Fprintf(GinkgoWriter, "Waiting for BuildRun to have success status %s", corev1.ConditionTrue)
					err := wait.PollImmediate(tb.Interval, tb.TimeOut, func() (bool, error) {
						buildRun, err := tb.GetBR(buildRun.Name)
						if err != nil {
							return false, err
						}
						succeeded := buildRun.Status.GetCondition(v1alpha1.Succeeded)
						if succeeded == nil {
							return false, nil
						}
						return succeeded.Status == corev1.ConditionTrue, nil
					})
					Expect(err).NotTo(HaveOccurred())
					succeeded := buildRun.Status.GetCondition(v1alpha1.Succeeded)
					Expect(succeeded).NotTo(BeNil())
					Expect(succeeded.Status).To(Equal(corev1.ConditionTrue))
					Expect(succeeded.Reason).To(Equal("Succeeded"))
					Expect(succeeded.Message).To(ContainSubstring("All Steps have completed executing"))
				})
			})
		})
	})

	Context("when a buildrun is created", func() {
		It("should reflect a Pending and Running reason", func() {
			// use a custom strategy here that just sleeps 30 seconds to prevent it from completing too fast so that we do not get the Running state
			WithCustomClusterBuildStrategy([]byte(test.ClusterBuildStrategySleep30s), func() {
				_, buildRunObject := setupBuildAndBuildRun([]byte(test.BuildCBSMinimal), []byte(test.MinimalBuildRun), STRATEGY+tb.Namespace+"custom")

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

			_, err = tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())

			expectedReason = "BuildRunTimeout"
			actualReason, err = tb.GetBRTillDesiredReason(buildRunObject.Name, expectedReason)
			Expect(err).To(BeNil(), fmt.Sprintf("failed to get desired reason; expected %s, got %s", expectedReason, actualReason))

			tr, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
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
			Expect(reason).To(Equal("BuildRegistrationFailed"))

			_, err = tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).ToNot(BeNil())

		})
	})

	Context("when a buildrun is created and the taskrun is cancelled", func() {

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

			_, err = tb.UpdateTaskRun(tr)
			Expect(err).To(BeNil())

			expectedReason := "TaskRunCancelled"
			actualReason, err := tb.GetTRTillDesiredReason(buildRunObject.Name, expectedReason)
			Expect(err).To(BeNil(), fmt.Sprintf("failed to get desired reason; expected %s, got %s", expectedReason, actualReason))
		})
	})

	Context("when a buildrun is created and the buildrun is cancelled", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSMinimal)
			buildRunSample = []byte(test.MinimalBuildRun)
		})

		It("should reflect a TaskRunCancelled reason in the taskrun, BuildRunCanceled in the buildrun, and no completionTime", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			err := wait.PollImmediate(1*time.Second, 4*time.Second, func() (done bool, err error) {
				bro, err := tb.GetBRTillStartTime(buildRunObject.Name)
				if err != nil {
					GinkgoT().Logf("error on br get: %s\n", err.Error())
					return false, nil
				}

				bro.Spec.State = v1alpha1.BuildRunStateCancel
				err = tb.UpdateBR(bro)
				if err != nil {
					GinkgoT().Logf("error on br update: %s\n", err.Error())
					return false, nil
				}
				return true, nil
			})
			Expect(err).To(BeNil())

			expectedReason := "TaskRunCancelled"
			actualReason, err := tb.GetTRTillDesiredReason(buildRunObject.Name, expectedReason)
			Expect(err).To(BeNil(), fmt.Sprintf("failed to get desired TaskRun reason; expected %s, got %s", expectedReason, actualReason))

			expectedReason = v1alpha1.BuildRunStateCancel
			actualReason, err = tb.GetBRTillDesiredReason(buildRunObject.Name, expectedReason)
			Expect(err).To(BeNil(), fmt.Sprintf("failed to get desired BuildRun reason; expected %s, got %s", expectedReason, actualReason))
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

			expectedReason := "TaskRunIsMissing"
			actualReason, err := tb.GetBRTillDesiredReason(buildRunObject.Name, expectedReason)
			Expect(err).To(BeNil(), fmt.Sprintf("failed to get desired reason; expected %s, got %s", expectedReason, actualReason))
		})
	})

	Context("when a buildrun is created and the taskrun deleted after successful completion", func() {

		It("should reflect a Success reason", func() {
			WithCustomClusterBuildStrategy([]byte(test.ClusterBuildStrategyNoOp), func() {
				_, buildRunObject := setupBuildAndBuildRun([]byte(test.BuildCBSMinimal), []byte(test.MinimalBuildRun), STRATEGY+tb.Namespace+"custom")

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

	Context("when a buildrun is created with invalid name", func() {
		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSMinimal)
			buildRunSample = []byte(test.MinimalBuildRun)
		})

		It("fails the buildrun with a proper error in Reason", func() {
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			// Set buildrun name more than 63 characters
			buildRunObject.Name = strings.Repeat("s", 64)
			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			br, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())

			condition := br.Status.GetCondition(v1alpha1.Succeeded)
			Expect(condition.Status).To(Equal(corev1.ConditionFalse))
			Expect(condition.Reason).To(Equal(resources.BuildRunNameInvalid))
			Expect(condition.Message).To(Equal("must be no more than 63 characters"))
		})

		It("should reflect a BadRequest reason in TaskRun", func() {
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			// Set buildrun name more than 63 characters
			buildRunObject.Name = strings.Repeat("s", 64)
			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			_, err = tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())

			expectedReason := "BadRequest"
			actualReason, err := tb.GetTRTillDesiredReason(buildRunObject.Name, expectedReason)
			Expect(err).To(HaveOccurred(), fmt.Sprintf("failed to get desired reason; expected %s, got %s", expectedReason, actualReason))

			_, err = tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(HaveOccurred())
		})
	})
})
