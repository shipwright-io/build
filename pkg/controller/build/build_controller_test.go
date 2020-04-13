package build_test

import (
	"context"
	"fmt"

	"github.com/redhat-developer/build/test"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	build "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	buildController "github.com/redhat-developer/build/pkg/controller/build"
	"github.com/redhat-developer/build/pkg/controller/fakes"
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
		reconciler = buildController.NewReconciler(manager)
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

				statusCall := ctl.StubFunc(corev1.ConditionFalse)
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

				statusCall := ctl.StubFunc(corev1.ConditionTrue)
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

				statusCall := ctl.StubFunc(corev1.ConditionFalse)
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

				statusCall := ctl.StubFunc(corev1.ConditionTrue)
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

				statusCall := ctl.StubFunc(corev1.ConditionFalse)
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

				statusCall := ctl.StubFunc(corev1.ConditionTrue)
				statusWriter.UpdateCalls(statusCall)

				result, err := reconciler.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
				Expect(reconcile.Result{}).To(Equal(result))
			})
		})
	})
})
