package build_test

import (
	"context"
	"fmt"

	"github.com/redhat-developer/build/test"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	build "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	"github.com/redhat-developer/build/pkg/config"
	buildController "github.com/redhat-developer/build/pkg/controller/build"
	"github.com/redhat-developer/build/pkg/controller/fakes"
	"github.com/redhat-developer/build/pkg/ctxlog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	crc "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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
		reconciler = buildController.NewReconciler(testCtx, config.NewDefaultConfig(), manager)
	})

	Describe("Reconcile", func() {
		Context("when spec output registry secret is specified", func() {
			It("fails when the secret does not exists", func() {

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

				statusCall := ctl.StubFunc(corev1.ConditionFalse, fmt.Sprintf("secret %s does not exist", registrySecret))
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("secret %s does not exist", registrySecret)))
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

				statusCall := ctl.StubFunc(corev1.ConditionFalse, fmt.Sprintf("clusterBuildStrategy %s does not exist", buildStrategyName))
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("clusterBuildStrategy %s does not exist", buildStrategyName)))
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

				statusCall := ctl.StubFunc(corev1.ConditionFalse, fmt.Sprintf("BuildStrategy %s does not exist in namespace %s", buildStrategyName, namespace))
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("BuildStrategy %s does not exist in namespace %s", buildStrategyName, namespace)))
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

				statusCall := ctl.StubFunc(corev1.ConditionFalse, fmt.Sprintf("BuildStrategy %s does not exist in namespace %s", buildStrategyName, namespace))
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("BuildStrategy %s does not exist in namespace %s", buildStrategyName, namespace)))

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
		Context("when the annotation build-run-deletion is defined", func() {
			var annotationFinalizer map[string]string

			JustBeforeEach(func() {
				annotationFinalizer = map[string]string{}
			})

			It("sets a finalizer if annotation equals true", func() {
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

				// override Build definition with one annotation
				annotationFinalizer[build.AnnotationBuildRunDeletion] = "true"
				buildSample = ctl.BuildWithCustomAnnotationAndFinalizer(
					buildName,
					namespace,
					buildStrategyName,
					annotationFinalizer,
					[]string{},
				)

				clientUpdateCalls := ctl.StubBuildUpdateWithFinalizers(build.BuildFinalizer)
				client.UpdateCalls(clientUpdateCalls)

				result, err := reconciler.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
			})

			It("removes a finalizer if annotation equals false and finalizer exists", func() {
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

				// override Build definition with one annotation
				annotationFinalizer[build.AnnotationBuildRunDeletion] = "false"
				buildSample = ctl.BuildWithCustomAnnotationAndFinalizer(
					buildName,
					namespace,
					buildStrategyName,
					annotationFinalizer,
					[]string{build.BuildFinalizer},
				)
				clientUpdateCalls := ctl.StubBuildUpdateWithoutFinalizers()
				client.UpdateCalls(clientUpdateCalls)

				result, err := reconciler.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
			})
		})
	})
})
