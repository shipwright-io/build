package buildrun_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-developer/build/pkg/apis"
	build "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	"github.com/redhat-developer/build/pkg/config"
	buildrunctl "github.com/redhat-developer/build/pkg/controller/buildrun"
	"github.com/redhat-developer/build/pkg/controller/fakes"
	"github.com/redhat-developer/build/pkg/ctxlog"
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
		manager        *fakes.FakeManager
		reconciler     reconcile.Reconciler
		request        reconcile.Request
		client         *fakes.FakeClient
		ctl            test.Catalog
		buildSample    *build.Build
		buildRunSample *build.BuildRun
		taskRunSample  *v1beta1.TaskRun
		statusWriter   *fakes.FakeStatusWriter
		taskRunName    string
		buildName      string
		strategyName   string
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
		case *v1beta1.TaskRun:
			taskRunSample.DeepCopyInto(object)
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
		testCtx := ctxlog.NewContext(context.TODO(), "fake-logger")
		reconciler = buildrunctl.NewReconciler(testCtx, config.NewDefaultConfig(), manager, controllerutil.SetControllerReference)
		request = newReconcileRequest(buildRunSample)

	})

	Describe("Reconciling", func() {
		Context("when a TaskRun exists", func() {
			BeforeEach(func() {
				// init a TaskRun, we need this to fake the existance of a Tekton TaskRun
				taskRunSample = ctl.DefaultTaskRunWithStatus("foobar-buildrun", corev1.ConditionTrue, "Succeeded")

				// init the BuildRun resource from catalog
				buildRunSample = ctl.DefaultBuildRun("foobar-buildrun", buildName)
			})

			It("Updates the BuildRun status", func() {

				// Stub that asserts the BuildRun status fields when
				// Status updates for a BuildRun take place
				statusCall := ctl.StubBuildRunStatus(
					"Succeeded",
					&taskRunName,
					corev1.ConditionTrue,
					buildSample.Spec,
				)
				statusWriter.UpdateCalls(statusCall)

				// Stub that asserts the BuildRun label fields when
				// Label updates for a BuildRun take place
				labelCall := ctl.StubBuildRunLabel(buildSample)
				statusWriter.UpdateCalls(labelCall)

				// Assert for none errors while we exit the Reconcile
				// after updating the BuildRun status with the existing
				// TaskRun one
				result, err := reconciler.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
			})

			It("deletes a generated service account when the task run ends", func() {

				// setup a buildrun to use a generated service account
				buildRunSample = ctl.BuildRunWithSAGenerate(buildRunSample.Name, buildName)
				buildRunSample.Labels = make(map[string]string)
				buildRunSample.Labels[build.LabelBuild] = buildName

				// Override Stub get calls to include a service account
				client.GetCalls(ctl.StubBuildRunGetWithTaskRunAndSA(
					buildSample,
					buildRunSample,
					taskRunSample,
					ctl.DefaultServiceAccount(buildRunSample.Name+"-sa")),
				)

				// Call the reconciler
				_, err := reconciler.Reconcile(request)

				// Expect no error
				Expect(err).ToNot(HaveOccurred())

				// Expect one delete call for the service account
				Expect(client.DeleteCallCount()).To(Equal(1))
				_, obj, _ := client.DeleteArgsForCall(0)
				serviceAccount, castSuccessful := obj.(*corev1.ServiceAccount)
				Expect(castSuccessful).To(BeTrue())
				Expect(serviceAccount.Name).To(Equal(buildRunSample.Name + "-sa"))
				Expect(serviceAccount.Namespace).To(Equal(buildRunSample.Namespace))
			})
		})
		Context("when a TaskRun exists and have conditions", func() {
			BeforeEach(func() {

				// init the BuildRun resource from catalog
				buildRunSample = ctl.DefaultBuildRun("foobar-buildrun", buildName)
			})

			// Docs on the TaskRun conditions can be found here
			// https://github.com/tektoncd/pipeline/blob/master/docs/taskruns.md#monitoring-execution-status
			It("updates the BuildRun status with a PENDING reason", func() {

				taskRunSample = ctl.DefaultTaskRunWithStatus("foobar-task", corev1.ConditionUnknown, "Pending")

				// Stub that asserts the BuildRun status fields when
				// Status updates for a BuildRun take place
				statusCall := ctl.StubBuildRunStatus(
					"Pending",
					&taskRunName,
					corev1.ConditionUnknown,
					buildSample.Spec,
				)
				statusWriter.UpdateCalls(statusCall)

				// Assert for none errors while we exit the Reconcile
				// after updating the BuildRun status with the existing
				// TaskRun one
				result, err := reconciler.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
			})

			It("updates the BuildRun status with a RUNNING reason", func() {

				taskRunSample = ctl.DefaultTaskRunWithStatus("foobar-task", corev1.ConditionUnknown, "Running")

				statusCall := ctl.StubBuildRunStatus(
					"Running",
					&taskRunName,
					corev1.ConditionUnknown,
					buildSample.Spec,
				)
				statusWriter.UpdateCalls(statusCall)

				result, err := reconciler.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
			})

			It("updates the BuildRun status with a SUCCEEDED reason", func() {

				taskRunSample = ctl.DefaultTaskRunWithStatus("foobar-task", corev1.ConditionTrue, "Succeeded")

				statusCall := ctl.StubBuildRunStatus(
					"Succeeded",
					&taskRunName,
					corev1.ConditionTrue,
					buildSample.Spec,
				)
				statusWriter.UpdateCalls(statusCall)

				result, err := reconciler.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
			})

			It("updates the BuildRun status when a FALSE status occurs", func() {

				taskRunSample = ctl.DefaultTaskRunWithFalseStatus("foobar-task")

				// Based on the current buildRun controller, if the TaskRun condition.Status
				// is FALSE, we will then populate our buildRun.Status.Reason with the
				// TaskRun condition.Message, rather than the condition.Reason
				statusCall := ctl.StubBuildRunStatus(
					"some message",
					&taskRunName,
					corev1.ConditionFalse,
					buildSample.Spec,
				)
				statusWriter.UpdateCalls(statusCall)

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

				// init a TaskRun, we need this to fake the existance of a Tekton TaskRun
				// taskRunSample = ctl.DefaultTaskRunWithStatus(buildRunName, corev1.ConditionTrue, "Succeeded")

				// override the BuildRun resource to use a BuildRun with a specified
				// serviceaccount
				buildRunSample = ctl.BuildRunWithSA(buildRunName, buildName, saName)
			})

			It("fails on creation due to missing service account", func() {

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

				client.GetCalls(getClientStub)

				// Stub that asserts the BuildRun status fields when
				// Status updates for a BuildRun take place
				statusCall := ctl.StubBuildRunStatus(
					fmt.Sprintf(" \"%s\" not found", saName),
					emptyTaskRunName,
					corev1.ConditionFalse,
					buildSample.Spec,
				)
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
			})

			It("fails on creation due to missing namespaced buildstrategy", func() {
				// override the Build to use a namespaced BuildStrategy
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
				// override the Build to use a cluster BuildStrategy
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

			It("only generates the service account once if the task run cannot be created", func() {
				// override the Build to use a cluster BuildStrategy
				buildSample = ctl.DefaultBuild(buildName, strategyName, build.ClusterBuildStrategyKind)

				// override buildrun to use a generated service account
				buildRunName = "foobar-buildrun-sa-generate"
				buildRunSample = ctl.BuildRunWithSAGenerate(buildRunName, buildName)
				buildRunSample.Labels = make(map[string]string)
				buildRunSample.Labels[build.LabelBuild] = buildName

				// Override stub calls to include only build and buildrun
				client.GetCalls(ctl.StubBuildRunGetWithoutSA(buildSample, buildRunSample))

				// Call the reconciler
				_, err := reconciler.Reconcile(request)

				// Expect an error stating that the strategy does not exist
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(" \"%s\" not found", strategyName)))

				// Expect one create call (service account)
				Expect(client.CreateCallCount()).To(Equal(1))
				_, obj, _ := client.CreateArgsForCall(0)
				serviceAccount, castSuccessful := obj.(*corev1.ServiceAccount)
				Expect(castSuccessful).To(BeTrue())

				// Expect zero update calls
				Expect(client.UpdateCallCount()).To(Equal(0))

				// Change the Get stub to also return the service account
				client.GetCalls(ctl.StubBuildRunGetWithSA(buildSample, buildRunSample, serviceAccount))

				// Call the reconciler again
				_, err = reconciler.Reconcile(request)

				// Expect an error stating that the strategy does not exist
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(" \"%s\" not found", strategyName)))

				// Expect no more create call because service account already existed
				Expect(client.CreateCallCount()).To(Equal(1))

				// Expect zero update calls because service account already existed and did not need to be modified
				Expect(client.UpdateCallCount()).To(Equal(0))
			})

			It("fails on creation due to owner references errors", func() {
				// override the Build to use a namespaced BuildStrategy
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

				testCtx := ctxlog.NewContext(context.TODO(), "fake-logger")
				reconciler = buildrunctl.NewReconciler(testCtx, config.NewDefaultConfig(), manager,
					func(owner, object metav1.Object, scheme *runtime.Scheme) error {
						return fmt.Errorf("foobar error")
					})
				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("errors: foobar error"))
			})

			It("succeed creating a task from a namespaced buildstrategy", func() {
				// override the Build to use a namespaced BuildStrategy
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
				Expect(err).ToNot(HaveOccurred())

				Expect(client.CreateCallCount()).To(Equal(1))
			})

			It("succeed creating a task from a cluster buildstrategy", func() {
				// override the Build to use a cluster BuildStrategy
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
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("When a build is not ready", func() {
			It("stops creation when a FALSE registered status of the build occurs", func() {
				// Init the Build with registered status false
				buildSample = ctl.DefaultBuildWithFalseRegistered(buildName, strategyName, build.ClusterBuildStrategyKind)
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

				client.GetCalls(getClientStub)
				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("reason: something bad happened"))
			})
		})

	})
})
