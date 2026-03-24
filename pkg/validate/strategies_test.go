// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	crc "sigs.k8s.io/controller-runtime/pkg/client"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/controller/fakes"
	. "github.com/shipwright-io/build/pkg/validate"
)

var _ = Describe("BuildStrategy", func() {
	var ctx context.Context
	var client *fakes.FakeClient

	BeforeEach(func() {
		ctx = context.TODO()
		client = &fakes.FakeClient{}
	})

	var sampleBuild = func(kind buildapi.BuildStrategyKind, name string) *buildapi.Build {
		return &buildapi.Build{
			Spec: buildapi.BuildSpec{
				Strategy: buildapi.Strategy{
					Kind: &kind,
					Name: name,
				},
			},
		}
	}

	Context("namespaced build strategy is used", func() {
		It("should pass when the referenced build strategy exists", func() {
			sample := sampleBuild(buildapi.NamespacedBuildStrategyKind, "buildkit")
			client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object, getOptions ...crc.GetOption) error {
				switch object := object.(type) {
				case *buildapi.BuildStrategy:
					(&buildapi.BuildStrategy{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: nn.Namespace,
							Name:      nn.Name},
					}).DeepCopyInto(object)
					return nil
				}

				return errors.NewNotFound(schema.GroupResource{}, "schema not found")
			})

			Expect(NewStrategies(client, sample).ValidatePath(ctx)).To(Succeed())
			Expect(sample.Status.Reason).To(BeNil())
		})

		It("should fail when the referenced build strategy does not exists", func() {
			sample := sampleBuild(buildapi.NamespacedBuildStrategyKind, "buildkit")
			client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object, getOptions ...crc.GetOption) error {
				return errors.NewNotFound(schema.GroupResource{}, "schema not found")
			})

			Expect(NewStrategies(client, sample).ValidatePath(ctx)).To(Succeed())
			Expect(*sample.Status.Reason).To(Equal(buildapi.BuildStrategyNotFound))
		})

		It("should error when there is an unexpected result", func() {
			sample := sampleBuild(buildapi.NamespacedBuildStrategyKind, "buildkit")
			client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object, getOptions ...crc.GetOption) error {
				return errors.NewInternalError(fmt.Errorf("monkey wrench"))
			})

			Expect(NewStrategies(client, sample).ValidatePath(ctx)).ToNot(Succeed())
		})
	})

	Context("cluster build strategy is used", func() {
		It("should pass when the referenced build strategy exists", func() {
			sample := sampleBuild(buildapi.ClusterBuildStrategyKind, "buildkit")
			client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object, getOptions ...crc.GetOption) error {
				switch object := object.(type) {
				case *buildapi.ClusterBuildStrategy:
					(&buildapi.ClusterBuildStrategy{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: nn.Namespace,
							Name:      nn.Name},
					}).DeepCopyInto(object)
					return nil
				}

				return errors.NewNotFound(schema.GroupResource{}, "schema not found")
			})

			Expect(NewStrategies(client, sample).ValidatePath(ctx)).To(Succeed())
			Expect(sample.Status.Reason).To(BeNil())
		})

		It("should fail when the referenced build strategy does not exists", func() {
			sample := sampleBuild(buildapi.ClusterBuildStrategyKind, "buildkit")
			client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object, getOptions ...crc.GetOption) error {
				return errors.NewNotFound(schema.GroupResource{}, "schema not found")
			})

			Expect(NewStrategies(client, sample).ValidatePath(ctx)).To(Succeed())
			Expect(*sample.Status.Reason).To(Equal(buildapi.ClusterBuildStrategyNotFound))
		})

		It("should error when there is an unexpected result", func() {
			sample := sampleBuild(buildapi.ClusterBuildStrategyKind, "buildkit")
			client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object, getOptions ...crc.GetOption) error {
				return errors.NewInternalError(fmt.Errorf("monkey wrench"))
			})

			Expect(NewStrategies(client, sample).ValidatePath(ctx)).ToNot(Succeed())
		})
	})

	Context("edge cases", func() {
		It("should default to namespace build strategy when kind is nil", func() {
			sample := &buildapi.Build{
				Spec: buildapi.BuildSpec{
					Strategy: buildapi.Strategy{
						Kind: nil,
						Name: "foobar",
					},
				},
			}

			client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object, getOptions ...crc.GetOption) error {
				switch object := object.(type) {
				case *buildapi.BuildStrategy:
					(&buildapi.BuildStrategy{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: nn.Namespace,
							Name:      nn.Name},
					}).DeepCopyInto(object)
					return nil
				}

				return errors.NewNotFound(schema.GroupResource{}, "schema not found")
			})

			Expect(NewStrategies(client, sample).ValidatePath(ctx)).To(Succeed())
			Expect(sample.Status.Reason).To(BeNil())
		})

		It("should fail validation if the strategy kind is unknown", func() {
			sample := sampleBuild("abc", "xyz")
			client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object, getOptions ...crc.GetOption) error {
				return errors.NewNotFound(schema.GroupResource{}, "schema not found")
			})

			Expect(NewStrategies(client, sample).ValidatePath(ctx)).To(Succeed())
			Expect(*sample.Status.Reason).To(Equal(buildapi.UnknownBuildStrategyKind))
		})
	})

	Context("stepResources validation", func() {
		It("should pass when stepResources references valid step names", func() {
			sample := sampleBuild(buildapi.NamespacedBuildStrategyKind, "buildkit")
			sample.Spec.Strategy.StepResources = []buildapi.StepResourceOverride{
				{Name: "build"},
			}

			client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object, getOptions ...crc.GetOption) error {
				switch object := object.(type) {
				case *buildapi.BuildStrategy:
					(&buildapi.BuildStrategy{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: nn.Namespace,
							Name:      nn.Name,
						},
						Spec: buildapi.BuildStrategySpec{
							Steps: []buildapi.Step{
								{Name: "build", Image: "busybox"},
							},
						},
					}).DeepCopyInto(object)
					return nil
				}
				return errors.NewNotFound(schema.GroupResource{}, "schema not found")
			})

			Expect(NewStrategies(client, sample).ValidatePath(ctx)).To(Succeed())
			Expect(sample.Status.Reason).To(BeNil())
		})

		It("should fail when stepResources references non-existent step name", func() {
			sample := sampleBuild(buildapi.NamespacedBuildStrategyKind, "buildkit")
			sample.Spec.Strategy.StepResources = []buildapi.StepResourceOverride{
				{Name: "non-existent-step"},
			}

			client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object, getOptions ...crc.GetOption) error {
				switch object := object.(type) {
				case *buildapi.BuildStrategy:
					(&buildapi.BuildStrategy{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: nn.Namespace,
							Name:      nn.Name,
						},
						Spec: buildapi.BuildStrategySpec{
							Steps: []buildapi.Step{
								{Name: "build", Image: "busybox"},
							},
						},
					}).DeepCopyInto(object)
					return nil
				}
				return errors.NewNotFound(schema.GroupResource{}, "schema not found")
			})

			Expect(NewStrategies(client, sample).ValidatePath(ctx)).To(Succeed())
			Expect(*sample.Status.Reason).To(Equal(buildapi.UndefinedStepResource))
			Expect(*sample.Status.Message).To(ContainSubstring("non-existent-step"))
		})

		It("should fail when stepResources has one valid and one invalid step", func() {
			sample := sampleBuild(buildapi.NamespacedBuildStrategyKind, "buildkit")
			sample.Spec.Strategy.StepResources = []buildapi.StepResourceOverride{
				{Name: "build"},
				{Name: "non-existent-step"},
			}

			client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object, getOptions ...crc.GetOption) error {
				switch object := object.(type) {
				case *buildapi.BuildStrategy:
					(&buildapi.BuildStrategy{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: nn.Namespace,
							Name:      nn.Name,
						},
						Spec: buildapi.BuildStrategySpec{
							Steps: []buildapi.Step{
								{Name: "build", Image: "busybox"},
								{Name: "push", Image: "busybox"},
							},
						},
					}).DeepCopyInto(object)
					return nil
				}
				return errors.NewNotFound(schema.GroupResource{}, "schema not found")
			})

			Expect(NewStrategies(client, sample).ValidatePath(ctx)).To(Succeed())
			Expect(*sample.Status.Reason).To(Equal(buildapi.UndefinedStepResource))
			Expect(*sample.Status.Message).To(ContainSubstring("non-existent-step"))
		})

		It("should pass when no stepResources are specified", func() {
			sample := sampleBuild(buildapi.NamespacedBuildStrategyKind, "buildkit")
			// No stepResources specified

			client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object, getOptions ...crc.GetOption) error {
				switch object := object.(type) {
				case *buildapi.BuildStrategy:
					(&buildapi.BuildStrategy{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: nn.Namespace,
							Name:      nn.Name,
						},
						Spec: buildapi.BuildStrategySpec{
							Steps: []buildapi.Step{
								{Name: "build", Image: "busybox"},
							},
						},
					}).DeepCopyInto(object)
					return nil
				}
				return errors.NewNotFound(schema.GroupResource{}, "schema not found")
			})

			Expect(NewStrategies(client, sample).ValidatePath(ctx)).To(Succeed())
			Expect(sample.Status.Reason).To(BeNil())
		})

		It("should validate stepResources for ClusterBuildStrategy", func() {
			sample := sampleBuild(buildapi.ClusterBuildStrategyKind, "buildkit")
			sample.Spec.Strategy.StepResources = []buildapi.StepResourceOverride{
				{Name: "non-existent-step"},
			}

			client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object, getOptions ...crc.GetOption) error {
				switch object := object.(type) {
				case *buildapi.ClusterBuildStrategy:
					(&buildapi.ClusterBuildStrategy{
						ObjectMeta: metav1.ObjectMeta{
							Name: nn.Name,
						},
						Spec: buildapi.BuildStrategySpec{
							Steps: []buildapi.Step{
								{Name: "build", Image: "busybox"},
								{Name: "push", Image: "busybox"},
							},
						},
					}).DeepCopyInto(object)
					return nil
				}
				return errors.NewNotFound(schema.GroupResource{}, "schema not found")
			})

			Expect(NewStrategies(client, sample).ValidatePath(ctx)).To(Succeed())
			Expect(*sample.Status.Reason).To(Equal(buildapi.UndefinedStepResource))
		})
	})
})
