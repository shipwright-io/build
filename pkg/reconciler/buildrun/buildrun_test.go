// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package buildrun_test

import (
	"context"
	"fmt"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	knativeapi "knative.dev/pkg/apis"
	knativev1beta1 "knative.dev/pkg/apis/duck/v1beta1"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	crc "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/shipwright-io/build/pkg/apis"
	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/controller/fakes"
	"github.com/shipwright-io/build/pkg/ctxlog"
	buildrunctl "github.com/shipwright-io/build/pkg/reconciler/buildrun"
	"github.com/shipwright-io/build/test"
)

var _ = Describe("Reconcile BuildRun", func() {
	var (
		manager                                                *fakes.FakeManager
		reconciler                                             reconcile.Reconciler
		taskRunRequest, buildRunRequest                        reconcile.Request
		client                                                 *fakes.FakeClient
		ctl                                                    test.Catalog
		buildSample                                            *build.Build
		buildRunSample                                         *build.BuildRun
		taskRunSample                                          *v1beta1.TaskRun
		statusWriter                                           *fakes.FakeStatusWriter
		fakeBuildStrategyKind                                  build.BuildStrategyKind
		taskRunName, buildRunName, buildName, strategyName, ns string
	)

	// returns a reconcile.Request based on an resource name and namespace
	newReconcileRequest := func(name string, ns string) reconcile.Request {
		return reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      name,
				Namespace: ns,
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
		strategyName = "foobar-strategy"
		buildName = "foobar-build"
		buildRunName = "foobar-buildrun"
		taskRunName = "foobar-buildrun-p8nts"
		ns = "default"

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
		// l := ctxlog.NewLogger("buildrun-controller-test")
		// testCtx := ctxlog.NewParentContext(l)
		testCtx := ctxlog.NewContext(context.TODO(), "fake-logger")
		reconciler = buildrunctl.NewReconciler(testCtx, config.NewDefaultConfig(), manager, controllerutil.SetControllerReference)
	})

	Describe("Reconciling", func() {
		Context("from an existing TaskRun resource", func() {
			BeforeEach(func() {

				// Generate a new Reconcile Request using the existing TaskRun name and namespace
				taskRunRequest = newReconcileRequest(taskRunName, ns)

				// initialize a TaskRun, we need this to fake the existence of a Tekton TaskRun
				taskRunSample = ctl.DefaultTaskRunWithStatus(taskRunName, buildRunName, ns, corev1.ConditionTrue, "Succeeded")

				// initialize a BuildRun, we need this to fake the existence of a BuildRun
				buildRunSample = ctl.DefaultBuildRun(buildRunName, buildName)
			})

			It("is able to retrieve a TaskRun, Build and a BuildRun", func() {

				// stub the existence of a Build, BuildRun and
				// a TaskRun via the getClientStub, therefore we
				// expect the Reconcile to Succeed because all resources
				// exist
				result, err := reconciler.Reconcile(taskRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
				Expect(client.GetCallCount()).To(Equal(2))
			})
			It("does not fail when the BuildRun does not exist", func() {

				// override the initial getClientStub, and generate a new stub
				// that only contains a Build and TaskRun, none BuildRun
				stubGetCalls := ctl.StubBuildAndTaskRun(buildSample, taskRunSample)
				client.GetCalls(stubGetCalls)

				result, err := reconciler.Reconcile(taskRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
				Expect(client.GetCallCount()).To(Equal(2))
			})
			It("does not fail when the Build does not exist", func() {

				// override the initial getClientStub, and generate a new stub
				// that only contains a BuildRun and TaskRun, none Build
				stubGetCalls := ctl.StubBuildRunAndTaskRun(buildRunSample, taskRunSample)
				client.GetCalls(stubGetCalls)

				result, err := reconciler.Reconcile(taskRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
				Expect(client.GetCallCount()).To(Equal(2))
			})
			It("updates the BuildRun status", func() {

				// generated stub that asserts the BuildRun status fields when
				// status updates for a BuildRun take place
				statusCall := ctl.StubBuildRunStatus(
					"Succeeded",
					&taskRunName,
					build.Condition{
						Type:   build.Succeeded,
						Reason: "Succeeded",
						Status: corev1.ConditionTrue,
					},
					corev1.ConditionTrue,
					buildSample.Spec,
					false,
				)
				statusWriter.UpdateCalls(statusCall)

				// Assert for none errors while we exit the Reconcile
				// after updating the BuildRun status with the existing
				// TaskRun one
				result, err := reconciler.Reconcile(taskRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
				Expect(client.GetCallCount()).To(Equal(2))
				Expect(client.StatusCallCount()).To(Equal(1))
			})

			It("does not update the BuildRun status if the BuildRun is already completed", func() {
				buildRunSample = ctl.BuildRunWithSAGenerate(buildRunName, buildName)
				buildRunSample.Status.CompletionTime = &metav1.Time{
					Time: time.Now(),
				}

				client.GetCalls(ctl.StubBuildRunAndTaskRun(buildRunSample, taskRunSample))

				// Call the reconciler
				_, err := reconciler.Reconcile(taskRunRequest)

				// Expect no error
				Expect(err).ToNot(HaveOccurred())

				// Expect no delete call and no status update
				Expect(client.GetCallCount()).To(Equal(2))
				Expect(client.DeleteCallCount()).To(Equal(0))
				Expect(client.StatusCallCount()).To(Equal(0))
			})

			It("deletes a generated service account when the task run ends", func() {

				// setup a buildrun to use a generated service account
				buildSample = ctl.DefaultBuild(buildName, "foobar-strategy", build.ClusterBuildStrategyKind)
				buildRunSample = ctl.BuildRunWithSAGenerate(buildRunSample.Name, buildName)
				buildRunSample.Status.BuildSpec = &buildSample.Spec
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
				_, err := reconciler.Reconcile(taskRunRequest)

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
		Context("from an existing TaskRun with Conditions", func() {
			BeforeEach(func() {

				// Generate a new Reconcile Request using the existing TaskRun name and namespace
				taskRunRequest = newReconcileRequest(taskRunName, ns)

				// initialize a BuildRun, we need this to fake the existence of a BuildRun
				buildRunSample = ctl.DefaultBuildRun(buildRunName, buildName)
			})

			// Docs on the TaskRun conditions can be found here
			// https://github.com/tektoncd/pipeline/blob/master/docs/taskruns.md#monitoring-execution-status
			It("updates the BuildRun status with a PENDING reason", func() {

				// initialize a TaskRun, we need this to fake the existence of a Tekton TaskRun
				taskRunSample = ctl.DefaultTaskRunWithStatus(taskRunName, buildRunName, ns, corev1.ConditionUnknown, "Pending")

				// Stub that asserts the BuildRun status fields when
				// Status updates for a BuildRun take place
				statusCall := ctl.StubBuildRunStatus(
					"Pending",
					&taskRunName,
					build.Condition{
						Type:   build.Succeeded,
						Reason: "Pending",
						Status: corev1.ConditionUnknown,
					},
					corev1.ConditionUnknown,
					buildSample.Spec,
					false,
				)
				statusWriter.UpdateCalls(statusCall)

				// Assert for none errors while we exit the Reconcile
				// after updating the BuildRun status with the existing
				// TaskRun one
				result, err := reconciler.Reconcile(taskRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
				Expect(client.GetCallCount()).To(Equal(2))
				Expect(client.StatusCallCount()).To(Equal(1))
			})

			It("updates the BuildRun status with a RUNNING reason", func() {

				taskRunSample = ctl.DefaultTaskRunWithStatus(taskRunName, buildRunName, ns, corev1.ConditionUnknown, "Running")

				// Stub that asserts the BuildRun status fields when
				// Status updates for a BuildRun take place
				statusCall := ctl.StubBuildRunStatus(
					"Running",
					&taskRunName,
					build.Condition{
						Type:   build.Succeeded,
						Reason: "Running",
						Status: corev1.ConditionUnknown,
					},
					corev1.ConditionUnknown,
					buildSample.Spec,
					false,
				)
				statusWriter.UpdateCalls(statusCall)

				result, err := reconciler.Reconcile(taskRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
				Expect(client.GetCallCount()).To(Equal(2))
				Expect(client.StatusCallCount()).To(Equal(1))
			})

			It("updates the BuildRun status with a SUCCEEDED reason", func() {

				taskRunSample = ctl.DefaultTaskRunWithStatus(taskRunName, buildRunName, ns, corev1.ConditionTrue, "Succeeded")

				// Stub that asserts the BuildRun status fields when
				// Status updates for a BuildRun take place
				statusCall := ctl.StubBuildRunStatus(
					"Succeeded",
					&taskRunName,
					build.Condition{
						Type:   build.Succeeded,
						Reason: "Succeeded",
						Status: corev1.ConditionTrue,
					},
					corev1.ConditionTrue,
					buildSample.Spec,
					false,
				)
				statusWriter.UpdateCalls(statusCall)

				result, err := reconciler.Reconcile(taskRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
				Expect(client.GetCallCount()).To(Equal(2))
				Expect(client.StatusCallCount()).To(Equal(1))
			})

			It("updates the BuildRun status when a FALSE status occurs", func() {

				taskRunSample = ctl.DefaultTaskRunWithFalseStatus(taskRunName, buildRunName, ns)

				// Based on the current buildRun controller, if the TaskRun condition.Status
				// is FALSE, we will then populate our buildRun.Status.Reason with the
				// TaskRun condition.Message, rather than the condition.Reason
				statusCall := ctl.StubBuildRunStatus(
					"some message",
					&taskRunName,
					build.Condition{
						Type:   build.Succeeded,
						Reason: "something bad happened",
						Status: corev1.ConditionFalse,
					},
					corev1.ConditionFalse,
					buildSample.Spec,
					false,
				)
				statusWriter.UpdateCalls(statusCall)

				result, err := reconciler.Reconcile(taskRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
			})

			It("does not break the reconcile when a taskrun pod initcontainers are not ready", func() {
				taskRunSample = ctl.TaskRunWithCompletionAndStartTime(taskRunName, buildRunName, ns)

				buildRunSample = ctl.BuildRunWithBuildSnapshot(buildRunName, buildName)

				// Override Stub get calls to include a completed TaskRun
				// and a Pod with one initContainer Status
				client.GetCalls(ctl.StubBuildCRDsPodAndTaskRun(
					buildSample,
					buildRunSample,
					ctl.DefaultServiceAccount("foobar"),
					ctl.DefaultClusterBuildStrategy(),
					ctl.DefaultNamespacedBuildStrategy(),
					taskRunSample,
					ctl.PodWithInitContainerStatus("foobar", "init-foobar")),
				)

				result, err := reconciler.Reconcile(taskRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))

				// Three client calls because based on the Stub, we should
				// trigger a call to get the related TaskRun pod.
				Expect(client.GetCallCount()).To(Equal(3))
			})

			It("does not break the reconcile when a failed taskrun has a pod with no failed container", func() {
				buildRunSample = ctl.BuildRunWithBuildSnapshot(buildRunName, buildName)
				taskRunSample = &v1beta1.TaskRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      taskRunName,
						Namespace: ns,
						Labels:    map[string]string{"buildrun.shipwright.io/name": buildRunName},
					},
					Spec: v1beta1.TaskRunSpec{},
					Status: v1beta1.TaskRunStatus{
						TaskRunStatusFields: v1beta1.TaskRunStatusFields{
							PodName: "foobar",
							CompletionTime: &metav1.Time{
								Time: time.Now(),
							},
							StartTime: &metav1.Time{
								Time: time.Now(),
							},
						},
						Status: knativev1beta1.Status{
							Conditions: knativev1beta1.Conditions{
								{
									Type:    knativeapi.ConditionSucceeded,
									Reason:  string(v1beta1.TaskRunReasonFailed),
									Status:  corev1.ConditionFalse,
									Message: "some message",
								},
							},
						},
					},
				}

				client.GetCalls(ctl.StubBuildCRDsPodAndTaskRun(
					buildSample,
					buildRunSample,
					ctl.DefaultServiceAccount("foobar"),
					ctl.DefaultClusterBuildStrategy(),
					ctl.DefaultNamespacedBuildStrategy(),
					taskRunSample,
					&corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "foobar",
						},
						Status: corev1.PodStatus{},
					},
				))

				// Verify issue #591 by checking that Reconcile does not
				// fail with a panic due to a nil pointer dereference:
				// The pod has no container details and therefore the
				// look-up logic will find no container (result is nil).
				result, err := reconciler.Reconcile(taskRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
			})
		})

		Context("from an existing BuildRun resource", func() {
			var (
				saName           string
				emptyTaskRunName *string
			)
			BeforeEach(func() {
				saName = "foobar-sa"

				// Generate a new Reconcile Request using the existing BuildRun name and namespace
				buildRunRequest = newReconcileRequest(buildRunName, ns)

				// override the BuildRun resource to use a BuildRun with a specified
				// serviceaccount
				buildRunSample = ctl.BuildRunWithSA(buildRunName, buildName, saName)
			})

			It("fails on a TaskRun creation due to missing service account", func() {

				// override the initial getClientStub, and generate a new stub
				// that only contains a Build and Buildrun, none TaskRun
				stubGetCalls := func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
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

				client.GetCalls(stubGetCalls)

				// Stub that asserts the BuildRun status fields when
				// Status updates for a BuildRun take place
				statusCall := ctl.StubBuildRunStatus(
					fmt.Sprintf(" \"%s\" not found", saName),
					emptyTaskRunName,
					build.Condition{
						Type:   build.Succeeded,
						Reason: "Failed",
						Status: corev1.ConditionFalse,
					},
					corev1.ConditionFalse,
					buildSample.Spec,
					true,
				)
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(buildRunRequest)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Failed to choose a service account to use"))
				Expect(client.GetCallCount()).To(Equal(4))
				Expect(client.StatusCallCount()).To(Equal(2))
			})

			It("use the default serviceAccount when pipeline serviceAccount doesn't exist and generate serviceAccount is false", func() {
				// override the BuildRun without serviceAccount and generate is false
				buildRunSample = ctl.BuildRunWithoutSA(buildRunName, buildName)

				// override the initial getClientStub, and generate a new stub
				// that only contains a Build and Buildrun, none TaskRun
				stubGetCalls := func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
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
				client.GetCalls(stubGetCalls)

				_, err := reconciler.Reconcile(buildRunRequest)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(`"default" not found, msg: Failed to choose a service account to use`))
			})

			It("fails on a TaskRun creation due to missing namespaced buildstrategy", func() {

				// override the Build to use a namespaced BuildStrategy
				buildSample = ctl.DefaultBuild(buildName, strategyName, build.NamespacedBuildStrategyKind)

				// Override Stub get calls to include a service account
				// but none BuildStrategy
				client.GetCalls(ctl.StubBuildRunGetWithSA(
					buildSample,
					buildRunSample,
					ctl.DefaultServiceAccount(saName)),
				)

				_, err := reconciler.Reconcile(buildRunRequest)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(" \"%s\" not found", strategyName)))
			})

			It("fails on a TaskRun creation due to missing cluster buildstrategy", func() {
				// override the Build to use a cluster BuildStrategy
				buildSample = ctl.DefaultBuild(buildName, strategyName, build.ClusterBuildStrategyKind)

				// Override Stub get calls to include a service account
				// but none BuildStrategy
				client.GetCalls(ctl.StubBuildRunGetWithSA(
					buildSample,
					buildRunSample,
					ctl.DefaultServiceAccount(saName)),
				)

				_, err := reconciler.Reconcile(buildRunRequest)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(" \"%s\" not found", strategyName)))
			})

			It("fails on a TaskRun creation due to unknown buildStrategy kind", func() {
				buildSample = ctl.DefaultBuild(buildName, strategyName, fakeBuildStrategyKind)

				// Override Stub get calls to include a service account
				// but none BuildStrategy
				client.GetCalls(ctl.StubBuildRunGetWithSA(
					buildSample,
					buildRunSample,
					ctl.DefaultServiceAccount(saName)),
				)

				_, err := reconciler.Reconcile(buildRunRequest)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(" Unsupported BuildStrategy Kind"))
			})

			It("only generates the service account once if the taskRun cannot be created", func() {
				// override the Build to use a cluster BuildStrategy
				buildSample = ctl.DefaultBuild(buildName, strategyName, build.ClusterBuildStrategyKind)

				// override buildrun to use a generated service account
				buildRunName = "foobar-buildrun-sa-generate"
				buildRunSample = ctl.BuildRunWithSAGenerate(buildRunName, buildName)
				buildRunSample.Labels = make(map[string]string)
				buildRunSample.Labels[build.LabelBuild] = buildName
				buildRunSample.Labels[build.LabelBuildGeneration] = strconv.FormatInt(buildSample.Generation, 10)

				// Override stub calls to include only build and buildrun
				client.GetCalls(ctl.StubBuildRunGetWithoutSA(buildSample, buildRunSample))

				// Call the reconciler
				_, err := reconciler.Reconcile(buildRunRequest)

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
				_, err = reconciler.Reconcile(buildRunRequest)

				// Expect an error stating that the strategy does not exist
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(" \"%s\" not found", strategyName)))

				// Expect no more create call because service account already existed
				Expect(client.CreateCallCount()).To(Equal(1))

				// Expect zero update calls because service account already existed and did not need to be modified
				Expect(client.UpdateCallCount()).To(Equal(0))
			})

			It("fails on a TaskRun creation due to owner references errors", func() {
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
				_, err := reconciler.Reconcile(buildRunRequest)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("errors: foobar error"))
			})

			It("succeeds creating a TaskRun from a namespaced buildstrategy", func() {
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
						ctl.DefaultTaskRunWithStatus(taskRunName, buildRunName, ns, corev1.ConditionTrue, "Succeeded").DeepCopyInto(object)
					}
					return nil
				})

				_, err := reconciler.Reconcile(buildRunRequest)
				Expect(err).ToNot(HaveOccurred())

				Expect(client.CreateCallCount()).To(Equal(1))
			})

			It("succeeds creating a TaskRun from a cluster buildstrategy", func() {
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
						ctl.DefaultTaskRunWithStatus(taskRunName, buildRunName, ns, corev1.ConditionTrue, "Succeeded").DeepCopyInto(object)
					}
					return nil
				})

				_, err := reconciler.Reconcile(buildRunRequest)
				Expect(err).ToNot(HaveOccurred())
			})
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
				_, err := reconciler.Reconcile(buildRunRequest)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("reason: something bad happened"))
				Expect(client.StatusCallCount()).To(Equal(1))
			})

			It("delays creation if the registered status of the build is not yet set", func() {
				buildSample = ctl.DefaultBuild(buildName, strategyName, build.ClusterBuildStrategyKind)
				buildSample.Status.Registered = ""
				buildSample.Status.Reason = ""

				client.GetCalls(ctl.StubBuildRunGetWithoutSA(buildSample, buildRunSample))

				_, err := reconciler.Reconcile(buildRunRequest)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(fmt.Sprintf("the Build is not yet validated, build: %s", buildName)))
				Expect(client.StatusCallCount()).To(Equal(0))
			})

			It("succeeds creating a TaskRun even if the BuildSpec is already referenced", func() {
				// Set the build spec
				buildRunSample = ctl.DefaultBuildRun(buildRunName, buildName)
				buildRunSample.Status.BuildSpec = &buildSample.Spec

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
						ctl.DefaultTaskRunWithStatus(taskRunName, buildRunName, ns, corev1.ConditionTrue, "Succeeded").DeepCopyInto(object)
					}
					return nil
				})

				_, err := reconciler.Reconcile(buildRunRequest)
				Expect(err).ToNot(HaveOccurred())

				Expect(client.CreateCallCount()).To(Equal(1))
			})

			It("updates Build with error when BuildRun is already owned", func() {

				fakeOwnerName := "fakeOwner"

				// Set the build spec
				buildRunSample = ctl.BuildRunWithExistingOwnerReferences(buildRunName, buildName, fakeOwnerName)
				buildRunSample.Status.BuildSpec = &buildSample.Spec

				// override the Build to use a namespaced BuildStrategy
				buildSample = ctl.BuildWithBuildRunDeletions(buildName, strategyName, build.NamespacedBuildStrategyKind)

				// Override Stub get calls to include a service account
				// and BuildStrategies
				client.GetCalls(ctl.StubBuildRunGetWithSAandStrategies(
					buildSample,
					buildRunSample,
					ctl.DefaultServiceAccount(saName),
					ctl.DefaultClusterBuildStrategy(),
					ctl.DefaultNamespacedBuildStrategy()),
				)

				statusCall := ctl.StubBuildStatusReason(build.SetOwnerReferenceFailed,
					fmt.Sprintf("unexpected error when trying to set the ownerreference: Object /%s is already owned by another %s controller ", buildRunName, fakeOwnerName),
				)
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(buildRunRequest)
				Expect(err).ToNot(HaveOccurred())

				Expect(client.CreateCallCount()).To(Equal(1))
			})

			It("updates Build with error when BuildRun and Build are not in the same ns when setting ownerreferences", func() {
				// Set the build spec
				buildRunSample = ctl.BuildRunWithFakeNamespace(buildRunName, buildName)
				buildRunSample.Status.BuildSpec = &buildSample.Spec

				// override the Build to use a namespaced BuildStrategy
				buildSample = ctl.BuildWithBuildRunDeletionsAndFakeNS(buildName, strategyName, build.NamespacedBuildStrategyKind)

				// Override Stub get calls to include a service account
				// and BuildStrategies
				client.GetCalls(ctl.StubBuildRunGetWithSAandStrategies(
					buildSample,
					buildRunSample,
					ctl.DefaultServiceAccount(saName),
					ctl.DefaultClusterBuildStrategy(),
					ctl.DefaultNamespacedBuildStrategy()),
				)

				statusCall := ctl.StubBuildStatusReason(build.SetOwnerReferenceFailed,
					fmt.Sprintf("unexpected error when trying to set the ownerreference: cross-namespace owner references are disallowed, owner's namespace %s, obj's namespace %s", buildSample.Namespace, buildRunSample.Namespace),
				)
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(buildRunRequest)
				Expect(err).ToNot(HaveOccurred())

				Expect(client.CreateCallCount()).To(Equal(1))
			})

			It("ensure the Build can own a BuildRun when using the proper annotation", func() {

				buildRunSample = ctl.BuildRunWithoutSA(buildRunName, buildName)
				buildSample = ctl.BuildWithBuildRunDeletions(buildName, strategyName, build.NamespacedBuildStrategyKind)

				// Override Stub get calls to include a service account
				// and BuildStrategies
				client.GetCalls(ctl.StubBuildRunGetWithSAandStrategies(
					buildSample,
					buildRunSample,
					ctl.DefaultServiceAccount(saName),
					ctl.DefaultClusterBuildStrategy(),
					ctl.DefaultNamespacedBuildStrategy()),
				)

				// Ensure the BuildRun gets an ownershipReference when
				// the buildv1alpha1.AnnotationBuildRunDeletion is set to true
				// in the build
				clientUpdateCalls := ctl.StubBuildUpdateOwnerReferences("Build",
					buildName,
					pointer.BoolPtr(true),
					pointer.BoolPtr(true),
				)
				client.UpdateCalls(clientUpdateCalls)

				_, err := reconciler.Reconcile(buildRunRequest)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
