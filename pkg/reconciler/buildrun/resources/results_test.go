// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	build "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/controller/fakes"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
	test "github.com/shipwright-io/build/test/v1beta1_samples"

	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("TaskRun results to BuildRun", func() {
	var ctl test.Catalog

	Context("when a BuildRun complete successfully", func() {
		var (
			taskRunRequest reconcile.Request
			br             *build.BuildRun
			tr             *pipelineapi.TaskRun
		)

		ctx := context.Background()

		// returns a reconcile.Request based on an resource name and namespace
		newReconcileRequest := func(name string, ns string) reconcile.Request {
			return reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      name,
					Namespace: ns,
				},
			}
		}

		BeforeEach(func() {
			taskRunRequest = newReconcileRequest("foo-p8nts", "bar")
			tr = ctl.DefaultTaskRun("foo-p8nts", "bar")
			br = ctl.DefaultBuildRun("foo", "bar")
		})

		It("should surface the TaskRun results emitting from default(git) source step", func() {
			commitSha := "0e0583421a5e4bf562ffe33f3651e16ba0c78591"
			br.Status.BuildSpec = &build.BuildSpec{
				Source: &build.Source{
					Type: build.GitType,
					Git: &build.Git{
						URL: "https://github.com/shipwright-io/sample-go",
					},
				},
			}
			tr.Status.Results = append(tr.Status.Results,
				pipelineapi.TaskRunResult{
					Name: "shp-source-default-commit-sha",
					Value: pipelineapi.ParamValue{
						Type:      pipelineapi.ParamTypeString,
						StringVal: commitSha,
					},
				},
				pipelineapi.TaskRunResult{
					Name: "shp-source-default-commit-author",
					Value: pipelineapi.ParamValue{
						Type:      pipelineapi.ParamTypeString,
						StringVal: "foo bar",
					},
				})

			resources.UpdateBuildRunUsingTaskResults(ctx, br, tr.Status.Results, taskRunRequest)

			Expect(br.Status.Source).ToNot(BeNil())
			Expect(br.Status.Source.Git.CommitSha).To(Equal(commitSha))
			Expect(br.Status.Source.Git.CommitAuthor).To(Equal("foo bar"))
		})

		It("should surface the TaskRun results emitting from default(bundle) source step", func() {
			bundleImageDigest := "sha256:fe1b73cd25ac3f11dec752755e2"
			br.Status.BuildSpec = &build.BuildSpec{
				Source: &build.Source{
					Type: build.OCIArtifactType,
					OCIArtifact: &build.OCIArtifact{
						Image: "ghcr.io/shipwright-io/sample-go/source-bundle:latest",
					},
				},
			}

			tr.Status.Results = append(tr.Status.Results,
				pipelineapi.TaskRunResult{
					Name: "shp-source-default-image-digest",
					Value: pipelineapi.ParamValue{
						Type:      pipelineapi.ParamTypeString,
						StringVal: bundleImageDigest,
					},
				})

			resources.UpdateBuildRunUsingTaskResults(ctx, br, tr.Status.Results, taskRunRequest)

			Expect(br.Status.Source).ToNot(BeNil())
			Expect(br.Status.Source.OciArtifact.Digest).To(Equal(bundleImageDigest))
		})

		It("should surface the TaskRun results emitting from output step with image vulnerabilities", func() {
			imageDigest := "sha256:fe1b73cd25ac3f11dec752755e2"
			tr.Status.Results = append(tr.Status.Results,
				pipelineapi.TaskRunResult{
					Name: "shp-image-digest",
					Value: pipelineapi.ParamValue{
						Type:      pipelineapi.ParamTypeString,
						StringVal: imageDigest,
					},
				},
				pipelineapi.TaskRunResult{
					Name: "shp-image-size",
					Value: pipelineapi.ParamValue{
						Type:      pipelineapi.ParamTypeString,
						StringVal: "230",
					},
				},
				pipelineapi.TaskRunResult{
					Name: "shp-image-vulnerabilities",
					Value: pipelineapi.ParamValue{
						Type:      pipelineapi.ParamTypeString,
						StringVal: "CVE-2019-12900:c,CVE-2019-8457:h",
					},
				})

			resources.UpdateBuildRunUsingTaskResults(ctx, br, tr.Status.Results, taskRunRequest)

			Expect(br.Status.Output.Digest).To(Equal(imageDigest))
			Expect(br.Status.Output.Size).To(Equal(int64(230)))
			Expect(br.Status.Output.Vulnerabilities).To(HaveLen(2))
			Expect(br.Status.Output.Vulnerabilities[0].ID).To(Equal("CVE-2019-12900"))
			Expect(br.Status.Output.Vulnerabilities[0].Severity).To(Equal(build.Critical))
		})

		It("should surface the TaskRun results emitting from output step without image vulnerabilities", func() {
			imageDigest := "sha256:fe1b73cd25ac3f11dec752755e2"
			tr.Status.Results = append(tr.Status.Results,
				pipelineapi.TaskRunResult{
					Name: "shp-image-digest",
					Value: pipelineapi.ParamValue{
						Type:      pipelineapi.ParamTypeString,
						StringVal: imageDigest,
					},
				},
				pipelineapi.TaskRunResult{
					Name: "shp-image-size",
					Value: pipelineapi.ParamValue{
						Type:      pipelineapi.ParamTypeString,
						StringVal: "230",
					},
				},
				pipelineapi.TaskRunResult{
					Name: "shp-image-vulnerabilities",
					Value: pipelineapi.ParamValue{
						Type:      pipelineapi.ParamTypeString,
						StringVal: "",
					},
				})

			resources.UpdateBuildRunUsingTaskResults(ctx, br, tr.Status.Results, taskRunRequest)

			Expect(br.Status.Output.Digest).To(Equal(imageDigest))
			Expect(br.Status.Output.Size).To(Equal(int64(230)))
			Expect(br.Status.Output.Vulnerabilities).To(HaveLen(0))
		})

		It("should surface the TaskRun results emitting from source and output step", func() {
			commitSha := "0e0583421a5e4bf562ffe33f3651e16ba0c78591"
			imageDigest := "sha256:fe1b73cd25ac3f11dec752755e2"
			br.Status.BuildSpec = &build.BuildSpec{
				Source: &build.Source{
					Type: build.GitType,
					Git: &build.Git{
						URL: "https://github.com/shipwright-io/sample-go",
					},
				},
			}

			tr.Status.Results = append(tr.Status.Results,
				pipelineapi.TaskRunResult{
					Name: "shp-source-default-commit-sha",
					Value: pipelineapi.ParamValue{
						Type:      pipelineapi.ParamTypeString,
						StringVal: commitSha,
					},
				},
				pipelineapi.TaskRunResult{
					Name: "shp-source-default-commit-author",
					Value: pipelineapi.ParamValue{
						Type:      pipelineapi.ParamTypeString,
						StringVal: "foo bar",
					},
				},
				pipelineapi.TaskRunResult{
					Name: "shp-image-digest",
					Value: pipelineapi.ParamValue{
						Type:      pipelineapi.ParamTypeString,
						StringVal: imageDigest,
					},
				},
				pipelineapi.TaskRunResult{
					Name: "shp-image-size",
					Value: pipelineapi.ParamValue{
						Type:      pipelineapi.ParamTypeString,
						StringVal: "230",
					},
				})

			resources.UpdateBuildRunUsingTaskResults(ctx, br, tr.Status.Results, taskRunRequest)

			Expect(br.Status.Source).ToNot(BeNil())
			Expect(br.Status.Source.Git).ToNot(BeNil())
			Expect(br.Status.Source.Git.CommitSha).To(Equal(commitSha))
			Expect(br.Status.Source.Git.CommitAuthor).To(Equal("foo bar"))
			Expect(br.Status.Output.Digest).To(Equal(imageDigest))
			Expect(br.Status.Output.Size).To(Equal(int64(230)))
		})
	})
})

var _ = Describe("Multi-arch PipelineRun results to BuildRun", func() {
	var (
		ctx        context.Context
		br         *build.BuildRun
		pr         *pipelineapi.PipelineRun
		fakeClient *fakes.FakeClient
		platforms  []build.ImagePlatform
		taskRuns   map[string]*pipelineapi.TaskRun
	)

	newTaskRunWithResults := func(name string, succeeded corev1.ConditionStatus, digest string, size string, failMsg string, vulns ...string) *pipelineapi.TaskRun {
		tr := &pipelineapi.TaskRun{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
			Status: pipelineapi.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Type:    apis.ConditionSucceeded,
							Status:  succeeded,
							Message: failMsg,
						},
					},
				},
			},
		}
		if digest != "" {
			tr.Status.Results = append(tr.Status.Results, pipelineapi.TaskRunResult{
				Name:  "shp-image-digest",
				Value: pipelineapi.ParamValue{Type: pipelineapi.ParamTypeString, StringVal: digest},
			})
		}
		if size != "" {
			tr.Status.Results = append(tr.Status.Results, pipelineapi.TaskRunResult{
				Name:  "shp-image-size",
				Value: pipelineapi.ParamValue{Type: pipelineapi.ParamTypeString, StringVal: size},
			})
		}
		if len(vulns) > 0 && vulns[0] != "" {
			tr.Status.Results = append(tr.Status.Results, pipelineapi.TaskRunResult{
				Name:  "shp-image-vulnerabilities",
				Value: pipelineapi.ParamValue{Type: pipelineapi.ParamTypeString, StringVal: vulns[0]},
			})
		}
		return tr
	}

	BeforeEach(func() {
		ctx = context.Background()
		br = &build.BuildRun{
			ObjectMeta: metav1.ObjectMeta{Name: "test-br", Namespace: "default"},
		}
		platforms = []build.ImagePlatform{
			{OS: "linux", Arch: "amd64"},
			{OS: "linux", Arch: "arm64"},
		}
		taskRuns = make(map[string]*pipelineapi.TaskRun)

		fakeClient = &fakes.FakeClient{}
		fakeClient.GetStub = func(_ context.Context, key types.NamespacedName, obj client.Object, _ ...client.GetOption) error {
			if tr, ok := taskRuns[key.Name]; ok {
				tr.DeepCopyInto(obj.(*pipelineapi.TaskRun))
				return nil
			}
			return fmt.Errorf("TaskRun %s not found", key.Name)
		}
	})

	It("should populate PlatformResults for succeeded builds", func() {
		taskRuns["pr-build-linux-amd64"] = newTaskRunWithResults("pr-build-linux-amd64", corev1.ConditionTrue, "sha256:amd64digest", "100", "")
		taskRuns["pr-build-linux-arm64"] = newTaskRunWithResults("pr-build-linux-arm64", corev1.ConditionTrue, "sha256:arm64digest", "200", "")
		taskRuns["pr-assemble-index"] = newTaskRunWithResults("pr-assemble-index", corev1.ConditionTrue, "sha256:indexdigest", "500", "")

		pr = &pipelineapi.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{Name: "pr", Namespace: "default"},
			Status: pipelineapi.PipelineRunStatus{
				PipelineRunStatusFields: pipelineapi.PipelineRunStatusFields{
					ChildReferences: []pipelineapi.ChildStatusReference{
						{TypeMeta: runtime.TypeMeta{Kind: "TaskRun"}, Name: "pr-build-linux-amd64", PipelineTaskName: "build-linux-amd64"},
						{TypeMeta: runtime.TypeMeta{Kind: "TaskRun"}, Name: "pr-build-linux-arm64", PipelineTaskName: "build-linux-arm64"},
						{TypeMeta: runtime.TypeMeta{Kind: "TaskRun"}, Name: "pr-assemble-index", PipelineTaskName: "assemble-index"},
					},
				},
			},
		}

		resources.UpdateBuildRunWithMultiArchResults(ctx, br, pr, platforms, fakeClient)

		Expect(br.Status.PlatformResults).To(HaveLen(2))
		Expect(br.Status.PlatformResults[0].Platform).To(Equal(build.ImagePlatform{OS: "linux", Arch: "amd64"}))
		Expect(br.Status.PlatformResults[0].Status).To(Equal(build.PlatformBuildStatusSucceeded))
		Expect(br.Status.PlatformResults[0].Digest).To(Equal("sha256:amd64digest"))
		Expect(br.Status.PlatformResults[0].Size).To(Equal(int64(100)))

		Expect(br.Status.PlatformResults[1].Platform).To(Equal(build.ImagePlatform{OS: "linux", Arch: "arm64"}))
		Expect(br.Status.PlatformResults[1].Status).To(Equal(build.PlatformBuildStatusSucceeded))
		Expect(br.Status.PlatformResults[1].Digest).To(Equal("sha256:arm64digest"))
		Expect(br.Status.PlatformResults[1].Size).To(Equal(int64(200)))
		
		Expect(br.Status.ManifestDigest).To(Equal("sha256:indexdigest"))
		Expect(br.Status.Output).ToNot(BeNil())
		Expect(br.Status.Output.Digest).To(Equal("sha256:indexdigest"))
		Expect(br.Status.Output.Size).To(Equal(int64(0)))
		Expect(br.Status.Output.Vulnerabilities).To(BeEmpty())
	})

	It("should populate per-platform vulnerabilities and union them into Output.Vulnerabilities", func() {
		taskRuns["pr-build-linux-amd64"] = newTaskRunWithResults("pr-build-linux-amd64", corev1.ConditionTrue, "sha256:amd64digest", "100", "", "CVE-2024-0001:H,CVE-2024-0002:M")
		taskRuns["pr-build-linux-arm64"] = newTaskRunWithResults("pr-build-linux-arm64", corev1.ConditionTrue, "sha256:arm64digest", "200", "", "CVE-2024-0003:C")
		taskRuns["pr-assemble-index"] = newTaskRunWithResults("pr-assemble-index", corev1.ConditionTrue, "sha256:indexdigest", "", "")

		pr = &pipelineapi.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{Name: "pr", Namespace: "default"},
			Status: pipelineapi.PipelineRunStatus{
				PipelineRunStatusFields: pipelineapi.PipelineRunStatusFields{
					ChildReferences: []pipelineapi.ChildStatusReference{
						{TypeMeta: runtime.TypeMeta{Kind: "TaskRun"}, Name: "pr-build-linux-amd64", PipelineTaskName: "build-linux-amd64"},
						{TypeMeta: runtime.TypeMeta{Kind: "TaskRun"}, Name: "pr-build-linux-arm64", PipelineTaskName: "build-linux-arm64"},
						{TypeMeta: runtime.TypeMeta{Kind: "TaskRun"}, Name: "pr-assemble-index", PipelineTaskName: "assemble-index"},
					},
				},
			},
		}

		resources.UpdateBuildRunWithMultiArchResults(ctx, br, pr, platforms, fakeClient)

		Expect(br.Status.PlatformResults[0].Vulnerabilities).To(HaveLen(2))
		Expect(br.Status.PlatformResults[0].Vulnerabilities[0].ID).To(Equal("CVE-2024-0001"))
		Expect(br.Status.PlatformResults[0].Vulnerabilities[0].Severity).To(Equal(build.High))
		Expect(br.Status.PlatformResults[0].Vulnerabilities[1].ID).To(Equal("CVE-2024-0002"))

		Expect(br.Status.PlatformResults[1].Vulnerabilities).To(HaveLen(1))
		Expect(br.Status.PlatformResults[1].Vulnerabilities[0].ID).To(Equal("CVE-2024-0003"))
		Expect(br.Status.PlatformResults[1].Vulnerabilities[0].Severity).To(Equal(build.Critical))

		Expect(br.Status.Output.Vulnerabilities).To(HaveLen(3))
	})

	It("should report failed platform builds", func() {
		taskRuns["pr-build-linux-amd64"] = newTaskRunWithResults("pr-build-linux-amd64", corev1.ConditionTrue, "sha256:amd64digest", "100", "")
		taskRuns["pr-build-linux-arm64"] = newTaskRunWithResults("pr-build-linux-arm64", corev1.ConditionFalse, "", "", "no arm64 node available")

		pr = &pipelineapi.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{Name: "pr", Namespace: "default"},
			Status: pipelineapi.PipelineRunStatus{
				PipelineRunStatusFields: pipelineapi.PipelineRunStatusFields{
					ChildReferences: []pipelineapi.ChildStatusReference{
						{TypeMeta: runtime.TypeMeta{Kind: "TaskRun"}, Name: "pr-build-linux-amd64", PipelineTaskName: "build-linux-amd64"},
						{TypeMeta: runtime.TypeMeta{Kind: "TaskRun"}, Name: "pr-build-linux-arm64", PipelineTaskName: "build-linux-arm64"},
					},
				},
			},
		}

		resources.UpdateBuildRunWithMultiArchResults(ctx, br, pr, platforms, fakeClient)

		Expect(br.Status.PlatformResults).To(HaveLen(2))
		Expect(br.Status.PlatformResults[0].Status).To(Equal(build.PlatformBuildStatusSucceeded))
		Expect(br.Status.PlatformResults[1].Status).To(Equal(build.PlatformBuildStatusFailed))
		Expect(br.Status.PlatformResults[1].FailureMessage).To(ContainSubstring("no arm64 node available"))
	})

	It("should report Pending for platforms with no child TaskRun yet", func() {
		pr = &pipelineapi.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{Name: "pr", Namespace: "default"},
			Status: pipelineapi.PipelineRunStatus{
				PipelineRunStatusFields: pipelineapi.PipelineRunStatusFields{
					ChildReferences: []pipelineapi.ChildStatusReference{},
				},
			},
		}

		resources.UpdateBuildRunWithMultiArchResults(ctx, br, pr, platforms, fakeClient)

		Expect(br.Status.PlatformResults).To(HaveLen(2))
		Expect(br.Status.PlatformResults[0].Status).To(Equal(build.PlatformBuildStatusPending))
		Expect(br.Status.PlatformResults[1].Status).To(Equal(build.PlatformBuildStatusPending))
		Expect(br.Status.ManifestDigest).To(BeEmpty())
	})

	It("should report Running for in-progress builds", func() {
		runningTR := &pipelineapi.TaskRun{
			ObjectMeta: metav1.ObjectMeta{Name: "pr-build-linux-amd64", Namespace: "default"},
			Status: pipelineapi.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Type:   apis.ConditionSucceeded,
							Status: corev1.ConditionUnknown,
						},
					},
				},
			},
		}
		taskRuns["pr-build-linux-amd64"] = runningTR

		pr = &pipelineapi.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{Name: "pr", Namespace: "default"},
			Status: pipelineapi.PipelineRunStatus{
				PipelineRunStatusFields: pipelineapi.PipelineRunStatusFields{
					ChildReferences: []pipelineapi.ChildStatusReference{
						{TypeMeta: runtime.TypeMeta{Kind: "TaskRun"}, Name: "pr-build-linux-amd64", PipelineTaskName: "build-linux-amd64"},
					},
				},
			},
		}

		resources.UpdateBuildRunWithMultiArchResults(ctx, br, pr, platforms, fakeClient)

		Expect(br.Status.PlatformResults).To(HaveLen(2))
		Expect(br.Status.PlatformResults[0].Status).To(Equal(build.PlatformBuildStatusRunning))
		Expect(br.Status.PlatformResults[1].Status).To(Equal(build.PlatformBuildStatusPending))
	})

	It("should not populate anything for nil PipelineRun", func() {
		resources.UpdateBuildRunWithMultiArchResults(ctx, br, nil, platforms, fakeClient)
		Expect(br.Status.PlatformResults).To(BeNil())
		Expect(br.Status.ManifestDigest).To(BeEmpty())
	})

	It("should not populate anything for empty platforms", func() {
		emptyPR := &pipelineapi.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{Name: "pr", Namespace: "default"},
		}
		resources.UpdateBuildRunWithMultiArchResults(ctx, br, emptyPR, nil, fakeClient)
		Expect(br.Status.PlatformResults).To(BeNil())
		Expect(br.Status.ManifestDigest).To(BeEmpty())
	})
})
