// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package build_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	crc "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/controller/fakes"
	buildController "github.com/shipwright-io/build/pkg/reconciler/build"
	"github.com/shipwright-io/build/test"
)

var _ = Describe("Reconcile Build", func() {
	var (
		manager                      *fakes.FakeManager
		reconciler                   reconcile.Reconciler
		request                      reconcile.Request
		buildSample                  *build.Build
		secretSample                 *corev1.Secret
		clusterBuildStrategySample   *build.ClusterBuildStrategy
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
		client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object) error {
			switch object := object.(type) {
			case *build.Build:
				buildSample.DeepCopyInto(object)
			case *build.ClusterBuildStrategy:
				clusterBuildStrategySample.DeepCopyInto(object)
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
		clusterBuildStrategySample = ctl.ClusterBuildStrategy(buildStrategyName)
		// Reconcile
		reconciler = buildController.NewReconciler(config.NewDefaultConfig(), manager, controllerutil.SetControllerReference)
	})

	Describe("Reconcile", func() {
		Context("when source secret is specified", func() {
			It("fails when the secret does not exist", func() {
				buildSample.Spec.Source.Credentials = &corev1.LocalObjectReference{
					Name: "non-existing",
				}
				buildSample.Spec.Output.Credentials = nil

				statusCall := ctl.StubFunc(corev1.ConditionFalse, build.SpecSourceSecretRefNotFound, "referenced secret non-existing not found")
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).To(BeNil())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
			})

			It("succeeds when the secret exists foobar", func() {
				buildSample.Spec.Source.Credentials = &corev1.LocalObjectReference{
					Name: "existing",
				}
				buildSample.Spec.Output.Credentials = nil

				// Fake some client Get calls and ensure we populate all
				// different resources we could get during reconciliation
				client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object) error {
					switch object := object.(type) {
					case *build.Build:
						buildSample.DeepCopyInto(object)
					case *build.ClusterBuildStrategy:
						clusterBuildStrategySample.DeepCopyInto(object)
					case *corev1.Secret:
						secretSample = ctl.SecretWithoutAnnotation("existing", namespace)
						secretSample.DeepCopyInto(object)
					}
					return nil
				})

				statusCall := ctl.StubFunc(corev1.ConditionTrue, build.SucceedStatus, "all validations succeeded")
				statusWriter.UpdateCalls(statusCall)

				result, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
				Expect(reconcile.Result{}).To(Equal(result))
			})
		})

		Context("when builder image secret is specified", func() {
			It("fails when the secret does not exist", func() {
				buildSample.Spec.Builder = &build.Image{
					Image: "busybox",
					Credentials: &corev1.LocalObjectReference{
						Name: "non-existing",
					},
				}
				buildSample.Spec.Output.Credentials = nil

				statusCall := ctl.StubFunc(corev1.ConditionFalse, build.SpecBuilderSecretRefNotFound, "referenced secret non-existing not found")
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).To(BeNil())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
			})

			It("succeeds when the secret exists", func() {
				buildSample.Spec.Builder = &build.Image{
					Image: "busybox",
					Credentials: &corev1.LocalObjectReference{
						Name: "existing",
					},
				}
				buildSample.Spec.Output.Credentials = nil

				// Fake some client Get calls and ensure we populate all
				// different resources we could get during reconciliation
				client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object) error {
					switch object := object.(type) {
					case *build.Build:
						buildSample.DeepCopyInto(object)
					case *build.ClusterBuildStrategy:
						clusterBuildStrategySample.DeepCopyInto(object)
					case *corev1.Secret:
						secretSample = ctl.SecretWithoutAnnotation("existing", namespace)
						secretSample.DeepCopyInto(object)
					}
					return nil
				})

				statusCall := ctl.StubFunc(corev1.ConditionTrue, build.SucceedStatus, "all validations succeeded")
				statusWriter.UpdateCalls(statusCall)

				result, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
				Expect(reconcile.Result{}).To(Equal(result))
			})
		})

		Context("when spec output registry secret is specified", func() {
			It("fails when the secret does not exist", func() {

				statusCall := ctl.StubFunc(corev1.ConditionFalse, build.SpecOutputSecretRefNotFound, fmt.Sprintf("referenced secret %s not found", registrySecret))
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).To(BeNil())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
			})
			It("succeed when the secret exists", func() {
				// Fake some client Get calls and ensure we populate all
				// different resources we could get during reconciliation
				client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object) error {
					switch object := object.(type) {
					case *build.Build:
						buildSample.DeepCopyInto(object)
					case *build.ClusterBuildStrategy:
						clusterBuildStrategySample.DeepCopyInto(object)
					case *corev1.Secret:
						secretSample = ctl.SecretWithoutAnnotation("existing", namespace)
						secretSample.DeepCopyInto(object)
					}
					return nil
				})

				statusCall := ctl.StubFunc(corev1.ConditionTrue, build.SucceedStatus, "all validations succeeded")
				statusWriter.UpdateCalls(statusCall)

				result, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
				Expect(reconcile.Result{}).To(Equal(result))
			})
		})

		Context("when source secret and output secret are specified", func() {
			It("fails when both secrets do not exist", func() {
				buildSample.Spec.Source.Credentials = &corev1.LocalObjectReference{
					Name: "non-existing-source",
				}
				buildSample.Spec.Output.Credentials = &corev1.LocalObjectReference{
					Name: "non-existing-output",
				}

				statusCall := ctl.StubFunc(corev1.ConditionFalse, build.MultipleSecretRefNotFound, "missing secrets are non-existing-output,non-existing-source")
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).To(BeNil())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
			})
		})

		Context("when spec strategy ClusterBuildStrategy is specified", func() {
			It("fails when the strategy does not exists", func() {

				// Fake some client Get calls and ensure we populate all
				// different resources we could get during reconciliation
				client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object) error {
					switch object := object.(type) {
					case *build.Build:
						buildSample.DeepCopyInto(object)
					case *build.ClusterBuildStrategy:
						return ctl.FakeClusterBuildStrategyNotFound("ss")
					case *corev1.Secret:
						secretSample = ctl.SecretWithoutAnnotation("existing", namespace)
						secretSample.DeepCopyInto(object)
					}
					return nil
				})

				statusCall := ctl.StubFunc(corev1.ConditionFalse, build.ClusterBuildStrategyNotFound, fmt.Sprintf("clusterBuildStrategy %s does not exist", buildStrategyName))
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).To(BeNil())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
			})
			It("succeed when the strategy exists", func() {

				// Fake some client Get calls and ensure we populate all
				// different resources we could get during reconciliation
				client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object) error {
					switch object := object.(type) {
					case *build.Build:
						buildSample.DeepCopyInto(object)
					case *build.ClusterBuildStrategy:
						clusterBuildStrategySample.DeepCopyInto(object)
					case *corev1.Secret:
						secretSample = ctl.SecretWithoutAnnotation("existing", namespace)
						secretSample.DeepCopyInto(object)
					}
					return nil
				})

				statusCall := ctl.StubFunc(corev1.ConditionTrue, build.SucceedStatus, "all validations succeeded")
				statusWriter.UpdateCalls(statusCall)

				result, err := reconciler.Reconcile(context.TODO(), request)
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

				statusCall := ctl.StubFunc(corev1.ConditionFalse, build.BuildStrategyNotFound, fmt.Sprintf("buildStrategy %s does not exist in namespace %s", buildStrategyName, namespace))
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).To(BeNil())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
			})
			It("succeed when the strategy exists", func() {

				// Fake some client Get calls and ensure we populate all
				// different resources we could get during reconciliation
				client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object) error {
					switch object := object.(type) {
					case *build.Build:
						buildSample.DeepCopyInto(object)
					case *build.BuildStrategy:
						namespacedBuildStrategy := ctl.DefaultNamespacedBuildStrategy()
						namespacedBuildStrategy.DeepCopyInto(object)
					}
					return nil
				})

				statusCall := ctl.StubFunc(corev1.ConditionTrue, build.SucceedStatus, "all validations succeeded")
				statusWriter.UpdateCalls(statusCall)

				result, err := reconciler.Reconcile(context.TODO(), request)
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

				statusCall := ctl.StubFunc(corev1.ConditionFalse, build.BuildStrategyNotFound, fmt.Sprintf("buildStrategy %s does not exist in namespace %s", buildStrategyName, namespace))
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).To(BeNil())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))

			})
			It("default to BuildStrategy and succeed if the strategy exists", func() {
				// Fake some client Get calls and ensure we populate all
				// different resources we could get during reconciliation
				client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object) error {
					switch object := object.(type) {
					case *build.Build:
						buildSample.DeepCopyInto(object)
					case *build.BuildStrategy:
						namespacedBuildStrategy := ctl.DefaultNamespacedBuildStrategy()
						namespacedBuildStrategy.DeepCopyInto(object)
					}
					return nil
				})

				statusCall := ctl.StubFunc(corev1.ConditionTrue, build.SucceedStatus, "all validations succeeded")
				statusWriter.UpdateCalls(statusCall)

				result, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
				Expect(reconcile.Result{}).To(Equal(result))
			})
		})

		Context("when source URL is specified", func() {
			// validate file protocol
			It("fails when source URL is invalid", func() {
				buildSample.Spec.Source.URL = pointer.String("foobar")
				buildSample.SetAnnotations(map[string]string{
					build.AnnotationBuildVerifyRepository: "true",
				})
				statusCall := ctl.StubFunc(corev1.ConditionFalse, build.RemoteRepositoryUnreachable, "invalid source url")
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).To(BeNil())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
			})

			// validate https protocol
			It("fails when public source URL is unreachable", func() {
				buildSample.Spec.Source.URL = pointer.String("https://github.com/shipwright-io/sample-go-fake")
				buildSample.SetAnnotations(map[string]string{
					build.AnnotationBuildVerifyRepository: "true",
				})

				statusCall := ctl.StubFunc(corev1.ConditionFalse, build.RemoteRepositoryUnreachable, "remote repository unreachable")
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).To(BeNil())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
			})

			// skip validation because of empty sourceURL annotation
			It("succeed when source URL is invalid because source annotation is empty", func() {
				buildSample.Spec.Source.URL = pointer.String("foobar")

				// Fake some client Get calls and ensure we populate all
				// different resources we could get during reconciliation
				client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object) error {
					switch object := object.(type) {
					case *build.Build:
						buildSample.DeepCopyInto(object)
					case *build.ClusterBuildStrategy:
						clusterBuildStrategySample.DeepCopyInto(object)
					}
					return nil
				})

				statusCall := ctl.StubFunc(corev1.ConditionTrue, build.SucceedStatus, build.AllValidationsSucceeded)
				statusWriter.UpdateCalls(statusCall)

				result, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
				Expect(reconcile.Result{}).To(Equal(result))
			})

			// skip validation because of false sourceURL annotation
			It("succeed when source URL is invalid because source annotation is false", func() {
				buildSample = ctl.BuildWithClusterBuildStrategyAndFalseSourceAnnotation(buildName, namespace, buildStrategyName)

				// Fake some client Get calls and ensure we populate all
				// different resources we could get during reconciliation
				client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object) error {
					switch object := object.(type) {
					case *build.Build:
						buildSample.DeepCopyInto(object)
					case *build.ClusterBuildStrategy:
						clusterBuildStrategySample.DeepCopyInto(object)
					}
					return nil
				})

				statusCall := ctl.StubFunc(corev1.ConditionTrue, build.SucceedStatus, build.AllValidationsSucceeded)
				statusWriter.UpdateCalls(statusCall)

				result, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
				Expect(reconcile.Result{}).To(Equal(result))
			})

			// skip validation because build references a sourceURL secret
			It("succeed when source URL is fake private URL because build reference a sourceURL secret", func() {
				buildSample := ctl.BuildWithClusterBuildStrategyAndSourceSecret(buildName, namespace, buildStrategyName)
				buildSample.Spec.Source.URL = pointer.String("https://github.yourco.com/org/build-fake")
				buildSample.Spec.Source.Credentials.Name = registrySecret

				// Fake some client Get calls and ensure we populate all
				// different resources we could get during reconciliation
				client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object) error {
					switch object := object.(type) {
					case *build.Build:
						buildSample.DeepCopyInto(object)
					case *build.ClusterBuildStrategy:
						clusterBuildStrategySample.DeepCopyInto(object)
					}

					return nil
				})

				statusCall := ctl.StubFunc(corev1.ConditionTrue, build.SucceedStatus, build.AllValidationsSucceeded)
				statusWriter.UpdateCalls(statusCall)

				result, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
				Expect(reconcile.Result{}).To(Equal(result))
			})
		})

		Context("when environment variables are specified", func() {
			JustBeforeEach(func() {
				buildSample.Spec.Source.Credentials = &corev1.LocalObjectReference{
					Name: "existing",
				}
				buildSample.Spec.Output.Credentials = nil

				// Fake some client Get calls and ensure we populate all
				// different resources we could get during reconciliation
				client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object) error {
					switch object := object.(type) {
					case *build.Build:
						buildSample.DeepCopyInto(object)
					case *build.ClusterBuildStrategy:
						clusterBuildStrategySample.DeepCopyInto(object)
					case *corev1.Secret:
						secretSample = ctl.SecretWithoutAnnotation("existing", namespace)
						secretSample.DeepCopyInto(object)
					}
					return nil
				})
			})
			It("fails when the name is blank", func() {
				buildSample.Spec.Env = []corev1.EnvVar{
					{
						Name:  "",
						Value: "some-value",
					},
				}

				statusCall := ctl.StubFunc(corev1.ConditionFalse, build.SpecEnvNameCanNotBeBlank, "name for environment variable must not be blank")
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).To(BeNil())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))

			})
			It("fails when the name is blank using valueFrom", func() {
				buildSample.Spec.Env = []corev1.EnvVar{
					{
						Name: "",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "my-field-path",
							},
						},
					},
				}

				statusCall := ctl.StubFunc(corev1.ConditionFalse, build.SpecEnvNameCanNotBeBlank, "name for environment variable must not be blank")
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).To(BeNil())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))

			})
			It("fails when both value and valueFrom are specified", func() {
				buildSample.Spec.Env = []corev1.EnvVar{
					{
						Name:  "some-name",
						Value: "some-value",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "my-field-path",
							},
						},
					},
				}

				statusCall := ctl.StubFunc(corev1.ConditionFalse, build.SpecEnvOnlyOneOfValueOrValueFromMustBeSpecified, "only one of value or valueFrom must be specified")
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).To(BeNil())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))

			})
			It("succeeds with compliant env var using Value", func() {
				buildSample.Spec.Env = []corev1.EnvVar{
					{
						Name:  "some-name",
						Value: "some-value",
					},
				}

				statusCall := ctl.StubFunc(corev1.ConditionTrue, build.BuildReason(build.Succeeded), "all validations succeeded")
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).To(BeNil())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))

			})
			It("succeeds with compliant env var using ValueFrom", func() {
				buildSample.Spec.Env = []corev1.EnvVar{
					{
						Name: "some-name",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "my-field-path",
							},
						},
					},
				}

				statusCall := ctl.StubFunc(corev1.ConditionTrue, build.BuildReason(build.Succeeded), "all validations succeeded")
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).To(BeNil())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))

			})
		})

		Context("when build object is not in the cluster (anymore)", func() {
			It("should finish reconciling when the build cannot be found", func() {
				client.GetCalls(func(_ context.Context, nn types.NamespacedName, o crc.Object) error {
					return errors.NewNotFound(build.Resource("build"), nn.Name)
				})

				_, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).To(BeNil())
			})

			It("should finish reconciling with an error when looking up the build fails with an unexpected error", func() {
				client.GetCalls(func(_ context.Context, nn types.NamespacedName, o crc.Object) error {
					return errors.NewBadRequest("foobar")
				})

				_, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).ToNot(BeNil())
			})
		})
	})
})
