// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package conversion

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

	buildapialpha "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

var _ = Describe("BuildRun Conversion", func() {
	ctx := context.TODO()

	Context("convertBuildRunBetaToAlpha", func() {
		It("converts a BuildRun from v1beta1 to v1alpha1 with BuildRef and generated serviceAccount", func() {
			timeout := metav1.Duration{Duration: 30 * time.Minute}
			retentionTTL := metav1.Duration{Duration: 2 * time.Hour}
			now := metav1.Now()

			betaBuildRun := &buildapi.BuildRun{
				TypeMeta: metav1.TypeMeta{
					APIVersion: betaGroupVersion,
					Kind:       buildRunKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sample-buildrun",
					Namespace: "default",
				},
				Spec: buildapi.BuildRunSpec{
					Build: buildapi.ReferencedBuild{
						Name: ptr.To("my-build"),
					},
					ServiceAccount: ptr.To(".generate"),
					Timeout:        &timeout,
					ParamValues: []buildapi.ParamValue{
						{
							Name: "param1",
							SingleValue: &buildapi.SingleValue{
								Value: ptr.To("val1"),
							},
						},
						{
							Name: "param2",
							SingleValue: &buildapi.SingleValue{
								ConfigMapValue: &buildapi.ObjectKeyRef{
									Name: "cm-ref",
									Key:  "key-ref",
								},
							},
						},
						{
							Name: "param3",
							SingleValue: &buildapi.SingleValue{
								SecretValue: &buildapi.ObjectKeyRef{
									Name: "sec-ref",
									Key:  "sec-key",
								},
							},
						},
					},
					Output: &buildapi.Image{
						Image:       "docker.io/repo/app:latest",
						PushSecret:  ptr.To("output-secret"),
						Insecure:    ptr.To(true),
						Timestamp:   ptr.To("Zero"),
						Annotations: map[string]string{"label": "test"},
						Labels:      map[string]string{"env": "staging"},
					},
					State: buildapi.BuildRunRequestedStatePtr(buildapi.BuildRunStateCancel),
					Env: []corev1.EnvVar{
						{Name: "FOO", Value: "BAR"},
					},
					Retention: &buildapi.BuildRunRetention{
						TTLAfterFailed: &retentionTTL,
					},
					Volumes: []buildapi.BuildVolume{
						{
							Name: "run-vol",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
				Status: buildapi.BuildRunStatus{
					StartTime:      &now,
					CompletionTime: &now,
					Conditions: []buildapi.Condition{
						{
							Type:               buildapi.Succeeded,
							Status:             corev1.ConditionFalse,
							Reason:             "Failed",
							Message:            "build step failed",
							LastTransitionTime: now,
						},
					},
					Source: &buildapi.SourceResult{
						Git: &buildapi.GitSourceResult{
							CommitAuthor: "dev@example.com",
							CommitSha:    "abcdef123456",
							BranchName:   "main",
						},
						Timestamp: &now,
					},
					Output: &buildapi.Output{
						Digest: "sha256:1234567890abcdef",
						Size:   1024,
					},
					Executor: &buildapi.BuildExecutor{
						Name: "sample-buildrun-tr",
						Kind: "TaskRun",
					},
					FailureDetails: &buildapi.FailureDetails{
						Reason:  "TaskRunFailed",
						Message: "Step error in container",
						Location: &buildapi.Location{
							Pod:       "sample-buildrun-tr-pod",
							Container: "step-build",
						},
					},
				},
			}

			unstructuredObj := &unstructured.Unstructured{}
			err := convertBuildRunBetaToAlpha(ctx, betaBuildRun, unstructuredObj)
			Expect(err).ToNot(HaveOccurred())

			var alphaBuildRun buildapialpha.BuildRun
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.UnstructuredContent(), &alphaBuildRun)
			Expect(err).ToNot(HaveOccurred())

			Expect(alphaBuildRun.APIVersion).To(Equal(alphaGroupVersion))
			Expect(alphaBuildRun.Name).To(Equal("sample-buildrun"))

			// BuildRef
			Expect(alphaBuildRun.Spec.BuildRef).ToNot(BeNil())
			Expect(alphaBuildRun.Spec.BuildRef.Name).To(Equal("my-build"))
			Expect(alphaBuildRun.Spec.BuildSpec).To(BeNil())

			// ServiceAccount generated
			Expect(alphaBuildRun.Spec.ServiceAccount).ToNot(BeNil())
			Expect(alphaBuildRun.Spec.ServiceAccount.Name).To(Equal(ptr.To("sample-buildrun")))
			Expect(alphaBuildRun.Spec.ServiceAccount.Generate).To(Equal(ptr.To(true)))

			// Timeout, State, Env, Retention, Volumes
			Expect(alphaBuildRun.Spec.Timeout).To(Equal(&timeout))
			Expect(alphaBuildRun.Spec.State).To(Equal(buildapialpha.BuildRunRequestedStatePtr(buildapialpha.BuildRunStateCancel)))
			Expect(alphaBuildRun.Spec.Env).To(Equal([]corev1.EnvVar{{Name: "FOO", Value: "BAR"}}))
			Expect(alphaBuildRun.Spec.Retention.TTLAfterFailed).To(Equal(&retentionTTL))
			Expect(alphaBuildRun.Spec.Volumes).To(HaveLen(1))
			Expect(alphaBuildRun.Spec.Volumes[0].Name).To(Equal("run-vol"))

			// Output
			Expect(alphaBuildRun.Spec.Output.Image).To(Equal("docker.io/repo/app:latest"))
			Expect(alphaBuildRun.Spec.Output.Credentials).To(Equal(&corev1.LocalObjectReference{Name: "output-secret"}))
			Expect(alphaBuildRun.Spec.Output.Insecure).To(Equal(ptr.To(true)))
			Expect(alphaBuildRun.Spec.Output.Timestamp).To(Equal(ptr.To("Zero")))
			Expect(alphaBuildRun.Spec.Output.Annotations).To(Equal(map[string]string{"label": "test"}))
			Expect(alphaBuildRun.Spec.Output.Labels).To(Equal(map[string]string{"env": "staging"}))

			// Status conditions, source, output
			Expect(alphaBuildRun.Status.Conditions).To(HaveLen(1))
			Expect(alphaBuildRun.Status.Conditions[0].Reason).To(Equal("Failed"))
			Expect(alphaBuildRun.Status.Sources).To(HaveLen(1))
			Expect(alphaBuildRun.Status.Sources[0].Git.CommitSha).To(Equal("abcdef123456"))
			Expect(alphaBuildRun.Status.Output.Digest).To(Equal("sha256:1234567890abcdef"))
			Expect(alphaBuildRun.Status.Output.Size).To(Equal(int64(1024)))

			// Executor -> LatestTaskRunRef
			Expect(alphaBuildRun.Status.LatestTaskRunRef).To(Equal(ptr.To("sample-buildrun-tr")))

			// FailureDetails and deprecated FailedAt
			Expect(alphaBuildRun.Status.FailureDetails).ToNot(BeNil())
			Expect(alphaBuildRun.Status.FailureDetails.Reason).To(Equal("TaskRunFailed"))
			Expect(alphaBuildRun.Status.FailureDetails.Location.Pod).To(Equal("sample-buildrun-tr-pod"))
			Expect(alphaBuildRun.Status.FailureDetails.Location.Container).To(Equal("step-build"))
			// Deprecated FailedAt must mirror Location
			//nolint:staticcheck // SA1019 verify deprecated FailedAt field is populated for backward compatibility
			Expect(alphaBuildRun.Status.FailedAt).ToNot(BeNil())
			//nolint:staticcheck // SA1019 verify deprecated FailedAt field is populated for backward compatibility
			Expect(alphaBuildRun.Status.FailedAt.Pod).To(Equal("sample-buildrun-tr-pod"))
			//nolint:staticcheck // SA1019 verify deprecated FailedAt field is populated for backward compatibility
			Expect(alphaBuildRun.Status.FailedAt.Container).To(Equal("step-build"))
		})

		It("converts a BuildRun from v1beta1 to v1alpha1 with embedded BuildSpec and Local source", func() {
			localTimeout := metav1.Duration{Duration: 2 * time.Minute}
			betaBuildRun := &buildapi.BuildRun{
				TypeMeta: metav1.TypeMeta{
					APIVersion: betaGroupVersion,
					Kind:       buildRunKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "embed-buildrun",
				},
				Spec: buildapi.BuildRunSpec{
					Build: buildapi.ReferencedBuild{
						Spec: &buildapi.BuildSpec{
							Strategy: buildapi.Strategy{
								Name: "buildah",
							},
						},
					},
					Source: &buildapi.BuildRunSource{
						Type: buildapi.LocalType,
						Local: &buildapi.Local{
							Name:    "upload-data",
							Timeout: &localTimeout,
						},
					},
					ServiceAccount: ptr.To("custom-sa"),
				},
				Status: buildapi.BuildRunStatus{
					TaskRunName: ptr.To("fallback-taskrun-ref"), //nolint:staticcheck // SA1019 verify deprecated TaskRunName fallback
					Source: &buildapi.SourceResult{
						OciArtifact: &buildapi.OciArtifactSourceResult{
							Digest: "sha256:bundle-digest",
						},
					},
					BuildSpec: &buildapi.BuildSpec{
						Strategy: buildapi.Strategy{
							Name: "buildah",
						},
					},
				},
			}

			unstructuredObj := &unstructured.Unstructured{}
			err := convertBuildRunBetaToAlpha(ctx, betaBuildRun, unstructuredObj)
			Expect(err).ToNot(HaveOccurred())

			var alphaBuildRun buildapialpha.BuildRun
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.UnstructuredContent(), &alphaBuildRun)
			Expect(err).ToNot(HaveOccurred())

			Expect(alphaBuildRun.Spec.BuildSpec).ToNot(BeNil())
			Expect(alphaBuildRun.Spec.BuildSpec.Strategy.Name).To(Equal("buildah"))
			Expect(alphaBuildRun.Spec.BuildRef).To(BeNil())

			Expect(alphaBuildRun.Spec.Sources).To(HaveLen(1))
			Expect(alphaBuildRun.Spec.Sources[0].Type).To(Equal(buildapialpha.LocalCopy))
			Expect(alphaBuildRun.Spec.Sources[0].Name).To(Equal("upload-data"))
			Expect(alphaBuildRun.Spec.Sources[0].Timeout).To(Equal(&localTimeout))

			Expect(alphaBuildRun.Spec.ServiceAccount.Name).To(Equal(ptr.To("custom-sa")))
			Expect(alphaBuildRun.Spec.ServiceAccount.Generate).To(BeNil())

			// Fallback TaskRunName -> LatestTaskRunRef when Executor is nil
			Expect(alphaBuildRun.Status.LatestTaskRunRef).To(Equal(ptr.To("fallback-taskrun-ref")))

			// OCIArtifact status
			Expect(alphaBuildRun.Status.Sources).To(HaveLen(1))
			Expect(alphaBuildRun.Status.Sources[0].Bundle.Digest).To(Equal("sha256:bundle-digest"))

			// Status.BuildSpec
			Expect(alphaBuildRun.Status.BuildSpec).ToNot(BeNil())
			Expect(alphaBuildRun.Status.BuildSpec.Strategy.Name).To(Equal("buildah"))
		})
	})

	Context("convertBuildRunAlphaToBeta", func() {
		It("converts a BuildRun from v1alpha1 to v1beta1 with BuildRef and Generate SA", func() {
			now := metav1.Now()
			alphaBuildRun := &buildapialpha.BuildRun{
				TypeMeta: metav1.TypeMeta{
					APIVersion: alphaGroupVersion,
					Kind:       buildRunKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "alpha-run",
					Namespace: "ns1",
				},
				Spec: buildapialpha.BuildRunSpec{
					BuildRef: &buildapialpha.BuildRef{
						Name: "alpha-build-ref",
					},
					ServiceAccount: &buildapialpha.ServiceAccount{
						Generate: ptr.To(true),
					},
					Sources: []buildapialpha.BuildSource{
						{
							Type: buildapialpha.LocalCopy,
							Name: "local-copy-src",
						},
					},
					Output: &buildapialpha.Image{
						Image:       "quay.io/target/image:v1",
						Credentials: &corev1.LocalObjectReference{Name: "target-cred"},
						Insecure:    ptr.To(false),
					},
				},
				Status: buildapialpha.BuildRunStatus{
					LatestTaskRunRef: ptr.To("taskrun-abc"),
					FailureDetails: &buildapialpha.FailureDetails{
						Reason:  "StepFailed",
						Message: "exit code 1",
						Location: &buildapialpha.FailedAt{
							Pod:       "pod-1",
							Container: "step-1",
						},
					},
					Sources: []buildapialpha.SourceResult{
						{
							Git: &buildapialpha.GitSourceResult{
								CommitSha: "deadbeef",
							},
							Timestamp: &now,
						},
					},
					Output: &buildapialpha.Output{
						Digest: "sha256:digest-abc",
						Size:   512,
					},
				},
			}

			unstructuredObj := &unstructured.Unstructured{}
			mapito, err := runtime.DefaultUnstructuredConverter.ToUnstructured(alphaBuildRun)
			Expect(err).ToNot(HaveOccurred())
			unstructuredObj.SetUnstructuredContent(mapito)

			var betaBuildRun buildapi.BuildRun
			err = convertBuildRunAlphaToBeta(ctx, &betaBuildRun, unstructuredObj)
			Expect(err).ToNot(HaveOccurred())

			Expect(betaBuildRun.Spec.Build.Name).To(Equal(ptr.To("alpha-build-ref")))
			Expect(betaBuildRun.Spec.Build.Spec).To(BeNil())

			// ServiceAccount Generate: true -> ".generate"
			Expect(betaBuildRun.Spec.ServiceAccount).To(Equal(ptr.To(".generate")))

			// Local source
			Expect(betaBuildRun.Spec.Source).ToNot(BeNil())
			Expect(betaBuildRun.Spec.Source.Type).To(Equal(buildapi.LocalType))
			Expect(betaBuildRun.Spec.Source.Local.Name).To(Equal("local-copy-src"))

			// Output
			Expect(betaBuildRun.Spec.Output.Image).To(Equal("quay.io/target/image:v1"))
			Expect(betaBuildRun.Spec.Output.PushSecret).To(Equal(ptr.To("target-cred")))
			Expect(betaBuildRun.Spec.Output.Insecure).To(Equal(ptr.To(false)))

			// Status: LatestTaskRunRef -> Executor (TaskRun) and TaskRunName
			Expect(betaBuildRun.Status.Executor).ToNot(BeNil())
			Expect(betaBuildRun.Status.Executor.Name).To(Equal("taskrun-abc"))
			Expect(betaBuildRun.Status.Executor.Kind).To(Equal("TaskRun"))
			//nolint:staticcheck // SA1019 verify deprecated TaskRunName field is populated for backward compatibility
			Expect(betaBuildRun.Status.TaskRunName).To(Equal(ptr.To("taskrun-abc")))

			// FailureDetails
			Expect(betaBuildRun.Status.FailureDetails).ToNot(BeNil())
			Expect(betaBuildRun.Status.FailureDetails.Reason).To(Equal("StepFailed"))
			Expect(betaBuildRun.Status.FailureDetails.Location.Pod).To(Equal("pod-1"))
			Expect(betaBuildRun.Status.FailureDetails.Location.Container).To(Equal("step-1"))

			// Source Result
			Expect(betaBuildRun.Status.Source).ToNot(BeNil())
			Expect(betaBuildRun.Status.Source.Git.CommitSha).To(Equal("deadbeef"))
			Expect(betaBuildRun.Status.Output.Digest).To(Equal("sha256:digest-abc"))
		})

		It("converts a BuildRun from v1alpha1 to v1beta1 with embedded BuildSpec and standard SA", func() {
			alphaBuildRun := &buildapialpha.BuildRun{
				TypeMeta: metav1.TypeMeta{
					APIVersion: alphaGroupVersion,
					Kind:       buildRunKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "embed-alpha-run",
				},
				Spec: buildapialpha.BuildRunSpec{
					BuildSpec: &buildapialpha.BuildSpec{
						Strategy: buildapialpha.Strategy{
							Name: "s2i",
						},
					},
					ServiceAccount: &buildapialpha.ServiceAccount{
						Name: ptr.To("pipeline-sa"),
					},
				},
				Status: buildapialpha.BuildRunStatus{
					BuildSpec: &buildapialpha.BuildSpec{
						Strategy: buildapialpha.Strategy{
							Name: "s2i",
						},
					},
				},
			}

			unstructuredObj := &unstructured.Unstructured{}
			mapito, err := runtime.DefaultUnstructuredConverter.ToUnstructured(alphaBuildRun)
			Expect(err).ToNot(HaveOccurred())
			unstructuredObj.SetUnstructuredContent(mapito)

			var betaBuildRun buildapi.BuildRun
			err = convertBuildRunAlphaToBeta(ctx, &betaBuildRun, unstructuredObj)
			Expect(err).ToNot(HaveOccurred())

			Expect(betaBuildRun.Spec.Build.Spec).ToNot(BeNil())
			Expect(betaBuildRun.Spec.Build.Spec.Strategy.Name).To(Equal("s2i"))
			Expect(betaBuildRun.Spec.ServiceAccount).To(Equal(ptr.To("pipeline-sa")))
			Expect(betaBuildRun.Status.BuildSpec).ToNot(BeNil())
			Expect(betaBuildRun.Status.BuildSpec.Strategy.Name).To(Equal("s2i"))
		})
	})

	Context("Round-trip BuildRun conversions", func() {
		It("preserves data in a round trip beta -> alpha -> beta", func() {
			timeout := metav1.Duration{Duration: 25 * time.Minute}
			retentionTTL := metav1.Duration{Duration: 4 * time.Hour}

			originalBeta := &buildapi.BuildRun{
				TypeMeta: metav1.TypeMeta{
					APIVersion: betaGroupVersion,
					Kind:       buildRunKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rt-run",
					Namespace: "default",
				},
				Spec: buildapi.BuildRunSpec{
					Build: buildapi.ReferencedBuild{
						Name: ptr.To("rt-build-target"),
					},
					ServiceAccount: ptr.To("my-sa"),
					Timeout:        &timeout,
					ParamValues: []buildapi.ParamValue{
						{
							Name: "PARAM",
							SingleValue: &buildapi.SingleValue{
								Value: ptr.To("VAL"),
							},
						},
					},
					Output: &buildapi.Image{
						Image:      "repo/img:tag",
						PushSecret: ptr.To("push-secret"),
						Insecure:   ptr.To(true),
					},
					Retention: &buildapi.BuildRunRetention{
						TTLAfterFailed: &retentionTTL,
					},
				},
				Status: buildapi.BuildRunStatus{
					Conditions: []buildapi.Condition{
						{
							Type:   buildapi.Succeeded,
							Status: corev1.ConditionTrue,
						},
					},
					Executor: &buildapi.BuildExecutor{
						Name: "taskrun-ref-123",
						Kind: "TaskRun",
					},
					Output: &buildapi.Output{
						Digest: "sha256:xyz",
						Size:   2048,
					},
				},
			}

			// Beta -> Alpha
			alphaUnstructured := &unstructured.Unstructured{}
			err := convertBuildRunBetaToAlpha(ctx, originalBeta, alphaUnstructured)
			Expect(err).ToNot(HaveOccurred())

			// Alpha -> Beta
			var convertedBeta buildapi.BuildRun
			err = convertBuildRunAlphaToBeta(ctx, &convertedBeta, alphaUnstructured)
			Expect(err).ToNot(HaveOccurred())

			Expect(convertedBeta.Name).To(Equal(originalBeta.Name))
			Expect(convertedBeta.Spec.Build.Name).To(Equal(originalBeta.Spec.Build.Name))
			Expect(convertedBeta.Spec.ServiceAccount).To(Equal(originalBeta.Spec.ServiceAccount))
			Expect(convertedBeta.Spec.Timeout).To(Equal(originalBeta.Spec.Timeout))
			Expect(convertedBeta.Spec.Output.Image).To(Equal(originalBeta.Spec.Output.Image))
			Expect(convertedBeta.Spec.Output.PushSecret).To(Equal(originalBeta.Spec.Output.PushSecret))
			Expect(convertedBeta.Spec.Output.Insecure).To(Equal(originalBeta.Spec.Output.Insecure))
			Expect(convertedBeta.Spec.Retention.TTLAfterFailed).To(Equal(originalBeta.Spec.Retention.TTLAfterFailed))
			Expect(convertedBeta.Status.Conditions[0].Status).To(Equal(corev1.ConditionTrue))
			Expect(convertedBeta.Status.Executor.Name).To(Equal("taskrun-ref-123"))
			Expect(convertedBeta.Status.Executor.Kind).To(Equal("TaskRun"))
			Expect(convertedBeta.Status.Output.Digest).To(Equal("sha256:xyz"))
			Expect(convertedBeta.Status.Output.Size).To(Equal(int64(2048)))
		})
	})
})
