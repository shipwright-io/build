// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package runner

import (
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/result"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	test "github.com/shipwright-io/build/test/v1beta1_samples"
)

var _ = Describe("TaskRun SyncBuildRunStatus", func() {

	var (
		buildRun  *v1beta1.BuildRun
		runner    *TaskRunBuildRunner
		k8sClient client.Client
	)

	BeforeEach(func() {
		catalog := &test.Catalog{}
		buildRun = catalog.DefaultBuildRun("test-build-run", "test-build")
		taskRun := catalog.DefaultTaskRun("test-build-run-t9sk", "test-build")
		runner = &TaskRunBuildRunner{
			TaskRun: taskRun,
		}
		k8sClient = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
			buildRun, taskRun).Build()
	})

	When("the build fetches source from Git", func() {

		It("should report the git details if the clone succeeds", func(ctx SpecContext) {
			commitSha := "0e0583421a5e4bf562ffe33f3651e16ba0c78591"
			buildRun.Status.BuildSpec = &v1beta1.BuildSpec{
				Source: &v1beta1.Source{
					Type: v1beta1.GitType,
					Git: &v1beta1.Git{
						URL: "https://github.com/shipwright-io/sample-go",
					},
				},
			}
			runner.Status.Results = append(runner.Status.Results,
				pipelinev1.TaskRunResult{
					Name: "shp-source-default-commit-sha",
					Value: pipelinev1.ParamValue{
						Type:      pipelinev1.ParamTypeString,
						StringVal: commitSha,
					},
				},
				pipelinev1.TaskRunResult{
					Name: "shp-source-default-commit-author",
					Value: pipelinev1.ParamValue{
						Type:      pipelinev1.ParamTypeString,
						StringVal: "foo bar",
					},
				})

			runner.SyncBuildRunStatus(ctx, k8sClient, buildRun)

			Expect(buildRun.Status.Source).ToNot(BeNil(), "source status")
			Expect(buildRun.Status.Source.Git.CommitSha).To(Equal(commitSha), "source status git commitSha")
			Expect(buildRun.Status.Source.Git.CommitAuthor).To(Equal("foo bar"), "source status git commit author")
		})

		PIt("should report an error if the clone fails")

		PIt("should report an error if the clone times out")
	})

	When("the build fetches source from an OCI artifact", func() {

		It("should report the OCI artifact details if it succeeds", func(ctx SpecContext) {
			bundleImageDigest := "sha256:fe1b73cd25ac3f11dec752755e2"
			buildRun.Status.BuildSpec = &v1beta1.BuildSpec{
				Source: &v1beta1.Source{
					Type: v1beta1.OCIArtifactType,
					OCIArtifact: &v1beta1.OCIArtifact{
						Image: "ghcr.io/shipwright-io/sample-go/source-bundle:latest",
					},
				},
			}

			runner.Status.Results = append(runner.Status.Results,
				pipelinev1.TaskRunResult{
					Name: "shp-source-default-image-digest",
					Value: pipelinev1.ParamValue{
						Type:      pipelinev1.ParamTypeString,
						StringVal: bundleImageDigest,
					},
				})

			runner.SyncBuildRunStatus(ctx, k8sClient, buildRun)

			Expect(buildRun.Status.Source).ToNot(BeNil())
			Expect(buildRun.Status.Source.OciArtifact.Digest).To(Equal(bundleImageDigest))
		})

		PIt("should report an error if the OCI artifact pull fails")

		PIt("should report an error if the OCI artifact pull times out")
	})

	When("the build pushes a container image", func() {

		It("should report the source details from git and output image", func(ctx SpecContext) {
			commitSha := "0e0583421a5e4bf562ffe33f3651e16ba0c78591"
			imageDigest := "sha256:fe1b73cd25ac3f11dec752755e2"
			buildRun.Status.BuildSpec = &v1beta1.BuildSpec{
				Source: &v1beta1.Source{
					Type: v1beta1.GitType,
					Git: &v1beta1.Git{
						URL: "https://github.com/shipwright-io/sample-go",
					},
				},
			}

			runner.Status.Results = append(runner.Status.Results,
				pipelinev1.TaskRunResult{
					Name: "shp-source-default-commit-sha",
					Value: pipelinev1.ParamValue{
						Type:      pipelinev1.ParamTypeString,
						StringVal: commitSha,
					},
				},
				pipelinev1.TaskRunResult{
					Name: "shp-source-default-commit-author",
					Value: pipelinev1.ParamValue{
						Type:      pipelinev1.ParamTypeString,
						StringVal: "foo bar",
					},
				},
				pipelinev1.TaskRunResult{
					Name: "shp-image-digest",
					Value: pipelinev1.ParamValue{
						Type:      pipelinev1.ParamTypeString,
						StringVal: imageDigest,
					},
				},
				pipelinev1.TaskRunResult{
					Name: "shp-image-size",
					Value: pipelinev1.ParamValue{
						Type:      pipelinev1.ParamTypeString,
						StringVal: "230",
					},
				})

			runner.SyncBuildRunStatus(ctx, k8sClient, buildRun)

			Expect(buildRun.Status.Source).ToNot(BeNil())
			Expect(buildRun.Status.Source.Git).ToNot(BeNil())
			Expect(buildRun.Status.Source.Git.CommitSha).To(Equal(commitSha))
			Expect(buildRun.Status.Source.Git.CommitAuthor).To(Equal("foo bar"))
			Expect(buildRun.Status.Output.Digest).To(Equal(imageDigest))
			Expect(buildRun.Status.Output.Size).To(Equal(int64(230)))

		})

		PIt("should report the source details from an OCI artifact as well as output image")

		It("should report image vulnerabilities if a scan was run", func(ctx SpecContext) {
			imageDigest := "sha256:fe1b73cd25ac3f11dec752755e2"
			runner.Status.Results = append(runner.Status.Results,
				pipelinev1.TaskRunResult{
					Name: "shp-image-digest",
					Value: pipelinev1.ParamValue{
						Type:      pipelinev1.ParamTypeString,
						StringVal: imageDigest,
					},
				},
				pipelinev1.TaskRunResult{
					Name: "shp-image-size",
					Value: pipelinev1.ParamValue{
						Type:      pipelinev1.ParamTypeString,
						StringVal: "230",
					},
				},
				pipelinev1.TaskRunResult{
					Name: "shp-image-vulnerabilities",
					Value: pipelinev1.ParamValue{
						Type:      pipelinev1.ParamTypeString,
						StringVal: "CVE-2019-12900:c,CVE-2019-8457:h",
					},
				})

			runner.SyncBuildRunStatus(ctx, k8sClient, buildRun)

			Expect(buildRun.Status.Output.Digest).To(Equal(imageDigest))
			Expect(buildRun.Status.Output.Size).To(Equal(int64(230)))
			Expect(buildRun.Status.Output.Vulnerabilities).To(HaveLen(2))
			Expect(buildRun.Status.Output.Vulnerabilities[0].ID).To(Equal("CVE-2019-12900"))
			Expect(buildRun.Status.Output.Vulnerabilities[0].Severity).To(Equal(v1beta1.Critical))

		})

		It("should not report image vulnerabilities if a scan was not run", func(ctx SpecContext) {
			imageDigest := "sha256:fe1b73cd25ac3f11dec752755e2"
			runner.Status.Results = append(runner.Status.Results,
				pipelinev1.TaskRunResult{
					Name: "shp-image-digest",
					Value: pipelinev1.ParamValue{
						Type:      pipelinev1.ParamTypeString,
						StringVal: imageDigest,
					},
				},
				pipelinev1.TaskRunResult{
					Name: "shp-image-size",
					Value: pipelinev1.ParamValue{
						Type:      pipelinev1.ParamTypeString,
						StringVal: "230",
					},
				},
				pipelinev1.TaskRunResult{
					Name: "shp-image-vulnerabilities",
					Value: pipelinev1.ParamValue{
						Type:      pipelinev1.ParamTypeString,
						StringVal: "",
					},
				})

			runner.SyncBuildRunStatus(ctx, k8sClient, buildRun)

			Expect(buildRun.Status.Output.Digest).To(Equal(imageDigest))
			Expect(buildRun.Status.Output.Size).To(Equal(int64(230)))
			Expect(buildRun.Status.Output.Vulnerabilities).To(HaveLen(0))
		})
	})

	When("the build has started", func() {

		Context("and the TaskRun hasn't started yet", func() {
			// In this scenario, the TaskRun may not even have a "Succeeded" condition
			// This is perhaps an edge case that may not be found in production environments

			var readyCondition *apis.Condition

			BeforeEach(func() {
				readyCondition = &apis.Condition{
					Type:    apis.ConditionReady,
					Status:  corev1.ConditionUnknown,
					Reason:  "Pending",
					Message: "TaskRun is in pending status",
				}
				runner.TaskRun.Status.SetCondition(readyCondition)
			})

			It("does not report failure details and messages", func(ctx SpecContext) {
				Expect(runner.SyncBuildRunStatus(ctx, k8sClient, buildRun)).To(Succeed())
				Expect(buildRun.Status.FailureDetails).To(BeNil(), "status failure details")
			})

		})

		Context("and the TaskRun is in progress", func() {

			var succeededCondition *apis.Condition

			BeforeEach(func() {

				runner.TaskRun.Status.StartTime = &metav1.Time{Time: time.Now()}
				// Reason is not set so tests can modify for different Tekton scenarios
				succeededCondition = &apis.Condition{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionUnknown,
					Message: "not relevant",
				}
			})

			It("updates the BuildRun start time if it hasn't been set", func(ctx SpecContext) {
				succeededCondition.Reason = pipelinev1.TaskRunReasonRunning.String()
				runner.TaskRun.Status.SetCondition(succeededCondition)

				Expect(runner.SyncBuildRunStatus(ctx, k8sClient, buildRun)).To(Succeed(), "sync BuildRun status")
				Expect(buildRun.Status.StartTime).ToNot(BeNil(), "buildRun start time")
				Expect(buildRun.Status.StartTime).To(BeEquivalentTo(runner.TaskRun.Status.StartTime), "buildRun start time")
			})

			It("does not update the BuildRun start time if it has been set", func(ctx SpecContext) {
				// This covers a hypothetical situation where the underlying runner can be retried.
				// Set the start time to "10 minutes ago"
				expected := &metav1.Time{Time: time.Now().Add(-10 * time.Minute)}
				buildRun.Status.StartTime = expected
				succeededCondition.Reason = pipelinev1.TaskRunReasonRunning.String()
				runner.TaskRun.Status.SetCondition(succeededCondition)

				Expect(runner.SyncBuildRunStatus(ctx, k8sClient, buildRun)).To(Succeed(), "sync BuildRun status")
				Expect(buildRun.Status.StartTime).ToNot(BeNil(), "buildRun start time")
				Expect(buildRun.Status.StartTime).To(BeEquivalentTo(expected), "buildRun start time")
			})

			PIt("updates Succeeded status to Unknown if the TaskRun has started")

			It("updates Succeeded status to Unknown if the TaskRun is running", func(ctx SpecContext) {
				succeededCondition.Reason = pipelinev1.TaskRunReasonRunning.String()
				runner.TaskRun.Status.SetCondition(succeededCondition)

				Expect(runner.SyncBuildRunStatus(ctx, k8sClient, buildRun)).To(Succeed(), "sync BuildRun status")
				buildRunCondition := buildRun.Status.GetCondition(v1beta1.Succeeded)
				Expect(buildRunCondition).NotTo(BeNil(), "succeeded condition")
				Expect(buildRunCondition.Status).To(Equal(corev1.ConditionUnknown), "succeeded condition status")
			})

			It("does not report failure details and messages", func(ctx SpecContext) {
				succeededCondition.Reason = pipelinev1.TaskRunReasonRunning.String()
				runner.TaskRun.Status.SetCondition(succeededCondition)

				Expect(runner.SyncBuildRunStatus(ctx, k8sClient, buildRun)).To(Succeed(), "sync BuildRun status")
				Expect(buildRun.Status.FailureDetails).To(BeNil(), "status failure details")
			})

		})

		Context("and the TaskRun has timed out", func() {

			var succeededCondition *apis.Condition

			BeforeEach(func() {
				// Succeeded condition does not have Reason set
				// Each test case likely needs to set this independently
				buildRun.Spec.Timeout = &metav1.Duration{Duration: 10 * time.Minute}
				runner.TaskRun.Spec.Timeout = &metav1.Duration{Duration: 10 * time.Minute}
				runner.TaskRun.Status.StartTime = &metav1.Time{Time: time.Now().Add(-11 * time.Minute)}
				succeededCondition = &apis.Condition{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionFalse,
					Reason:  pipelinev1.TaskRunReasonTimedOut.String(),
					Message: "not relevant",
				}
				runner.TaskRun.Status.SetCondition(succeededCondition)
			})

			It("updates Succeeded status to False if the TaskRun has timed out", func(ctx SpecContext) {
				Expect(runner.SyncBuildRunStatus(ctx, k8sClient, buildRun)).To(Succeed(), "sync BuildRun status")

				buildRunCondition := buildRun.Status.GetCondition(v1beta1.Succeeded)
				Expect(buildRunCondition).NotTo(BeNil(), "succeeded condition")
				Expect(buildRunCondition.Status).To(Equal(corev1.ConditionFalse), "succeeded condition status")
				Expect(buildRunCondition.Reason).To(Equal("BuildRunTimeout"), "succeeded condition reason")
			})

		})

		Context("and the TaskRun has completed with a failure", func() {

			var succeededCondition *apis.Condition

			BeforeEach(func() {
				runner.TaskRun.Status.StartTime = &metav1.Time{Time: time.Now().Add(-10 * time.Minute)}
				runner.TaskRun.Status.CompletionTime = &metav1.Time{Time: time.Now()}
				succeededCondition = &apis.Condition{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionFalse,
					Reason:  pipelinev1.TaskRunReasonFailed.String(),
					Message: "not relevant",
				}
				runner.TaskRun.Status.SetCondition(succeededCondition)
			})

			It("updates Succeeded status to False if the underlying pod is not found", func(ctx SpecContext) {
				Expect(runner.SyncBuildRunStatus(ctx, k8sClient, buildRun)).To(Succeed(), "sync BuildRun status")
				buildRunCondition := buildRun.Status.GetCondition(v1beta1.Succeeded)
				Expect(buildRunCondition).NotTo(BeNil(), "succeeded condition")
				Expect(buildRunCondition.Status).To(Equal(corev1.ConditionFalse), "succeeded condition status")
				Expect(buildRunCondition.Reason).To(Equal(succeededCondition.Reason), "succeeded condition reason")
			})

			It("updates Succeeded status to False if underlying pod containers are not found", func(ctx SpecContext) {
				// This might be an edge case of a TaskRun creating a pod that has no container status reported.
				failedPod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "evilpod",
						Namespace: runner.TaskRun.GetNamespace(),
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "evilpod-container",
							},
						},
					},
					Status: corev1.PodStatus{},
				}
				Expect(k8sClient.Create(ctx, failedPod)).To(Succeed(), "create fake pod")

				Expect(runner.SyncBuildRunStatus(ctx, k8sClient, buildRun)).To(Succeed(), "sync BuildRun status")
				buildRunCondition := buildRun.Status.GetCondition(v1beta1.Succeeded)
				Expect(buildRunCondition).NotTo(BeNil(), "succeeded condition")
				Expect(buildRunCondition.Status).To(Equal(corev1.ConditionFalse), "succeeded condition status")
				Expect(buildRunCondition.Reason).To(Equal("Failed"), "succeeded condition reason")
			})

			It("updates Succeeded status to False if the TaskRun has failed due to pod eviction", func(ctx SpecContext) {
				// Generate a pod with the status to be evicted
				failedPod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "evilpod",
						Namespace: runner.TaskRun.GetNamespace(),
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "evilpod-container",
							},
						},
					},
					Status: corev1.PodStatus{
						Phase:  corev1.PodFailed,
						Reason: "Evicted",
					},
				}
				Expect(k8sClient.Create(ctx, failedPod)).To(Succeed(), "create fake pod")
				runner.TaskRun.Status.PodName = failedPod.Name

				Expect(runner.SyncBuildRunStatus(ctx, k8sClient, buildRun)).To(Succeed(), "sync BuildRun status")
				buildRunCondition := buildRun.Status.GetCondition(v1beta1.Succeeded)
				Expect(buildRunCondition).NotTo(BeNil(), "succeeded condition")
				Expect(buildRunCondition.Status).To(Equal(corev1.ConditionFalse), "succeeded condition status")
				Expect(buildRunCondition.Reason).To(Equal(v1beta1.BuildRunStatePodEvicted), "succeeded condition reason")
			})

			It("updates Succeeded status to False if the TaskRun has failed due to to OOMKilled", func(ctx SpecContext) {
				// Generate a pod with the status to out of memory
				failedPod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "evilpod",
						Namespace: runner.TaskRun.GetNamespace(),
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "evilpod-container",
							},
						},
					},
					Status: corev1.PodStatus{
						Phase:  corev1.PodFailed,
						Reason: "Error",
						ContainerStatuses: []corev1.ContainerStatus{{
							Name: "evilpod-container",
							State: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 128,
									Reason:   "OOMKilled",
								},
							},
						}},
					},
				}
				Expect(k8sClient.Create(ctx, failedPod)).To(Succeed(), "create fake pod")
				runner.TaskRun.Status.PodName = failedPod.Name

				Expect(runner.SyncBuildRunStatus(ctx, k8sClient, buildRun)).To(Succeed(), "sync BuildRun status")
				buildRunCondition := buildRun.Status.GetCondition(v1beta1.Succeeded)
				Expect(buildRunCondition).NotTo(BeNil(), "succeeded condition")
				Expect(buildRunCondition.Status).To(Equal(corev1.ConditionFalse), "succeeded condition status")
				Expect(buildRunCondition.Reason).To(Equal(v1beta1.BuildRunStateStepOutOfMemory), "succeeded condition reason")
			})

			It("updates Succeeded status to False if the TaskRun has failed due to the build failing", func(ctx SpecContext) {
				// generate a pod that have a single container and
				// one entry in the ContainerStatuses field, with
				// an exitCode
				failedPod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foopod",
						Namespace: runner.TaskRun.GetNamespace(),
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "step-build",
							},
						},
					},
					Status: corev1.PodStatus{
						ContainerStatuses: []corev1.ContainerStatus{
							{
								Name: "step-build",
								State: corev1.ContainerState{
									Terminated: &corev1.ContainerStateTerminated{
										Reason:   "Error",
										ExitCode: 1,
									},
								},
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, failedPod)).To(Succeed(), "create fake pod")
				runner.TaskRun.Status.PodName = failedPod.Name

				Expect(runner.SyncBuildRunStatus(ctx, k8sClient, buildRun)).To(Succeed(), "sync BuildRun status")
				buildRunCondition := buildRun.Status.GetCondition(v1beta1.Succeeded)
				Expect(buildRunCondition).NotTo(BeNil(), "succeeded condition")
				Expect(buildRunCondition.Status).To(Equal(corev1.ConditionFalse), "succeeded condition status")
				Expect(buildRunCondition.Reason).To(Equal("Failed"), "succeeded condition reason")

			})

			It("updates Succeeded status to False if the TaskRun has failed to a vulnerability scan", func(ctx SpecContext) {
				// Generate a pod with name step-image-processing and exitCode 22
				failedPod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "evilpod",
						Namespace: runner.TaskRun.GetNamespace(),
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "step-image-processing",
							},
						},
					},
					Status: corev1.PodStatus{
						Reason: "VulnerabilityScanFailed",
						ContainerStatuses: []corev1.ContainerStatus{
							{
								Name: "step-image-processing",
								State: corev1.ContainerState{
									Terminated: &corev1.ContainerStateTerminated{
										ExitCode: 22,
									},
								},
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, failedPod)).To(Succeed(), "create fake pod")
				runner.TaskRun.Status.PodName = failedPod.Name

				Expect(runner.SyncBuildRunStatus(ctx, k8sClient, buildRun)).To(Succeed(), "sync BuildRun status")
				buildRunCondition := buildRun.Status.GetCondition(v1beta1.Succeeded)
				Expect(buildRunCondition).NotTo(BeNil(), "succeeded condition")
				Expect(buildRunCondition.Status).To(Equal(corev1.ConditionFalse), "succeeded condition status")
				Expect(buildRunCondition.Reason).To(Equal(v1beta1.BuildRunStateVulnerabilitiesFound), "succeeded condition reason")

			})

			It("reports failure reasons and messages in the TaskRun results", func(ctx SpecContext) {
				// Shipwright build steps can surface failure messages by writing to the
				// container's termination log, with a JSON-formatted message that matches the
				// Tekton RunResult schema.
				errorReasonValue := "PullBaseImageFailed"
				errorMessageValue := "Failed to pull the base image."
				errorReasonKey := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, resultErrorReason)
				errorMessageKey := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, resultErrorMessage)

				errorReason := result.RunResult{Key: errorReasonKey, Value: errorReasonValue}
				errorMessage := result.RunResult{Key: errorMessageKey, Value: errorMessageValue}
				unrelated := result.RunResult{Key: "unrelated-resource-key", Value: "Unrelated resource value"}

				message, err := json.Marshal([]result.RunResult{errorReason, errorMessage, unrelated})
				Expect(err).NotTo(HaveOccurred())

				failedStep := pipelinev1.StepState{
					Name: "build",
					ContainerState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							Message:  string(message),
							ExitCode: 1,
						},
					},
				}

				followUpStep := pipelinev1.StepState{
					Name: "push",
				}

				runner.TaskRun.Status.Steps = append(runner.TaskRun.Status.Steps, failedStep, followUpStep)

				Expect(runner.SyncBuildRunStatus(ctx, k8sClient, buildRun)).To(Succeed())
				Expect(buildRun.Status.FailureDetails).NotTo(BeNil(), "status failure details")
				Expect(buildRun.Status.FailureDetails.Message).To(Equal(errorMessageValue), "status failure details message")
				Expect(buildRun.Status.FailureDetails.Reason).To(Equal(errorReasonValue), "status failure details reason")
			})

			It("does not report failure reasons and messages from unrelated TaskRun results", func(ctx SpecContext) {

				// Tasks can in theory surface unrelated result messages through separate keys
				unrelated := result.RunResult{Key: "unrelated-resource-key", Value: "Unrelated resource value"}

				message, err := json.Marshal([]result.RunResult{unrelated})
				Expect(err).NotTo(HaveOccurred())

				failedStep := pipelinev1.StepState{
					Name: "build",
					ContainerState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							Message:  string(message),
							ExitCode: 1,
						},
					},
				}

				followUpStep := pipelinev1.StepState{
					Name: "push",
				}

				runner.TaskRun.Status.Steps = append(runner.TaskRun.Status.Steps, failedStep, followUpStep)

				Expect(runner.SyncBuildRunStatus(ctx, k8sClient, buildRun)).To(Succeed())
				Expect(buildRun.Status.FailureDetails).NotTo(BeNil(), "status failure details")
				Expect(buildRun.Status.FailureDetails.Message).To(BeEmpty(), "status failure details message")
				Expect(buildRun.Status.FailureDetails.Reason).To(BeEmpty(), "status failure details reason")
			})

			It("updates the completion time if it has not been set", func(ctx SpecContext) {
				Expect(runner.SyncBuildRunStatus(ctx, k8sClient, buildRun)).To(Succeed())
				Expect(buildRun.Status.CompletionTime).To(BeEquivalentTo(runner.GetCompletionTime()))
			})

			It("does not update the completion time if it has been set", func(ctx SpecContext) {
				buildRun.Status.CompletionTime = &metav1.Time{Time: time.Now().Add(-15 * time.Minute)}
				Expect(runner.SyncBuildRunStatus(ctx, k8sClient, buildRun)).To(Succeed())
				Expect(buildRun.Status.CompletionTime).NotTo(BeEquivalentTo(runner.GetCompletionTime()))
			})

		})

		Context("and the TaskRun has completed with success", func() {

			var succeededCondition *apis.Condition

			BeforeEach(func() {
				runner.TaskRun.Status.StartTime = &metav1.Time{Time: time.Now().Add(-10 * time.Minute)}
				runner.TaskRun.Status.CompletionTime = &metav1.Time{Time: time.Now()}
				succeededCondition = &apis.Condition{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionTrue,
					Reason:  pipelinev1.TaskRunReasonSuccessful.String(),
					Message: "not relevant",
				}
				runner.TaskRun.Status.SetCondition(succeededCondition)
			})

			PIt("updates Succeeded status to True if the TaskRun has succeeded")

			It("does not report failure reasons and messages", func(ctx SpecContext) {
				Expect(runner.SyncBuildRunStatus(ctx, k8sClient, buildRun))
				Expect(buildRun.Status.FailureDetails).To(BeNil(), "status failure details")
			})

			It("does not report failure reasons and messages if step has well-formatted termination log results", func(ctx SpecContext) {
				errorReasonValue := "PullBaseImageFailed"
				errorMessageValue := "Failed to pull the base image."
				errorReasonKey := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, resultErrorReason)
				errorMessageKey := fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, resultErrorMessage)

				errorReason := result.RunResult{Key: errorReasonKey, Value: errorReasonValue}
				errorMessage := result.RunResult{Key: errorMessageKey, Value: errorMessageValue}

				message, err := json.Marshal([]result.RunResult{errorReason, errorMessage})
				Expect(err).NotTo(HaveOccurred())

				finishedStep := pipelinev1.StepState{
					Name: "build",
					ContainerState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							Message:  string(message),
							ExitCode: 0,
						},
					},
				}

				runner.TaskRun.Status.Steps = append(runner.TaskRun.Status.Steps, finishedStep)

				Expect(runner.SyncBuildRunStatus(ctx, k8sClient, buildRun)).To(Succeed())
				Expect(buildRun.Status.FailureDetails).To(BeNil(), "status failure details")
			})

			It("updates the completion time if it has not been set", func(ctx SpecContext) {
				Expect(runner.SyncBuildRunStatus(ctx, k8sClient, buildRun)).To(Succeed())
				Expect(buildRun.Status.CompletionTime).To(BeEquivalentTo(runner.GetCompletionTime()))
			})

			It("does not update the completion time if it has been set", func(ctx SpecContext) {
				buildRun.Status.CompletionTime = &metav1.Time{Time: time.Now().Add(-15 * time.Minute)}
				Expect(runner.SyncBuildRunStatus(ctx, k8sClient, buildRun)).To(Succeed())
				Expect(buildRun.Status.CompletionTime).NotTo(BeEquivalentTo(runner.GetCompletionTime()))
			})

		})

	})

	When("the build has been cancelled", func() {

		PIt("updates Succeeded status to Unknown if the TaskRun has not been canceled yet")

		PIt("updates Succeeded status to False if the TaskRun has been canceled")
	})
})
