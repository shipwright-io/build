// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
	"github.com/shipwright-io/build/test"

	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("TaskRun results to BuildRun", func() {
	var ctl test.Catalog

	Context("when a BuildRun complete successfully", func() {
		var (
			taskRunRequest reconcile.Request
			br             *build.BuildRun
			tr             *pipelinev1beta1.TaskRun
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
			br.Status.BuildSpec.Source.URL = pointer.StringPtr("https://github.com/shipwright-io/sample-go")

			tr.Status.TaskRunResults = append(tr.Status.TaskRunResults,
				pipelinev1beta1.TaskRunResult{
					Name:  "shp-source-default-commit-sha",
					Value: commitSha,
				},
				pipelinev1beta1.TaskRunResult{
					Name:  "shp-source-default-commit-author",
					Value: "foo bar",
				})

			resources.UpdateBuildRunUsingTaskResults(ctx, br, tr.Status.TaskRunResults, taskRunRequest)

			Expect(len(br.Status.Sources)).To(Equal(1))
			Expect(br.Status.Sources[0].Git.CommitSha).To(Equal(commitSha))
			Expect(br.Status.Sources[0].Git.CommitAuthor).To(Equal("foo bar"))
		})

		It("should surface the TaskRun results emitting from default(bundle) source step", func() {
			bundleImageDigest := "sha256:fe1b73cd25ac3f11dec752755e2"
			br.Status.BuildSpec.Source.BundleContainer = &build.BundleContainer{
				Image: "quay.io/shipwright/source-bundle:latest",
			}

			tr.Status.TaskRunResults = append(tr.Status.TaskRunResults,
				pipelinev1beta1.TaskRunResult{
					Name:  "shp-source-default-image-digest",
					Value: bundleImageDigest,
				})

			resources.UpdateBuildRunUsingTaskResults(ctx, br, tr.Status.TaskRunResults, taskRunRequest)

			Expect(len(br.Status.Sources)).To(Equal(1))
			Expect(br.Status.Sources[0].Bundle.Digest).To(Equal(bundleImageDigest))
		})

		It("should surface the TaskRun results emitting from output step", func() {
			imageDigest := "sha256:fe1b73cd25ac3f11dec752755e2"

			tr.Status.TaskRunResults = append(tr.Status.TaskRunResults,
				pipelinev1beta1.TaskRunResult{
					Name:  "shp-image-digest",
					Value: imageDigest,
				},
				pipelinev1beta1.TaskRunResult{
					Name:  "shp-image-size",
					Value: "230",
				})

			resources.UpdateBuildRunUsingTaskResults(ctx, br, tr.Status.TaskRunResults, taskRunRequest)

			Expect(br.Status.Output.Digest).To(Equal(imageDigest))
			Expect(br.Status.Output.Size).To(Equal(int64(230)))
		})

		It("should surface the TaskRun results emitting from source and output step", func() {
			commitSha := "0e0583421a5e4bf562ffe33f3651e16ba0c78591"
			imageDigest := "sha256:fe1b73cd25ac3f11dec752755e2"
			br.Status.BuildSpec.Source.URL = pointer.StringPtr("https://github.com/shipwright-io/sample-go")

			tr.Status.TaskRunResults = append(tr.Status.TaskRunResults,
				pipelinev1beta1.TaskRunResult{
					Name:  "shp-source-default-commit-sha",
					Value: commitSha,
				},
				pipelinev1beta1.TaskRunResult{
					Name:  "shp-source-default-commit-author",
					Value: "foo bar",
				},
				pipelinev1beta1.TaskRunResult{
					Name:  "shp-image-digest",
					Value: imageDigest,
				},
				pipelinev1beta1.TaskRunResult{
					Name:  "shp-image-size",
					Value: "230",
				})

			resources.UpdateBuildRunUsingTaskResults(ctx, br, tr.Status.TaskRunResults, taskRunRequest)

			Expect(len(br.Status.Sources)).To(Equal(1))
			Expect(br.Status.Sources[0].Git.CommitSha).To(Equal(commitSha))
			Expect(br.Status.Sources[0].Git.CommitAuthor).To(Equal("foo bar"))
			Expect(br.Status.Output.Digest).To(Equal(imageDigest))
			Expect(br.Status.Output.Size).To(Equal(int64(230)))
		})
	})
})
