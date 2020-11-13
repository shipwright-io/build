// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package build_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	crc "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	buildController "github.com/shipwright-io/build/pkg/controller/build"
	"github.com/shipwright-io/build/pkg/controller/fakes"
	"github.com/shipwright-io/build/pkg/ctxlog"
	"github.com/shipwright-io/build/test"
)

var _ = Describe("Reconcile Build", func() {
	var (
		manager                      *fakes.FakeManager
		reconciler                   reconcile.Reconciler
		request                      reconcile.Request
		buildSample                  *build.Build
		client                       *fakes.FakeClient
		ctl                          test.Catalog
		statusWriter                 *fakes.FakeStatusWriter
		registrySecret               string
		buildName                    string
		namespace, buildStrategyName string
	)

	BeforeEach(func() {
		registrySecret = "registry-secret"
		buildStrategyName = "buildah"
		namespace = "build-examples"
		buildName = "buildah-golang-build"

		// Fake the manager and get a reconcile Request
		manager = &fakes.FakeManager{}
		request = reconcile.Request{NamespacedName: types.NamespacedName{Name: buildName, Namespace: namespace}}

		// Fake the client GET calls when reconciling,
		// in order to get our Build CRD instance
		client = &fakes.FakeClient{}
		client.GetCalls(func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
			switch object := object.(type) {
			case *build.Build:
				buildSample.DeepCopyInto(object)
			default:
				return errors.NewNotFound(schema.GroupResource{}, "schema not found")
			}
			return nil
		})
		statusWriter = &fakes.FakeStatusWriter{}
		client.StatusCalls(func() crc.StatusWriter { return statusWriter })
		manager.GetClientReturns(client)
	})

	JustBeforeEach(func() {
		// Generate a Build CRD instance
		buildSample = ctl.BuildWithClusterBuildStrategy(buildName, namespace, buildStrategyName, registrySecret)
		// Reconcile
		testCtx := ctxlog.NewContext(context.TODO(), "fake-logger")
		reconciler = buildController.NewReconciler(testCtx, config.NewDefaultConfig(), manager, controllerutil.SetControllerReference)
	})

	Describe("Reconcile", func() {
		Context("when source secret is specified", func() {
			It("fails when the secret does not exist", func() {
				buildSample.Spec.Source.SecretRef = &corev1.LocalObjectReference{
					Name: "non-existing",
				}
				buildSample.Spec.Output.SecretRef = nil

				// Fake some client LIST calls and ensure we populate all
				// different resources we could get during reconciliation
				client.ListCalls(func(context context.Context, object runtime.Object, _ ...crc.ListOption) error {
					switch object := object.(type) {
					case *corev1.SecretList:
						list := ctl.FakeSecretList()
						list.DeepCopyInto(object)
					case *build.ClusterBuildStrategyList:
						list := ctl.ClusterBuildStrategyList(buildStrategyName)
						list.DeepCopyInto(object)
					}
					return nil
				})

				statusCall := ctl.StubFunc(corev1.ConditionFalse, fmt.Sprintf("%v: secret non-existing does not exist", buildController.SecretDoesNotExist))
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%v: secret non-existing does not exist", buildController.SecretDoesNotExist)))
			})

			It("succeeds when the secret exists", func() {
				buildSample.Spec.Source.SecretRef = &corev1.LocalObjectReference{
					Name: "existing",
				}
				buildSample.Spec.Output.SecretRef = nil

				// Fake some client LIST calls and ensure we populate all
				// different resources we could get during reconciliation
				client.ListCalls(func(context context.Context, object runtime.Object, _ ...crc.ListOption) error {
					switch object := object.(type) {
					case *corev1.SecretList:
						list := ctl.SecretList("existing")
						list.DeepCopyInto(object)
					case *build.ClusterBuildStrategyList:
						list := ctl.ClusterBuildStrategyList(buildStrategyName)
						list.DeepCopyInto(object)
					}
					return nil
				})

				statusCall := ctl.StubFunc(corev1.ConditionTrue, "Succeeded")
				statusWriter.UpdateCalls(statusCall)

				result, err := reconciler.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
				Expect(reconcile.Result{}).To(Equal(result))
			})
		})

		Context("when builder image secret is specified", func() {
			It("fails when the secret does not exist", func() {
				buildSample.Spec.BuilderImage = &build.Image{
					ImageURL: "busybox",
					SecretRef: &corev1.LocalObjectReference{
						Name: "non-existing",
					},
				}
				buildSample.Spec.Output.SecretRef = nil

				// Fake some client LIST calls and ensure we populate all
				// different resources we could get during reconciliation
				client.ListCalls(func(context context.Context, object runtime.Object, _ ...crc.ListOption) error {
					switch object := object.(type) {
					case *corev1.SecretList:
						list := ctl.FakeSecretList()
						list.DeepCopyInto(object)
					case *build.ClusterBuildStrategyList:
						list := ctl.ClusterBuildStrategyList(buildStrategyName)
						list.DeepCopyInto(object)
					}
					return nil
				})

				statusCall := ctl.StubFunc(corev1.ConditionFalse, fmt.Sprintf("%v: secret non-existing does not exist", buildController.SecretDoesNotExist))
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%v: secret non-existing does not exist", buildController.SecretDoesNotExist)))
			})

			It("succeeds when the secret exists", func() {
				buildSample.Spec.BuilderImage = &build.Image{
					ImageURL: "busybox",
					SecretRef: &corev1.LocalObjectReference{
						Name: "existing",
					},
				}
				buildSample.Spec.Output.SecretRef = nil

				// Fake some client LIST calls and ensure we populate all
				// different resources we could get during reconciliation
				client.ListCalls(func(context context.Context, object runtime.Object, _ ...crc.ListOption) error {
					switch object := object.(type) {
					case *corev1.SecretList:
						list := ctl.SecretList("existing")
						list.DeepCopyInto(object)
					case *build.ClusterBuildStrategyList:
						list := ctl.ClusterBuildStrategyList(buildStrategyName)
						list.DeepCopyInto(object)
					}
					return nil
				})

				statusCall := ctl.StubFunc(corev1.ConditionTrue, "Succeeded")
				statusWriter.UpdateCalls(statusCall)

				result, err := reconciler.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
				Expect(reconcile.Result{}).To(Equal(result))
			})
		})

		Context("when spec output registry secret is specified", func() {
			It("fails when the secret does not exist", func() {

				// Fake some client LIST calls and ensure we populate all
				// different resources we could get during reconciliation
				client.ListCalls(func(context context.Context, object runtime.Object, _ ...crc.ListOption) error {
					switch object := object.(type) {
					case *corev1.SecretList:
						list := ctl.FakeSecretList()
						list.DeepCopyInto(object)
					case *build.ClusterBuildStrategyList:
						list := ctl.ClusterBuildStrategyList(buildStrategyName)
						list.DeepCopyInto(object)
					}
					return nil
				})

				statusCall := ctl.StubFunc(corev1.ConditionFalse, fmt.Sprintf("%v: secret %s does not exist", buildController.SecretDoesNotExist, registrySecret))
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%v: secret %s does not exist", buildController.SecretDoesNotExist, registrySecret)))
			})
			It("succeed when the secret exists", func() {

				// Fake some client LIST calls and ensure we populate all
				// different resources we could get during reconciliation
				client.ListCalls(func(context context.Context, object runtime.Object, _ ...crc.ListOption) error {
					switch object := object.(type) {
					case *corev1.SecretList:
						list := ctl.SecretList(registrySecret)
						list.DeepCopyInto(object)
					case *build.ClusterBuildStrategyList:
						list := ctl.ClusterBuildStrategyList(buildStrategyName)
						list.DeepCopyInto(object)
					}
					return nil
				})
				statusCall := ctl.StubFunc(corev1.ConditionTrue, "Succeeded")
				statusWriter.UpdateCalls(statusCall)

				result, err := reconciler.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
				Expect(reconcile.Result{}).To(Equal(result))
			})
			It("fails when no any secret exists in namespace", func() {
				// Fake some client LIST calls and ensure we populate all
				// different resources we could get during reconciliation
				client.ListCalls(func(context context.Context, object runtime.Object, _ ...crc.ListOption) error {
					switch object := object.(type) {
					case *corev1.SecretList:
						list := ctl.FakeNoSecretListInNamespace()
						list.DeepCopyInto(object)
					case *build.ClusterBuildStrategyList:
						list := ctl.ClusterBuildStrategyList(buildStrategyName)
						list.DeepCopyInto(object)
					}
					return nil
				})
				statusCall := ctl.StubFunc(corev1.ConditionFalse, fmt.Sprintf("%v: there are no secrets in namespace %s", buildController.NoSecretsInNamespace, namespace))
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%v: there are no secrets in namespace %s", buildController.NoSecretsInNamespace, namespace)))
			})
		})

		Context("when source secret and output secret are specified", func() {
			It("fails when both secrets do not exist", func() {
				buildSample.Spec.Source.SecretRef = &corev1.LocalObjectReference{
					Name: "non-existing-source",
				}
				buildSample.Spec.Output.SecretRef = &corev1.LocalObjectReference{
					Name: "non-existing-output",
				}

				// Fake some client LIST calls and ensure we populate all
				// different resources we could get during reconciliation
				client.ListCalls(func(context context.Context, object runtime.Object, _ ...crc.ListOption) error {
					switch object := object.(type) {
					case *corev1.SecretList:
						list := ctl.FakeSecretList()
						list.DeepCopyInto(object)
					case *build.ClusterBuildStrategyList:
						list := ctl.ClusterBuildStrategyList(buildStrategyName)
						list.DeepCopyInto(object)
					}
					return nil
				})

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
				Expect(err.Error()).To(ContainSubstring("do not exist"))
				Expect(err.Error()).To(ContainSubstring("non-existing-source"))
				Expect(err.Error()).To(ContainSubstring("non-existing-output"))
			})
		})

		Context("when spec strategy ClusterBuildStrategy is specified", func() {
			It("fails when the strategy does not exists", func() {

				// Fake some client LIST calls and ensure we populate all
				// different resources we could get during reconciliation
				client.ListCalls(func(context context.Context, object runtime.Object, _ ...crc.ListOption) error {
					switch object := object.(type) {
					case *corev1.SecretList:
						list := ctl.SecretList(registrySecret)
						list.DeepCopyInto(object)
					case *build.ClusterBuildStrategyList:
						list := ctl.FakeClusterBuildStrategyList()
						list.DeepCopyInto(object)
					}
					return nil
				})

				statusCall := ctl.StubFunc(corev1.ConditionFalse, fmt.Sprintf("%v: clusterBuildStrategy %s does not exist", buildController.ClusterBuildStrategyDoesNotExist, buildStrategyName))
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%v: clusterBuildStrategy %s does not exist", buildController.ClusterBuildStrategyDoesNotExist, buildStrategyName)))
			})
			It("succeed when the strategy exists", func() {

				// Fake some client LIST calls and ensure we populate all
				// different resources we could get during reconciliation
				client.ListCalls(func(context context.Context, object runtime.Object, _ ...crc.ListOption) error {
					switch object := object.(type) {
					case *corev1.SecretList:
						list := ctl.SecretList(registrySecret)
						list.DeepCopyInto(object)
					case *build.ClusterBuildStrategyList:
						list := ctl.ClusterBuildStrategyList(buildStrategyName)
						list.DeepCopyInto(object)
					}
					return nil
				})

				statusCall := ctl.StubFunc(corev1.ConditionTrue, "Succeeded")
				statusWriter.UpdateCalls(statusCall)

				result, err := reconciler.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
				Expect(reconcile.Result{}).To(Equal(result))
			})
			It("fails when no any clusterStrategy exists", func() {

				// Fake some client LIST calls and ensure we populate all
				// different resources we could get during reconciliation
				client.ListCalls(func(context context.Context, object runtime.Object, _ ...crc.ListOption) error {
					switch object := object.(type) {
					case *corev1.SecretList:
						list := ctl.SecretList(registrySecret)
						list.DeepCopyInto(object)
					case *build.ClusterBuildStrategyList:
						list := ctl.FakeNoClusterBuildStrategyList()
						list.DeepCopyInto(object)
					}
					return nil
				})

				statusCall := ctl.StubFunc(corev1.ConditionFalse, fmt.Sprintf("%v: no ClusterBuildStrategies found", buildController.NoClusterBuildStrategyFound))
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%v: no ClusterBuildStrategies found", buildController.NoClusterBuildStrategyFound)))
			})
		})
		Context("when spec strategy BuildStrategy is specified", func() {
			JustBeforeEach(func() {
				buildStrategyName = "buildpacks-v3"
				buildName = "buildpack-nodejs-build-namespaced"
				// Override the buildSample to use a BuildStrategy instead of the Cluster one
				buildSample = ctl.BuildWithBuildStrategy(buildName, namespace, buildStrategyName)
			})

			It("fails when the strategy does not exists", func() {

				// Fake some client LIST calls and ensure we populate all
				// different resources we could get during reconciliation
				client.ListCalls(func(context context.Context, object runtime.Object, _ ...crc.ListOption) error {
					switch object := object.(type) {
					case *build.ClusterBuildStrategyList:
						list := ctl.FakeClusterBuildStrategyList()
						list.DeepCopyInto(object)
					case *build.BuildStrategyList:
						list := ctl.FakeBuildStrategyList()
						list.DeepCopyInto(object)
					}
					return nil
				})

				statusCall := ctl.StubFunc(corev1.ConditionFalse, fmt.Sprintf("%v: buildStrategy %s does not exist in namespace %s", buildController.BuildStrategyDoesNotExistInNamespace, buildStrategyName, namespace))
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%v: buildStrategy %s does not exist in namespace %s", buildController.BuildStrategyDoesNotExistInNamespace, buildStrategyName, namespace)))
			})
			It("succeed when the strategy exists", func() {

				// Fake some client LIST calls and ensure we populate all
				// different resources we could get during reconciliation
				client.ListCalls(func(context context.Context, object runtime.Object, _ ...crc.ListOption) error {
					switch object := object.(type) {
					case *build.ClusterBuildStrategyList:
						list := ctl.FakeClusterBuildStrategyList()
						list.DeepCopyInto(object)
					case *build.BuildStrategyList:
						list := ctl.BuildStrategyList(buildStrategyName, namespace)
						list.DeepCopyInto(object)
					}
					return nil
				})

				statusCall := ctl.StubFunc(corev1.ConditionTrue, "Succeeded")
				statusWriter.UpdateCalls(statusCall)

				result, err := reconciler.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
				Expect(reconcile.Result{}).To(Equal(result))
			})
			It("fails when no any strategy exists in namespace", func() {

				// Fake some client LIST calls and ensure we populate all
				// different resources we could get during reconciliation
				client.ListCalls(func(context context.Context, object runtime.Object, _ ...crc.ListOption) error {
					switch object := object.(type) {
					case *build.ClusterBuildStrategyList:
						list := ctl.FakeNoClusterBuildStrategyList()
						list.DeepCopyInto(object)
					case *build.BuildStrategyList:
						list := ctl.FakeNoBuildStrategyList()
						list.DeepCopyInto(object)
					}
					return nil
				})

				statusCall := ctl.StubFunc(corev1.ConditionFalse, fmt.Sprintf("%v: none BuildStrategies found in namespace %s", buildController.NoneBuildStrategyFoundInNamespace, namespace))
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%v: none BuildStrategies found in namespace %s", buildController.NoneBuildStrategyFoundInNamespace, namespace)))
			})
		})
		Context("when spec strategy kind is not specified", func() {
			JustBeforeEach(func() {
				buildStrategyName = "kaniko"
				buildName = "kaniko-example-build-namespaced"
				// Override the buildSample to use a BuildStrategy instead of the Cluster one, although the build strategy kind is nil
				buildSample = ctl.BuildWithNilBuildStrategyKind(buildName, namespace, buildStrategyName)
			})
			It("default to BuildStrategy and fails when the strategy does not exists", func() {
				// Fake some client LIST calls and ensure we populate all
				// different resources we could get during reconciliation
				client.ListCalls(func(context context.Context, object runtime.Object, _ ...crc.ListOption) error {
					switch object := object.(type) {
					case *build.ClusterBuildStrategyList:
						list := ctl.FakeClusterBuildStrategyList()
						list.DeepCopyInto(object)
					case *build.BuildStrategyList:
						list := ctl.FakeBuildStrategyList()
						list.DeepCopyInto(object)
					}
					return nil
				})

				statusCall := ctl.StubFunc(corev1.ConditionFalse, fmt.Sprintf("%v: buildStrategy %s does not exist in namespace %s", buildController.BuildStrategyDoesNotExistInNamespace, buildStrategyName, namespace))
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%v: buildStrategy %s does not exist in namespace %s", buildController.BuildStrategyDoesNotExistInNamespace, buildStrategyName, namespace)))

			})
			It("default to BuildStrategy and succeed if the strategy exists", func() {
				// Fake some client LIST calls and ensure we populate all
				// different resources we could get during reconciliation
				client.ListCalls(func(context context.Context, object runtime.Object, _ ...crc.ListOption) error {
					switch object := object.(type) {
					case *build.ClusterBuildStrategyList:
						list := ctl.FakeClusterBuildStrategyList()
						list.DeepCopyInto(object)
					case *build.BuildStrategyList:
						list := ctl.BuildStrategyList(buildStrategyName, namespace)
						list.DeepCopyInto(object)
					}
					return nil
				})

				statusCall := ctl.StubFunc(corev1.ConditionTrue, "Succeeded")
				statusWriter.UpdateCalls(statusCall)

				result, err := reconciler.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
				Expect(reconcile.Result{}).To(Equal(result))
			})
		})

		Context("Validate all build error code is correct", func() {
			It("1001 error code should represent listing secrets in namespace failed", func() {
				errorName, err := ctl.ValidateError(1001)
				Expect(err).ToNot(HaveOccurred())
				Expect(errorName).To(Equal("ListSecretInNamespaceFailed"))
			})
			It("1002 error code should represent there are no secrets in namespace", func() {
				errorName, err := ctl.ValidateError(1002)
				Expect(err).ToNot(HaveOccurred())
				Expect(errorName).To(Equal("NoSecretsInNamespace"))
			})
			It("1003 error code should represent secrets do not exist", func() {
				errorName, err := ctl.ValidateError(1003)
				Expect(err).ToNot(HaveOccurred())
				Expect(errorName).To(Equal("SecretsDoNotExist"))
			})
			It("1004 error code should represent secret does not exist", func() {
				errorName, err := ctl.ValidateError(1004)
				Expect(err).ToNot(HaveOccurred())
				Expect(errorName).To(Equal("SecretDoesNotExist"))
			})
			It("1005 error code should represent unknown strategy", func() {
				errorName, err := ctl.ValidateError(1005)
				Expect(err).ToNot(HaveOccurred())
				Expect(errorName).To(Equal("UnknownStrategy"))
			})
			It("1006 error code should represent listing BuildStrategies in ns failed", func() {
				errorName, err := ctl.ValidateError(1006)
				Expect(err).ToNot(HaveOccurred())
				Expect(errorName).To(Equal("ListBuildStrategyInNamespaceFailed"))
			})
			It("1007 error code should represent none BuildStrategies found in namespace", func() {
				errorName, err := ctl.ValidateError(1007)
				Expect(err).ToNot(HaveOccurred())
				Expect(errorName).To(Equal("NoneBuildStrategyFoundInNamespace"))
			})
			It("1008 error code should represent buildStrategy does not exist in namespace", func() {
				errorName, err := ctl.ValidateError(1008)
				Expect(err).ToNot(HaveOccurred())
				Expect(errorName).To(Equal("BuildStrategyDoesNotExistInNamespace"))
			})
			It("1009 error code should represent listing ClusterBuildStrategies failed", func() {
				errorName, err := ctl.ValidateError(1009)
				Expect(err).ToNot(HaveOccurred())
				Expect(errorName).To(Equal("ListClusterBuildStrategyFailed"))
			})
			It("1010 error code should represent no ClusterBuildStrategies found", func() {
				errorName, err := ctl.ValidateError(1010)
				Expect(err).ToNot(HaveOccurred())
				Expect(errorName).To(Equal("NoClusterBuildStrategyFound"))
			})
			It("1011 error code should represent clusterBuildStrategy does not exist", func() {
				errorName, err := ctl.ValidateError(1011)
				Expect(err).ToNot(HaveOccurred())
				Expect(errorName).To(Equal("ClusterBuildStrategyDoesNotExist"))
			})
			It("1012 error code should represent the property 'spec.runtime.paths' must not be empty", func() {
				errorName, err := ctl.ValidateError(1012)
				Expect(err).ToNot(HaveOccurred())
				Expect(errorName).To(Equal("RuntimePathsCanNotBeEmpty"))
			})
			It("1013 error code should represent unexpected error when trying to set the ownerreference", func() {
				errorName, err := ctl.ValidateError(1013)
				Expect(err).ToNot(HaveOccurred())
				Expect(errorName).To(Equal("SetOwnerReferenceFailed"))
			})
		})

	})
})
