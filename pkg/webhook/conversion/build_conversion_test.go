// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package conversion

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

	buildapialpha "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

var _ = Describe("Build Conversion", func() {
	ctx := context.TODO()

	Context("convertBuildBetaToAlpha", func() {
		It("converts a full Build from v1beta1 to v1alpha1 with Git source", func() {
			timeout := metav1.Duration{Duration: 15 * time.Minute}
			cleanTimeout := metav1.Duration{Duration: 10 * time.Minute}
			failedLimit := uint(5)
			succeededLimit := uint(10)

			betaBuild := &buildapi.Build{
				TypeMeta: metav1.TypeMeta{
					APIVersion: betaGroupVersion,
					Kind:       buildKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-build",
					Namespace: "default",
				},
				Spec: buildapi.BuildSpec{
					Source: &buildapi.Source{
						Type:       buildapi.GitType,
						ContextDir: ptr.To("subdir"),
						Git: &buildapi.Git{
							URL:         "https://github.com/shipwright-io/sample-go",
							Revision:    ptr.To("main"),
							CloneSecret: ptr.To("git-secret"),
						},
					},
					Strategy: buildapi.Strategy{
						Name: "source-to-image",
						Kind: ptr.To(buildapi.NamespacedBuildStrategyKind),
					},
					ParamValues: []buildapi.ParamValue{
						{
							Name: "single-param",
							SingleValue: &buildapi.SingleValue{
								Value: ptr.To("literal-val"),
							},
						},
						{
							Name: "configmap-param",
							SingleValue: &buildapi.SingleValue{
								ConfigMapValue: &buildapi.ObjectKeyRef{
									Name: "my-cm",
									Key:  "my-key",
								},
							},
						},
						{
							Name: "secret-param",
							SingleValue: &buildapi.SingleValue{
								SecretValue: &buildapi.ObjectKeyRef{
									Name: "my-sec",
									Key:  "sec-key",
								},
							},
						},
						{
							Name: "array-param",
							Values: []buildapi.SingleValue{
								{Value: ptr.To("val1")},
								{
									ConfigMapValue: &buildapi.ObjectKeyRef{
										Name: "cm2",
										Key:  "k2",
									},
								},
								{
									SecretValue: &buildapi.ObjectKeyRef{
										Name: "sec2",
										Key:  "k3",
									},
								},
							},
						},
					},
					Output: buildapi.Image{
						Image:       "image-registry.openshift-image-registry.svc:5000/example/build-output:latest",
						PushSecret:  ptr.To("push-secret"),
						Insecure:    ptr.To(true),
						Timestamp:   ptr.To("Zero"),
						Annotations: map[string]string{"foo": "bar"},
						Labels:      map[string]string{"env": "test"},
					},
					Retention: &buildapi.BuildRetention{
						AtBuildDeletion: ptr.To(true),
						TTLAfterFailed:  &cleanTimeout,
						FailedLimit:     &failedLimit,
						SucceededLimit:  &succeededLimit,
					},
					Volumes: []buildapi.BuildVolume{
						{
							Name: "cache-vol",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					Timeout: &timeout,
					Env: []corev1.EnvVar{
						{Name: "GOPROXY", Value: "off"},
					},
					Trigger: &buildapi.Trigger{
						When: []buildapi.TriggerWhen{
							{
								Name: "github-pr",
								Type: buildapi.GitHubWebHookTrigger,
								GitHub: &buildapi.WhenGitHub{
									Events:   []buildapi.GitHubEventName{buildapi.GitHubPullRequestEvent},
									Branches: []string{"main"},
								},
							},
							{
								Name: "image-trigger",
								Type: buildapi.ImageTrigger,
								Image: &buildapi.WhenImage{
									Names: []string{"registry/my-img"},
								},
							},
							{
								Name: "object-trigger",
								Type: buildapi.PipelineTrigger,
								ObjectRef: &buildapi.WhenObjectRef{
									Name: "pipeline-obj",
								},
							},
						},
					},
				},
				Status: buildapi.BuildStatus{
					Registered: ptr.To(corev1.ConditionTrue),
					Reason:     ptr.To(buildapi.SucceedStatus),
					Message:    ptr.To("build validated"),
				},
			}

			unstructuredObj := &unstructured.Unstructured{}
			err := convertBuildBetaToAlpha(ctx, betaBuild, unstructuredObj)
			Expect(err).ToNot(HaveOccurred())

			var alphaBuild buildapialpha.Build
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.UnstructuredContent(), &alphaBuild)
			Expect(err).ToNot(HaveOccurred())

			Expect(alphaBuild.APIVersion).To(Equal(alphaGroupVersion))
			Expect(alphaBuild.Name).To(Equal("test-build"))
			Expect(alphaBuild.Namespace).To(Equal("default"))

			// Source Git
			Expect(alphaBuild.Spec.Source.URL).To(Equal(ptr.To("https://github.com/shipwright-io/sample-go")))
			Expect(alphaBuild.Spec.Source.Revision).To(Equal(ptr.To("main")))
			Expect(alphaBuild.Spec.Source.ContextDir).To(Equal(ptr.To("subdir")))
			Expect(alphaBuild.Spec.Source.Credentials).To(Equal(&corev1.LocalObjectReference{Name: "git-secret"}))

			// Strategy
			Expect(alphaBuild.Spec.Strategy.Name).To(Equal("source-to-image"))
			Expect(*alphaBuild.Spec.Strategy.Kind).To(Equal(buildapialpha.NamespacedBuildStrategyKind))

			// ParamValues
			Expect(alphaBuild.Spec.ParamValues).To(HaveLen(4))
			Expect(alphaBuild.Spec.ParamValues[0].Name).To(Equal("single-param"))
			Expect(alphaBuild.Spec.ParamValues[0].Value).To(Equal(ptr.To("literal-val")))
			Expect(alphaBuild.Spec.ParamValues[1].Name).To(Equal("configmap-param"))
			Expect(alphaBuild.Spec.ParamValues[1].SingleValue.ConfigMapValue.Name).To(Equal("my-cm"))
			Expect(alphaBuild.Spec.ParamValues[2].Name).To(Equal("secret-param"))
			Expect(alphaBuild.Spec.ParamValues[2].SingleValue.SecretValue.Name).To(Equal("my-sec"))
			Expect(alphaBuild.Spec.ParamValues[3].Name).To(Equal("array-param"))
			Expect(alphaBuild.Spec.ParamValues[3].Values).To(HaveLen(3))
			Expect(alphaBuild.Spec.ParamValues[3].Values[0].Value).To(Equal(ptr.To("val1")))
			Expect(alphaBuild.Spec.ParamValues[3].Values[1].ConfigMapValue.Name).To(Equal("cm2"))
			Expect(alphaBuild.Spec.ParamValues[3].Values[2].SecretValue.Name).To(Equal("sec2"))

			// Output
			Expect(alphaBuild.Spec.Output.Image).To(Equal("image-registry.openshift-image-registry.svc:5000/example/build-output:latest"))
			Expect(alphaBuild.Spec.Output.Credentials).To(Equal(&corev1.LocalObjectReference{Name: "push-secret"}))
			Expect(alphaBuild.Spec.Output.Insecure).To(Equal(ptr.To(true)))
			Expect(alphaBuild.Spec.Output.Timestamp).To(Equal(ptr.To("Zero")))
			Expect(alphaBuild.Spec.Output.Annotations).To(Equal(map[string]string{"foo": "bar"}))
			Expect(alphaBuild.Spec.Output.Labels).To(Equal(map[string]string{"env": "test"}))

			// Retention and annotation
			Expect(alphaBuild.Annotations[buildapialpha.AnnotationBuildRunDeletion]).To(Equal("true"))
			Expect(alphaBuild.Spec.Retention.TTLAfterFailed).To(Equal(&cleanTimeout))
			Expect(alphaBuild.Spec.Retention.FailedLimit).To(Equal(&failedLimit))
			Expect(alphaBuild.Spec.Retention.SucceededLimit).To(Equal(&succeededLimit))

			// Volumes, Timeout, Env
			Expect(alphaBuild.Spec.Volumes).To(HaveLen(1))
			Expect(alphaBuild.Spec.Volumes[0].Name).To(Equal("cache-vol"))
			Expect(alphaBuild.Spec.Timeout).To(Equal(&timeout))
			Expect(alphaBuild.Spec.Env).To(Equal([]corev1.EnvVar{{Name: "GOPROXY", Value: "off"}}))

			// Triggers
			Expect(alphaBuild.Spec.Trigger.When).To(HaveLen(3))
			Expect(alphaBuild.Spec.Trigger.When[0].Name).To(Equal("github-pr"))
			Expect(alphaBuild.Spec.Trigger.When[0].Type).To(Equal(buildapialpha.GitHubWebHookTrigger))
			Expect(alphaBuild.Spec.Trigger.When[0].GitHub.Branches).To(Equal([]string{"main"}))
			Expect(alphaBuild.Spec.Trigger.When[1].Name).To(Equal("image-trigger"))
			Expect(alphaBuild.Spec.Trigger.When[1].Image.Names).To(Equal([]string{"registry/my-img"}))
			Expect(alphaBuild.Spec.Trigger.When[2].Name).To(Equal("object-trigger"))
			Expect(alphaBuild.Spec.Trigger.When[2].ObjectRef.Name).To(Equal("pipeline-obj"))

			// Status
			Expect(alphaBuild.Status.Registered).To(Equal(ptr.To(corev1.ConditionTrue)))
			Expect(*alphaBuild.Status.Reason).To(Equal(buildapialpha.SucceedStatus))
			Expect(alphaBuild.Status.Message).To(Equal(ptr.To("build validated")))
		})

		It("converts a Build from v1beta1 to v1alpha1 with OCIArtifact source and ClusterBuildStrategy", func() {
			prune := buildapi.PruneAfterPull
			betaBuild := &buildapi.Build{
				TypeMeta: metav1.TypeMeta{
					APIVersion: betaGroupVersion,
					Kind:       buildKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "oci-build",
				},
				Spec: buildapi.BuildSpec{
					Source: &buildapi.Source{
						Type: buildapi.OCIArtifactType,
						OCIArtifact: &buildapi.OCIArtifact{
							Image:      "quay.io/org/bundle:latest",
							PullSecret: ptr.To("pull-secret"),
							Prune:      &prune,
						},
					},
					Strategy: buildapi.Strategy{
						Name: "cluster-buildah",
						Kind: ptr.To(buildapi.ClusterBuildStrategyKind),
					},
				},
			}

			unstructuredObj := &unstructured.Unstructured{}
			err := convertBuildBetaToAlpha(ctx, betaBuild, unstructuredObj)
			Expect(err).ToNot(HaveOccurred())

			var alphaBuild buildapialpha.Build
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.UnstructuredContent(), &alphaBuild)
			Expect(err).ToNot(HaveOccurred())

			Expect(alphaBuild.Spec.Source.BundleContainer).ToNot(BeNil())
			Expect(alphaBuild.Spec.Source.BundleContainer.Image).To(Equal("quay.io/org/bundle:latest"))
			Expect(alphaBuild.Spec.Source.BundleContainer.Prune).To(Equal((*buildapialpha.PruneOption)(&prune)))
			Expect(alphaBuild.Spec.Source.Credentials).To(Equal(&corev1.LocalObjectReference{Name: "pull-secret"}))
			Expect(*alphaBuild.Spec.Strategy.Kind).To(Equal(buildapialpha.ClusterBuildStrategyKind))
		})

		It("converts a Build from v1beta1 to v1alpha1 with Local source", func() {
			timeout := metav1.Duration{Duration: 5 * time.Minute}
			betaBuild := &buildapi.Build{
				TypeMeta: metav1.TypeMeta{
					APIVersion: betaGroupVersion,
					Kind:       buildKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "local-build",
				},
				Spec: buildapi.BuildSpec{
					Source: &buildapi.Source{
						Type: buildapi.LocalType,
						Local: &buildapi.Local{
							Name:    "source-archive",
							Timeout: &timeout,
						},
					},
					Strategy: buildapi.Strategy{
						Name: "buildpacks",
					},
				},
			}

			unstructuredObj := &unstructured.Unstructured{}
			err := convertBuildBetaToAlpha(ctx, betaBuild, unstructuredObj)
			Expect(err).ToNot(HaveOccurred())

			var alphaBuild buildapialpha.Build
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.UnstructuredContent(), &alphaBuild)
			Expect(err).ToNot(HaveOccurred())

			Expect(alphaBuild.Spec.Sources).To(HaveLen(1))
			Expect(alphaBuild.Spec.Sources[0].Type).To(Equal(buildapialpha.LocalCopy))
			Expect(alphaBuild.Spec.Sources[0].Name).To(Equal("source-archive"))
			Expect(alphaBuild.Spec.Sources[0].Timeout).To(Equal(&timeout))
		})

		It("handles nil Source and empty fields gracefully", func() {
			betaBuild := &buildapi.Build{
				TypeMeta: metav1.TypeMeta{
					APIVersion: betaGroupVersion,
					Kind:       buildKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "empty-build",
				},
				Spec: buildapi.BuildSpec{
					Strategy: buildapi.Strategy{
						Name: "kaniko",
					},
				},
			}

			unstructuredObj := &unstructured.Unstructured{}
			err := convertBuildBetaToAlpha(ctx, betaBuild, unstructuredObj)
			Expect(err).ToNot(HaveOccurred())

			var alphaBuild buildapialpha.Build
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.UnstructuredContent(), &alphaBuild)
			Expect(err).ToNot(HaveOccurred())

			Expect(alphaBuild.Spec.Source.URL).To(BeNil())
			Expect(alphaBuild.Spec.Sources).To(BeNil())
			Expect(alphaBuild.Spec.Output.Image).To(BeEmpty())
			Expect(alphaBuild.Spec.Retention).To(BeNil())
			Expect(alphaBuild.Annotations[buildapialpha.AnnotationBuildRunDeletion]).To(BeEmpty())
		})
	})

	Context("convertBuildAlphaToBeta", func() {
		It("converts a full Build from v1alpha1 to v1beta1 with Git source and retention annotation", func() {
			timeout := metav1.Duration{Duration: 20 * time.Minute}
			cleanTimeout := metav1.Duration{Duration: 8 * time.Minute}
			failedLimit := uint(3)
			succeededLimit := uint(7)

			alphaBuild := &buildapialpha.Build{
				TypeMeta: metav1.TypeMeta{
					APIVersion: alphaGroupVersion,
					Kind:       buildKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "alpha-build",
					Namespace: "prod",
					Annotations: map[string]string{
						buildapialpha.AnnotationBuildRunDeletion: "true",
					},
				},
				Spec: buildapialpha.BuildSpec{
					Source: buildapialpha.Source{
						URL:         ptr.To("https://github.com/shipwright-io/build"),
						Revision:    ptr.To("v0.12.0"),
						ContextDir:  ptr.To("context"),
						Credentials: &corev1.LocalObjectReference{Name: "git-cred"},
					},
					Strategy: buildapialpha.Strategy{
						Name: "custom-strategy",
						Kind: ptr.To(buildapialpha.NamespacedBuildStrategyKind),
					},
					ParamValues: []buildapialpha.ParamValue{
						{
							Name: "single",
							SingleValue: &buildapialpha.SingleValue{
								Value: ptr.To("str-val"),
							},
						},
						{
							Name: "cm",
							SingleValue: &buildapialpha.SingleValue{
								ConfigMapValue: &buildapialpha.ObjectKeyRef{
									Name: "cm-name",
									Key:  "cm-key",
								},
							},
						},
						{
							Name: "sec",
							SingleValue: &buildapialpha.SingleValue{
								SecretValue: &buildapialpha.ObjectKeyRef{
									Name: "sec-name",
									Key:  "sec-key",
								},
							},
						},
						{
							Name: "arr",
							Values: []buildapialpha.SingleValue{
								{Value: ptr.To("v1")},
								{ConfigMapValue: &buildapialpha.ObjectKeyRef{Name: "cm-sub", Key: "k"}},
								{SecretValue: &buildapialpha.ObjectKeyRef{Name: "sec-sub", Key: "s"}},
							},
						},
					},
					Output: buildapialpha.Image{
						Image:       "quay.io/example/app:tag",
						Credentials: &corev1.LocalObjectReference{Name: "reg-secret"},
						Insecure:    ptr.To(false),
						Timestamp:   ptr.To("Zero"),
						Annotations: map[string]string{"release": "1.0"},
						Labels:      map[string]string{"app": "demo"},
					},
					Retention: &buildapialpha.BuildRetention{
						TTLAfterFailed: &cleanTimeout,
						FailedLimit:    &failedLimit,
						SucceededLimit: &succeededLimit,
					},
					Volumes: []buildapialpha.BuildVolume{
						{
							Name: "shared-data",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					Timeout: &timeout,
					Env: []corev1.EnvVar{
						{Name: "DEBUG", Value: "1"},
					},
					Trigger: &buildapialpha.Trigger{
						When: []buildapialpha.TriggerWhen{
							{
								Name: "gh-trigger",
								Type: buildapialpha.GitHubWebHookTrigger,
								GitHub: &buildapialpha.WhenGitHub{
									Events:   []buildapialpha.GitHubEventName{buildapialpha.GitHubPushEvent},
									Branches: []string{"release-*"},
								},
							},
							{
								Name: "img-trigger",
								Type: buildapialpha.ImageTrigger,
								Image: &buildapialpha.WhenImage{
									Names: []string{"base-image:latest"},
								},
							},
							{
								Name: "obj-trigger",
								Type: buildapialpha.PipelineTrigger,
								ObjectRef: &buildapialpha.WhenObjectRef{
									Name: "pipeline-obj",
								},
							},
						},
					},
				},
				Status: buildapialpha.BuildStatus{
					Registered: ptr.To(corev1.ConditionTrue),
					Reason:     ptr.To(buildapialpha.SucceedStatus),
					Message:    ptr.To("ready"),
				},
			}

			unstructuredObj := &unstructured.Unstructured{}
			mapito, err := runtime.DefaultUnstructuredConverter.ToUnstructured(alphaBuild)
			Expect(err).ToNot(HaveOccurred())
			unstructuredObj.SetUnstructuredContent(mapito)

			var betaBuild buildapi.Build
			err = convertBuildAlphaToBeta(ctx, &betaBuild, unstructuredObj)
			Expect(err).ToNot(HaveOccurred())

			Expect(betaBuild.APIVersion).To(Equal(betaGroupVersion))
			Expect(betaBuild.Name).To(Equal("alpha-build"))
			Expect(betaBuild.Namespace).To(Equal("prod"))

			// Source Git
			Expect(betaBuild.Spec.Source).ToNot(BeNil())
			Expect(betaBuild.Spec.Source.Type).To(Equal(buildapi.GitType))
			Expect(betaBuild.Spec.Source.Git.URL).To(Equal("https://github.com/shipwright-io/build"))
			Expect(betaBuild.Spec.Source.Git.Revision).To(Equal(ptr.To("v0.12.0")))
			Expect(betaBuild.Spec.Source.ContextDir).To(Equal(ptr.To("context")))
			Expect(betaBuild.Spec.Source.Git.CloneSecret).To(Equal(ptr.To("git-cred")))

			// Strategy
			Expect(betaBuild.Spec.Strategy.Name).To(Equal("custom-strategy"))
			Expect(*betaBuild.Spec.Strategy.Kind).To(Equal(buildapi.NamespacedBuildStrategyKind))

			// Retention
			Expect(betaBuild.Spec.Retention).ToNot(BeNil())
			Expect(betaBuild.Spec.Retention.AtBuildDeletion).To(Equal(ptr.To(true)))
			Expect(betaBuild.Spec.Retention.TTLAfterFailed).To(Equal(&cleanTimeout))
			Expect(betaBuild.Spec.Retention.FailedLimit).To(Equal(&failedLimit))
			Expect(betaBuild.Spec.Retention.SucceededLimit).To(Equal(&succeededLimit))

			// Params
			Expect(betaBuild.Spec.ParamValues).To(HaveLen(4))
			Expect(betaBuild.Spec.ParamValues[0].Name).To(Equal("single"))
			Expect(betaBuild.Spec.ParamValues[0].Value).To(Equal(ptr.To("str-val")))
			Expect(betaBuild.Spec.ParamValues[1].Name).To(Equal("cm"))
			Expect(betaBuild.Spec.ParamValues[1].ConfigMapValue.Name).To(Equal("cm-name"))
			Expect(betaBuild.Spec.ParamValues[2].Name).To(Equal("sec"))
			Expect(betaBuild.Spec.ParamValues[2].SecretValue.Name).To(Equal("sec-name"))
			Expect(betaBuild.Spec.ParamValues[3].Name).To(Equal("arr"))
			Expect(betaBuild.Spec.ParamValues[3].Values).To(HaveLen(3))
			Expect(betaBuild.Spec.ParamValues[3].Values[0].Value).To(Equal(ptr.To("v1")))
			Expect(betaBuild.Spec.ParamValues[3].Values[1].ConfigMapValue.Name).To(Equal("cm-sub"))
			Expect(betaBuild.Spec.ParamValues[3].Values[2].SecretValue.Name).To(Equal("sec-sub"))

			// Output
			Expect(betaBuild.Spec.Output.Image).To(Equal("quay.io/example/app:tag"))
			Expect(betaBuild.Spec.Output.PushSecret).To(Equal(ptr.To("reg-secret")))
			Expect(betaBuild.Spec.Output.Insecure).To(Equal(ptr.To(false)))
			Expect(betaBuild.Spec.Output.Timestamp).To(Equal(ptr.To("Zero")))
			Expect(betaBuild.Spec.Output.Annotations).To(Equal(map[string]string{"release": "1.0"}))
			Expect(betaBuild.Spec.Output.Labels).To(Equal(map[string]string{"app": "demo"}))

			// Triggers
			Expect(betaBuild.Spec.Trigger.When).To(HaveLen(3))
			Expect(betaBuild.Spec.Trigger.When[0].Name).To(Equal("gh-trigger"))
			Expect(betaBuild.Spec.Trigger.When[0].Type).To(Equal(buildapi.GitHubWebHookTrigger))
			Expect(betaBuild.Spec.Trigger.When[0].GitHub.Branches).To(Equal([]string{"release-*"}))
			Expect(betaBuild.Spec.Trigger.When[1].Name).To(Equal("img-trigger"))
			Expect(betaBuild.Spec.Trigger.When[1].Image.Names).To(Equal([]string{"base-image:latest"}))
			Expect(betaBuild.Spec.Trigger.When[2].Name).To(Equal("obj-trigger"))
			Expect(betaBuild.Spec.Trigger.When[2].ObjectRef.Name).To(Equal("pipeline-obj"))

			// Volumes, Timeout, Env
			Expect(betaBuild.Spec.Volumes).To(HaveLen(1))
			Expect(betaBuild.Spec.Timeout).To(Equal(&timeout))
			Expect(betaBuild.Spec.Env).To(Equal([]corev1.EnvVar{{Name: "DEBUG", Value: "1"}}))

			// Status
			Expect(betaBuild.Status.Registered).To(Equal(ptr.To(corev1.ConditionTrue)))
			Expect(*betaBuild.Status.Reason).To(Equal(buildapi.SucceedStatus))
			Expect(betaBuild.Status.Message).To(Equal(ptr.To("ready")))
		})

		It("converts a Build from v1alpha1 to v1beta1 with BundleContainer source", func() {
			prune := buildapialpha.PruneAfterPull
			alphaBuild := &buildapialpha.Build{
				TypeMeta: metav1.TypeMeta{
					APIVersion: alphaGroupVersion,
					Kind:       buildKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "bundle-build",
				},
				Spec: buildapialpha.BuildSpec{
					Source: buildapialpha.Source{
						BundleContainer: &buildapialpha.BundleContainer{
							Image: "registry/bundle:1.0",
							Prune: &prune,
						},
						Credentials: &corev1.LocalObjectReference{Name: "pull-cred"},
					},
					Strategy: buildapialpha.Strategy{
						Name: "cluster-buildpacks",
						Kind: ptr.To(buildapialpha.ClusterBuildStrategyKind),
					},
				},
			}

			unstructuredObj := &unstructured.Unstructured{}
			mapito, err := runtime.DefaultUnstructuredConverter.ToUnstructured(alphaBuild)
			Expect(err).ToNot(HaveOccurred())
			unstructuredObj.SetUnstructuredContent(mapito)

			var betaBuild buildapi.Build
			err = convertBuildAlphaToBeta(ctx, &betaBuild, unstructuredObj)
			Expect(err).ToNot(HaveOccurred())

			Expect(betaBuild.Spec.Source.Type).To(Equal(buildapi.OCIArtifactType))
			Expect(betaBuild.Spec.Source.OCIArtifact.Image).To(Equal("registry/bundle:1.0"))
			Expect(betaBuild.Spec.Source.OCIArtifact.Prune).To(Equal((*buildapi.PruneOption)(&prune)))
			Expect(betaBuild.Spec.Source.OCIArtifact.PullSecret).To(Equal(ptr.To("pull-cred")))
			Expect(*betaBuild.Spec.Strategy.Kind).To(Equal(buildapi.ClusterBuildStrategyKind))
		})

		It("converts a Build from v1alpha1 to v1beta1 with LocalCopy Sources", func() {
			timeout := metav1.Duration{Duration: 3 * time.Minute}
			alphaBuild := &buildapialpha.Build{
				TypeMeta: metav1.TypeMeta{
					APIVersion: alphaGroupVersion,
					Kind:       buildKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "local-copy-build",
				},
				Spec: buildapialpha.BuildSpec{
					Sources: []buildapialpha.BuildSource{
						{
							Type:    buildapialpha.LocalCopy,
							Name:    "source-payload",
							Timeout: &timeout,
						},
					},
					Strategy: buildapialpha.Strategy{
						Name: "buildpacks",
					},
				},
			}

			unstructuredObj := &unstructured.Unstructured{}
			mapito, err := runtime.DefaultUnstructuredConverter.ToUnstructured(alphaBuild)
			Expect(err).ToNot(HaveOccurred())
			unstructuredObj.SetUnstructuredContent(mapito)

			var betaBuild buildapi.Build
			err = convertBuildAlphaToBeta(ctx, &betaBuild, unstructuredObj)
			Expect(err).ToNot(HaveOccurred())

			Expect(betaBuild.Spec.Source.Type).To(Equal(buildapi.LocalType))
			Expect(betaBuild.Spec.Source.Local.Name).To(Equal("source-payload"))
			Expect(betaBuild.Spec.Source.Local.Timeout).To(Equal(&timeout))
		})
	})

	Context("Round-trip Build conversions", func() {
		It("preserves data in a round trip beta -> alpha -> beta", func() {
			timeout := metav1.Duration{Duration: 12 * time.Minute}
			failedLimit := uint(4)

			originalBeta := &buildapi.Build{
				TypeMeta: metav1.TypeMeta{
					APIVersion: betaGroupVersion,
					Kind:       buildKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rt-build",
					Namespace: "test-ns",
				},
				Spec: buildapi.BuildSpec{
					Source: &buildapi.Source{
						Type:       buildapi.GitType,
						ContextDir: ptr.To("workspace"),
						Git: &buildapi.Git{
							URL:         "https://github.com/shipwright-io/sample-nodejs",
							Revision:    ptr.To("staging"),
							CloneSecret: ptr.To("git-auth"),
						},
					},
					Strategy: buildapi.Strategy{
						Name: "node-builder",
						Kind: ptr.To(buildapi.NamespacedBuildStrategyKind),
					},
					ParamValues: []buildapi.ParamValue{
						{
							Name: "NODE_ENV",
							SingleValue: &buildapi.SingleValue{
								Value: ptr.To("production"),
							},
						},
					},
					Output: buildapi.Image{
						Image:      "registry.io/org/node-app:1.0",
						PushSecret: ptr.To("push-auth"),
						Insecure:   ptr.To(true),
					},
					Retention: &buildapi.BuildRetention{
						AtBuildDeletion: ptr.To(true),
						FailedLimit:     &failedLimit,
					},
					Timeout: &timeout,
					Env: []corev1.EnvVar{
						{Name: "PORT", Value: "8080"},
					},
				},
				Status: buildapi.BuildStatus{
					Registered: ptr.To(corev1.ConditionTrue),
					Reason:     ptr.To(buildapi.SucceedStatus),
					Message:    ptr.To("valid"),
				},
			}

			// Beta -> Alpha
			alphaUnstructured := &unstructured.Unstructured{}
			err := convertBuildBetaToAlpha(ctx, originalBeta, alphaUnstructured)
			Expect(err).ToNot(HaveOccurred())

			// Alpha -> Beta
			var convertedBeta buildapi.Build
			err = convertBuildAlphaToBeta(ctx, &convertedBeta, alphaUnstructured)
			Expect(err).ToNot(HaveOccurred())

			Expect(convertedBeta.Name).To(Equal(originalBeta.Name))
			Expect(convertedBeta.Namespace).To(Equal(originalBeta.Namespace))
			Expect(convertedBeta.Spec.Source.Git.URL).To(Equal(originalBeta.Spec.Source.Git.URL))
			Expect(convertedBeta.Spec.Source.Git.Revision).To(Equal(originalBeta.Spec.Source.Git.Revision))
			Expect(convertedBeta.Spec.Source.Git.CloneSecret).To(Equal(originalBeta.Spec.Source.Git.CloneSecret))
			Expect(convertedBeta.Spec.Source.ContextDir).To(Equal(originalBeta.Spec.Source.ContextDir))
			Expect(convertedBeta.Spec.Strategy.Name).To(Equal(originalBeta.Spec.Strategy.Name))
			Expect(*convertedBeta.Spec.Strategy.Kind).To(Equal(buildapi.NamespacedBuildStrategyKind))
			Expect(convertedBeta.Spec.Output.Image).To(Equal(originalBeta.Spec.Output.Image))
			Expect(convertedBeta.Spec.Output.PushSecret).To(Equal(originalBeta.Spec.Output.PushSecret))
			Expect(convertedBeta.Spec.Output.Insecure).To(Equal(originalBeta.Spec.Output.Insecure))
			Expect(convertedBeta.Spec.Retention.AtBuildDeletion).To(Equal(originalBeta.Spec.Retention.AtBuildDeletion))
			Expect(convertedBeta.Spec.Retention.FailedLimit).To(Equal(originalBeta.Spec.Retention.FailedLimit))
			Expect(convertedBeta.Spec.Timeout).To(Equal(originalBeta.Spec.Timeout))
			Expect(convertedBeta.Spec.Env).To(Equal(originalBeta.Spec.Env))
			Expect(convertedBeta.Status.Registered).To(Equal(originalBeta.Status.Registered))
			Expect(*convertedBeta.Status.Reason).To(Equal(buildapi.SucceedStatus))
			Expect(convertedBeta.Status.Message).To(Equal(originalBeta.Status.Message))
		})
	})

	Context("BuildSpec convertBuildSpecBetaToAlpha edge cases", func() {
		It("does not panic when the source type is OCIArtifact but OCIArtifact is nil", func() {
			src := buildapi.BuildSpec{
				Source: &buildapi.Source{
					Type: buildapi.OCIArtifactType,
				},
			}

			dest := &buildapialpha.BuildSpec{}

			Expect(func() {
				Expect(convertBuildSpecBetaToAlpha(&src, dest)).To(Succeed())
			}).ToNot(Panic())

			Expect(dest.Source.BundleContainer).To(BeNil())
		})

		It("does not panic when the source type is Git but Git is nil", func() {
			src := buildapi.BuildSpec{
				Source: &buildapi.Source{
					Type: buildapi.GitType,
				},
			}

			dest := &buildapialpha.BuildSpec{}

			Expect(func() {
				Expect(convertBuildSpecBetaToAlpha(&src, dest)).To(Succeed())
			}).ToNot(Panic())

			Expect(dest.Source.URL).To(BeNil())
			Expect(dest.Source.Revision).To(BeNil())
		})
	})
})
