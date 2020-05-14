package buildrun_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-developer/build/pkg/apis"
	build "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	buildrunctl "github.com/redhat-developer/build/pkg/controller/buildrun"
	"github.com/redhat-developer/build/pkg/controller/fakes"
	"github.com/redhat-developer/build/test"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	crc "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Reconcile BuildRun", func() {
	var (
		manager           *fakes.FakeManager
		reconciler        reconcile.Reconciler
		request           reconcile.Request
		client            *fakes.FakeClient
		ctl               test.Catalog
		buildSample       *build.Build
		buildRunSample    *build.BuildRun
		taskRunListSample *v1beta1.TaskRunList
		statusWriter      *fakes.FakeStatusWriter
		taskRunName       string
		buildName         string
		strategyName      string
	)

	// returns a reconcile.Request based on a buildRun instance
	newReconcileRequest := func(br *build.BuildRun) reconcile.Request {
		return reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      br.Name,
				Namespace: br.Namespace,
			},
		}
	}

	// Basic stubs that simulate the output of all client calls in the Reconciler logic.
	// This applies only for a Build and BuildRun client get.
	getClientStub := func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
		switch object := object.(type) {
		case *build.Build:
			buildSample.DeepCopyInto(object)
			return nil
		case *build.BuildRun:
			buildRunSample.DeepCopyInto(object)
			return nil
		}
		return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
	}

	BeforeEach(func() {
		taskRunName = "foobar-task"
		strategyName = "foobar-strategy"
		buildName = "foobar-build"

		// ensure resources are added to the Scheme
		// via the manager and initialize the fake Manager
		apis.AddToScheme(scheme.Scheme)
		manager = &fakes.FakeManager{}
		manager.GetSchemeReturns(scheme.Scheme)

		// initialize the fake client and let the
		// client know on the stubs when get calls are executed
		client = &fakes.FakeClient{}
		client.GetCalls(getClientStub)

		// initialize the fake status writer, this is needed for
		// all status updates during reconciliation
		statusWriter = &fakes.FakeStatusWriter{}
		client.StatusCalls(func() crc.StatusWriter { return statusWriter })
		manager.GetClientReturns(client)

		// init the Build resource, this never change throughout this test suite
		buildSample = ctl.DefaultBuild(buildName, strategyName, build.ClusterBuildStrategyKind)

	})

	// JustBeforeEach will always execute just before the It() specs,
	// this ensures that overrides on the BuildRun resource can happen under each
	// Context() BeforeEach() block
	JustBeforeEach(func() {
		reconciler = buildrunctl.NewReconciler(manager, controllerutil.SetControllerReference)
		request = newReconcileRequest(buildRunSample)

	})

	Describe("Reconciling", func() {
		Context("when a TaskRun exists", func() {
			BeforeEach(func() {
				// init a TaskRunList, we need this to fake the existance of a Tekton TaskRun
				taskRunListSample = ctl.DefaultTaskRunList(
					ctl.DefaultTaskRunWithStatus(taskRunName, corev1.ConditionTrue, "Succeeded"),
				)

				// init the BuildRun resource from catalog
				buildRunSample = ctl.DefaultBuildRun("foobar-buildrun", buildName)
			})

			It("Updates the BuildRun status", func() {
				// Stub that fakes the output when listing resources with the client
				client.ListCalls(func(contet context.Context, object runtime.Object, _ ...crc.ListOption) error {
					switch object := object.(type) {
					case *v1beta1.TaskRunList:
						taskRunListSample.DeepCopyInto(object)
					}
					return nil
				})

				// Stub that asserts the BuildRun status fields when
				// Status updates for a BuildRun take place
				statusCall := ctl.StubBuildRunStatus(
					"Succeeded",
					&taskRunName,
					corev1.ConditionTrue,
				)
				statusWriter.UpdateCalls(statusCall)

				// Assert for none errors while we exit the Reconcile
				// after updating the BuildRun status with the existing
				// TaskRun one
				result, err := reconciler.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
			})
		})

		Context("when a TaskRun does not exists", func() {
			var (
				saName           string
				emptyTaskRunName *string
				buildRunName     string
			)
			BeforeEach(func() {
				saName = "foobar-sa"
				buildRunName = "foobar-buildrun-with-sa"
				// override the BuildRun resource to use a BuildRun with a specified
				// serviceaccount
				buildRunSample = ctl.BuildRunWithSA(buildRunName, buildName, saName)
			})

			It("fails on creation due to missing service account", func() {

				// Stub that asserts the BuildRun status fields when
				// Status updates for a BuildRun take place
				statusCall := ctl.StubBuildRunStatus(
					fmt.Sprintf(" \"%s\" not found", saName),
					emptyTaskRunName,
					corev1.ConditionFalse,
				)
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
			})

			It("fails on creation due to missing namespaced buildstrategy", func() {
				// override the Build to use a namespaced BuildStragegy
				buildSample = ctl.DefaultBuild(buildName, strategyName, build.NamespacedBuildStrategyKind)

				// Override Stub get calls to include a service account
				// but none BuildStrategy
				client.GetCalls(ctl.StubBuildRunGetWithSA(
					buildSample,
					buildRunSample,
					ctl.DefaultServiceAccount(saName)),
				)

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(" \"%s\" not found", strategyName)))
			})

			It("fails on creation due to missing cluster buildstrategy", func() {
				// override the Build to use a cluster BuildStragegy
				buildSample = ctl.DefaultBuild(buildName, strategyName, build.ClusterBuildStrategyKind)

				// Override Stub get calls to include a service account
				// but none BuildStrategy
				client.GetCalls(ctl.StubBuildRunGetWithSA(
					buildSample,
					buildRunSample,
					ctl.DefaultServiceAccount(saName)),
				)

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(" \"%s\" not found", strategyName)))
			})

			It("fails on creation due to owner references errors", func() {
				// override the Build to use a namespaced BuildStragegy
				buildSample = ctl.DefaultBuild(buildName, strategyName, build.NamespacedBuildStrategyKind)

				// Override Stub get calls to include a service account
				// and BuildStrategies
				client.GetCalls(ctl.StubBuildRunGetWithSAandStrategies(
					buildSample,
					buildRunSample,
					ctl.DefaultServiceAccount(saName),
					ctl.DefaultClusterBuildStrategy(),
					ctl.DefaultNamespacedBuildStrategy()),
				)

				reconciler = buildrunctl.NewReconciler(manager,
					func(owner, object metav1.Object, scheme *runtime.Scheme) error {
						return fmt.Errorf("foobar error")
					})
				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("errors: foobar error"))
			})

			It("succeed creating a task from a namespaced buildstrategy", func() {
				// override the Build to use a namespaced BuildStragegy
				buildSample = ctl.DefaultBuild(buildName, strategyName, build.NamespacedBuildStrategyKind)

				// Override Stub get calls to include a service account
				// and BuildStrategies
				client.GetCalls(ctl.StubBuildRunGetWithSAandStrategies(
					buildSample,
					buildRunSample,
					ctl.DefaultServiceAccount(saName),
					ctl.DefaultClusterBuildStrategy(),
					ctl.DefaultNamespacedBuildStrategy()),
				)

				// Stub the create calls for a TaskRun
				client.CreateCalls(func(context context.Context, object runtime.Object, _ ...crc.CreateOption) error {
					switch object := object.(type) {
					case *v1beta1.TaskRun:
						ctl.DefaultTaskRunWithStatus(taskRunName, corev1.ConditionTrue, "Succeeded").DeepCopyInto(object)
					}
					return nil
				})

				_, err := reconciler.Reconcile(request)
				Expect(buildRunSample.Status.BuildSpec).ToNot(nil)
				Expect(err).ToNot(HaveOccurred())
			})

			It("succeed creating a task from a cluster buildstrategy", func() {
				// override the Build to use a cluster BuildStragegy
				buildSample = ctl.DefaultBuild(buildName, strategyName, build.ClusterBuildStrategyKind)

				// Override Stub get calls to include a service account
				// and BuildStrategies
				client.GetCalls(ctl.StubBuildRunGetWithSAandStrategies(
					buildSample,
					buildRunSample,
					ctl.DefaultServiceAccount(saName),
					ctl.DefaultClusterBuildStrategy(),
					ctl.DefaultNamespacedBuildStrategy()),
				)

				// Stub the create calls for a TaskRun
				client.CreateCalls(func(context context.Context, object runtime.Object, _ ...crc.CreateOption) error {
					switch object := object.(type) {
					case *v1beta1.TaskRun:
						ctl.DefaultTaskRunWithStatus(taskRunName, corev1.ConditionTrue, "Succeeded").DeepCopyInto(object)
					}
					return nil
				})

				_, err := reconciler.Reconcile(request)
				Expect(buildRunSample.Status.BuildSpec).To(Equal(buildSample.Spec))
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
