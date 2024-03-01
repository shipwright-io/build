// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/shipwright-io/build/pkg/apis/build/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/pointer"
)

var _ = Describe("Checking BuildRun Status fields", func() {
	Context("Verifying BuildRun status source results", func() {
		var (
			strategyName string
			buildRunName string
		)

		BeforeEach(func() {
			id := rand.String(5)
			strategyName = fmt.Sprintf("cbs-%s", id)
			buildRunName = fmt.Sprintf("buildrun-%s", id)
		})

		AfterEach(func() {
			tb.DeleteBR(buildRunName)
			tb.DeleteClusterBuildStrategy(strategyName)
		})

		It("should have the correct source timestamp for Git sources", func() {
			// Use an empty strategy to only have the source step
			strategy := tb.Catalog.ClusterBuildStrategy(strategyName)
			Expect(tb.CreateClusterBuildStrategy(strategy)).To(Succeed())
			cbs := v1beta1.ClusterBuildStrategyKind
			// Setup BuildRun with fixed revision where we know the commit details
			Expect(tb.CreateBR(&v1beta1.BuildRun{
				ObjectMeta: metav1.ObjectMeta{Name: buildRunName},
				Spec: v1beta1.BuildRunSpec{
					Build: v1beta1.ReferencedBuild{
						Spec: &v1beta1.BuildSpec{
							Source: &v1beta1.Source{
								Type: v1beta1.GitType,
								Git: &v1beta1.Git{
									URL:      "https://github.com/shipwright-io/sample-go",
									Revision: pointer.String("v0.1.0"),
								},
							},
							Strategy: v1beta1.Strategy{
								Kind: &cbs,
								Name: strategy.Name,
							},
						},
					},
				},
			})).ToNot(HaveOccurred())

			buildRun, err := tb.GetBRTillCompletion(buildRunName)
			Expect(err).ToNot(HaveOccurred())
			Expect(buildRun).ToNot(BeNil())

			Expect(buildRun.Status.Source).ToNot(BeNil())
			Expect(buildRun.Status.Source.Timestamp).ToNot(BeNil())
			Expect(buildRun.Status.Source.Timestamp.Time).To(BeTemporally("==", time.Unix(1619426578, 0)))
		})

		It("should have the correct source timestamp for Bundle sources", func() {
			// Use an empty strategy to only have the source step
			strategy := tb.Catalog.ClusterBuildStrategy(strategyName)
			Expect(tb.CreateClusterBuildStrategy(strategy)).To(Succeed())
			cbs := v1beta1.ClusterBuildStrategyKind
			// Setup BuildRun with fixed image sha where we know the timestamp details
			Expect(tb.CreateBR(&v1beta1.BuildRun{

				ObjectMeta: metav1.ObjectMeta{Name: buildRunName},
				Spec: v1beta1.BuildRunSpec{
					Build: v1beta1.ReferencedBuild{
						Spec: &v1beta1.BuildSpec{
							Strategy: v1beta1.Strategy{
								Kind: &cbs,
								Name: strategy.Name,
							},
							Source: &v1beta1.Source{
								Type: v1beta1.OCIArtifactType,
								OCIArtifact: &v1beta1.OCIArtifact{
									Image: "ghcr.io/shipwright-io/sample-go/source-bundle@sha256:9a5e264c19980387b8416e0ffa7460488272fb8a6a56127c657edaa2682daab2",
								},
							},
						},
					},
				},
			})).ToNot(HaveOccurred())

			buildRun, err := tb.GetBRTillCompletion(buildRunName)
			Expect(err).ToNot(HaveOccurred())
			Expect(buildRun).ToNot(BeNil())

			Expect(buildRun.Status.Source).ToNot(BeNil())
			Expect(buildRun.Status.Source.Timestamp).ToNot(BeNil())
			Expect(buildRun.Status.Source.Timestamp.Time).To(BeTemporally("==", time.Unix(1691650396, 0)))
		})
	})
})
