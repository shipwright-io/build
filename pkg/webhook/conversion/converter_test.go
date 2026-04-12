// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package conversion_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/utils/ptr"

	buildapialpha "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/webhook/conversion"
)

func getConversionReview(o string) (apiextensionsv1.ConversionReview, error) {
	convertReview := apiextensionsv1.ConversionReview{}
	response := httptest.NewRecorder()
	request, err := http.NewRequest("POST", "/convert", strings.NewReader(o))
	if err != nil {
		return convertReview, err
	}
	request.Header.Add("Content-Type", "application/yaml")

	conversion.CRDConvert(context.TODO(), response, request)

	scheme := runtime.NewScheme()

	yamlSerializer := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme, scheme)
	if _, _, err := yamlSerializer.Decode(response.Body.Bytes(), nil, &convertReview); err != nil {
		return convertReview, err
	}

	return convertReview, nil
}

var _ = Describe("ConvertCRD", func() {

	// common values across test cases
	var ctxDir = "docker-build"
	var apiVersion = "apiextensions.k8s.io/v1"
	var image = "dockerhub/foobar/hello"
	var secretName = "foobar"
	var url = "https://github.com/shipwright-io/sample-go"
	var revision = "main"
	var strategyName = "buildkit"
	var strategyKind = "ClusterBuildStrategy"

	Context("for a Build CR from v1beta1 to v1alpha1", func() {
		var desiredAPIVersion = "shipwright.io/v1alpha1"

		It("converts for spec source Local type", func() {
			// Create the yaml in v1beta1
			buildTemplate := `kind: ConversionReview
apiVersion: %s
request:
  uid: 0000-0000-0000-0000
  desiredAPIVersion: %s
  objects:
    - apiVersion: shipwright.io/v1beta1
      kind: Build
      metadata:
        name: buildkit-build
      spec:
        source:
          type: Local
          local:
            timeout: 1m
            name: foobar_local
        strategy:
          name: %s
          kind: %s
      status:
        message: "all validations succeeded"
        reason: Succeeded
        registered: "True"
`
			o := fmt.Sprintf(buildTemplate, apiVersion,
				desiredAPIVersion, strategyName, strategyKind)

			// Invoke the /convert webhook endpoint
			conversionReview, err := getConversionReview(o)
			Expect(err).To(BeNil())
			Expect(conversionReview.Response.Result.Status).To(Equal(v1.StatusSuccess))

			convertedObj, err := ToUnstructured(conversionReview)
			Expect(err).To(BeNil())

			build, err := toV1Alpha1BuildObject(convertedObj)
			Expect(err).To(BeNil())

			// Prepare our desired v1alpha1 Build
			desiredBuild := buildapialpha.Build{
				TypeMeta: v1.TypeMeta{
					APIVersion: "shipwright.io/v1alpha1",
					Kind:       "Build",
				},
				ObjectMeta: v1.ObjectMeta{
					Name: "buildkit-build",
				},
				Spec: buildapialpha.BuildSpec{
					Source: buildapialpha.Source{},
					Sources: []buildapialpha.BuildSource{
						{
							Name: "foobar_local",
							Type: buildapialpha.LocalCopy,
							Timeout: &v1.Duration{
								Duration: 1 * time.Minute,
							},
						},
					},
					Strategy: buildapialpha.Strategy{
						Name: strategyName,
						Kind: (*buildapialpha.BuildStrategyKind)(&strategyKind),
					},
				},
				Status: buildapialpha.BuildStatus{
					Message:    ptr.To("all validations succeeded"),
					Reason:     buildapialpha.BuildReasonPtr(buildapialpha.SucceedStatus),
					Registered: buildapialpha.ConditionStatusPtr(corev1.ConditionTrue),
				},
			}

			// Use ComparableTo and assert the whole object
			Expect(build).To(BeComparableTo(desiredBuild))
		})
		It("converts for spec source OCIArtifacts type, strategy and triggers", func() {
			branchMain := "main"
			branchDev := "develop"

			// Create the yaml in v1beta1
			buildTemplate := `kind: ConversionReview
apiVersion: %s
request:
  uid: 0000-0000-0000-0000
  desiredAPIVersion: %s
  objects:
    - apiVersion: shipwright.io/v1beta1
      kind: Build
      metadata:
        name: buildkit-build
      spec:
        source:
          type: OCI
          contextDir: %s
          ociArtifact:
            image: %s
            prune: AfterPull
            pullSecret: %s
        strategy:
          name: %s
          kind: %s
        trigger:
          when:
          - name:
            type: GitHub
            github:
              events:
              - Push
              branches:
              - %s
              - %s
          triggerSecret: %s
`
			o := fmt.Sprintf(buildTemplate, apiVersion,
				desiredAPIVersion, ctxDir,
				image, secretName,
				strategyName, strategyKind,
				branchMain, branchDev, secretName)

			// Invoke the /convert webhook endpoint
			conversionReview, err := getConversionReview(o)
			Expect(err).To(BeNil())
			Expect(conversionReview.Response.Result.Status).To(Equal(v1.StatusSuccess))

			convertedObj, err := ToUnstructured(conversionReview)
			Expect(err).To(BeNil())

			build, err := toV1Alpha1BuildObject(convertedObj)
			Expect(err).To(BeNil())

			// Prepare our desired v1alpha1 Build
			s := buildapialpha.PruneAfterPull
			desiredBuild := buildapialpha.Build{
				TypeMeta: v1.TypeMeta{
					APIVersion: "shipwright.io/v1alpha1",
					Kind:       "Build",
				},
				ObjectMeta: v1.ObjectMeta{
					Name: "buildkit-build",
				},
				Spec: buildapialpha.BuildSpec{
					Source: buildapialpha.Source{
						BundleContainer: &buildapialpha.BundleContainer{
							Image: image,
							Prune: &s,
						},
						Credentials: &corev1.LocalObjectReference{
							Name: secretName,
						},
						ContextDir: &ctxDir,
					},
					Strategy: buildapialpha.Strategy{
						Name: strategyName,
						Kind: (*buildapialpha.BuildStrategyKind)(&strategyKind),
					},
					Trigger: &buildapialpha.Trigger{
						When: []buildapialpha.TriggerWhen{
							{
								Name: "",
								Type: buildapialpha.GitHubWebHookTrigger,
								GitHub: &buildapialpha.WhenGitHub{
									Events: []buildapialpha.GitHubEventName{
										buildapialpha.GitHubPushEvent,
									},
									Branches: []string{
										branchMain,
										branchDev,
									},
								},
							},
						},
						SecretRef: &corev1.LocalObjectReference{
							Name: secretName,
						},
					},
				},
			}

			// Use ComparableTo and assert the whole object
			Expect(build).To(BeComparableTo(desiredBuild))
		})
		It("converts for spec source Git type, strategy, params and output", func() {
			// Create the yaml in v1beta1
			buildTemplate := `kind: ConversionReview
apiVersion: %s
request:
  uid: 0000-0000-0000-0000
  desiredAPIVersion: %s
  objects:
    - apiVersion: shipwright.io/v1beta1
      kind: Build
      metadata:
        name: buildkit-build
      spec:
        source:
          type: Git
          contextDir: %s
          git:
            url: %s
            revision: %s
            cloneSecret: %s
        strategy:
          name: %s
          kind: %s
        paramValues:
        - name: dockerfile
          value: Dockerfilefoobar
        - name: foo1
          value: disabled
        - name: foo2
          values:
          - secretValue:
              name: npm-registry-access
              key: npm-auth-token
              format: NPM_AUTH_TOKEN=${SECRET_VALUE}
        timeout: 10m
        output:
          image: %s
          pushSecret: %s
        retention:
          atBuildDeletion: true
`
			o := fmt.Sprintf(buildTemplate, apiVersion,
				desiredAPIVersion, ctxDir,
				url, revision,
				secretName, strategyName,
				strategyKind, image,
				secretName)

			// Invoke the /convert webhook endpoint
			conversionReview, err := getConversionReview(o)
			Expect(err).To(BeNil())
			Expect(conversionReview.Response.Result.Status).To(Equal(v1.StatusSuccess))

			convertedObj, err := ToUnstructured(conversionReview)
			Expect(err).To(BeNil())

			build, err := toV1Alpha1BuildObject(convertedObj)
			Expect(err).To(BeNil())

			// Prepare our desired v1alpha1 Build
			valDisable := "disabled"
			dockerfileVal := "Dockerfilefoobar"
			b := "NPM_AUTH_TOKEN=${SECRET_VALUE}"
			desiredBuild := buildapialpha.Build{
				TypeMeta: v1.TypeMeta{
					APIVersion: "shipwright.io/v1alpha1",
					Kind:       "Build",
				},
				ObjectMeta: v1.ObjectMeta{
					Name: "buildkit-build",
					Annotations: map[string]string{
						buildapialpha.AnnotationBuildRunDeletion: "true",
					},
				},
				Spec: buildapialpha.BuildSpec{
					Source: buildapialpha.Source{
						URL: &url,
						Credentials: &corev1.LocalObjectReference{
							Name: secretName,
						},
						Revision:   &revision,
						ContextDir: &ctxDir,
					},
					Dockerfile: &dockerfileVal,
					Strategy: buildapialpha.Strategy{
						Name: strategyName,
						Kind: (*buildapialpha.BuildStrategyKind)(&strategyKind),
					},
					Timeout: &v1.Duration{
						Duration: 10 * time.Minute,
					},
					ParamValues: []buildapialpha.ParamValue{
						{
							Name: "foo1",
							SingleValue: &buildapialpha.SingleValue{
								Value: &valDisable,
							},
						},
						{
							Name: "foo2",
							// todo: figure out why we need to set this one
							SingleValue: &buildapialpha.SingleValue{},
							Values: []buildapialpha.SingleValue{
								{
									SecretValue: &buildapialpha.ObjectKeyRef{
										Name:   "npm-registry-access",
										Key:    "npm-auth-token",
										Format: &b,
									},
								},
							},
						},
					},
					Output: buildapialpha.Image{
						Image: image,
						Credentials: &corev1.LocalObjectReference{
							Name: secretName,
						},
					},
				},
			}

			// Use ComparableTo and assert the whole object
			Expect(build).To(BeComparableTo(desiredBuild))
		})
		It("converts for spec source Git type, strategy, retention and volumes", func() {
			limit := uint(10)
			// Create the yaml in v1beta1
			buildTemplate := `kind: ConversionReview
apiVersion: %s
request:
  uid: 0000-0000-0000-0000
  desiredAPIVersion: %s
  objects:
    - apiVersion: shipwright.io/v1beta1
      kind: Build
      metadata:
        name: buildkit-build
      spec:
        source:
          type: Git
          contextDir: %s
          git:
            url: %s
            revision: %s
            cloneSecret: %s
        strategy:
          name: %s
          kind: %s
        retention:
          failedLimit: %v
          succeededLimit: %v
          ttlAfterFailed: 30m
          ttlAfterSucceeded: 30m
        volumes:
        - name: gocache
          emptyDir: {}
        - name: foobar
          emptyDir: {}
`
			o := fmt.Sprintf(buildTemplate, apiVersion,
				desiredAPIVersion, ctxDir,
				url, revision, secretName,
				strategyName, strategyKind,
				limit, limit)

			// Invoke the /convert webhook endpoint
			conversionReview, err := getConversionReview(o)
			Expect(err).To(BeNil())
			Expect(conversionReview.Response.Result.Status).To(Equal(v1.StatusSuccess))

			convertedObj, err := ToUnstructured(conversionReview)
			Expect(err).To(BeNil())

			build, err := toV1Alpha1BuildObject(convertedObj)
			Expect(err).To(BeNil())

			// Prepare our desired v1alpha1 Build
			desiredBuild := buildapialpha.Build{
				TypeMeta: v1.TypeMeta{
					APIVersion: "shipwright.io/v1alpha1",
					Kind:       "Build",
				},
				ObjectMeta: v1.ObjectMeta{
					Name: "buildkit-build",
				},
				Spec: buildapialpha.BuildSpec{
					Source: buildapialpha.Source{
						URL: &url,
						Credentials: &corev1.LocalObjectReference{
							Name: secretName,
						},
						Revision:   &revision,
						ContextDir: &ctxDir,
					},
					Strategy: buildapialpha.Strategy{
						Name: strategyName,
						Kind: (*buildapialpha.BuildStrategyKind)(&strategyKind),
					},
					Retention: &buildapialpha.BuildRetention{
						FailedLimit:    &limit,
						SucceededLimit: &limit,
						TTLAfterFailed: &v1.Duration{
							Duration: time.Minute * 30,
						},
						TTLAfterSucceeded: &v1.Duration{
							Duration: time.Minute * 30,
						},
					},
					Volumes: []buildapialpha.BuildVolume{
						{
							Name: "gocache",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "foobar",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			}

			// Use ComparableTo and assert the whole object
			Expect(build).To(BeComparableTo(desiredBuild))
		})
	})

	Context("for a Build CR from v1alpha1 to v1beta1", func() {
		var desiredAPIVersion = "shipwright.io/v1beta1"

		It("converts for spec sources local to source local", func() {
			// Create the yaml in v1alpha1
			// When source and sources are present, if sources with local type
			// exists, we will convert to Local type Source, and ignore the source url
			buildTemplate := `kind: ConversionReview
apiVersion: %s
request:
  uid: 0000-0000-0000-0000
  desiredAPIVersion: %s
  objects:
    - apiVersion: shipwright.io/v1alpha1
      kind: Build
      metadata:
        name: buildkit-build
      spec:
        source:
          url: fake_url
        sources:
        - name: foobar_local
          type: LocalCopy
          timeout: 1m
        - name: foobar_local_two
          type: LocalCopy
          timeout: 1m
`
			o := fmt.Sprintf(buildTemplate, apiVersion,
				desiredAPIVersion)

			// Invoke the /convert webhook endpoint
			conversionReview, err := getConversionReview(o)
			Expect(err).To(BeNil())
			Expect(conversionReview.Response.Result.Status).To(Equal(v1.StatusSuccess))

			convertedObj, err := ToUnstructured(conversionReview)
			Expect(err).To(BeNil())

			build, err := toV1Beta1BuildObject(convertedObj)
			Expect(err).To(BeNil())

			// Prepare our desired v1beta1 Build
			desiredBuild := buildapi.Build{
				TypeMeta: v1.TypeMeta{
					APIVersion: "shipwright.io/v1beta1",
					Kind:       "Build",
				},
				ObjectMeta: v1.ObjectMeta{
					Name: "buildkit-build",
				},
				Spec: buildapi.BuildSpec{
					Source: &buildapi.Source{
						Type: buildapi.LocalType,
						Local: &buildapi.Local{
							Name: "foobar_local",
							Timeout: &v1.Duration{
								Duration: 1 * time.Minute,
							},
						},
					},
				},
			}

			// Use ComparableTo and assert the whole object
			Expect(build).To(BeComparableTo(desiredBuild))

		})

		It("converts for spec bundleContainer source type, triggers and output", func() {
			pruneOption := "Never"
			branchMain := "main"
			branchDev := "develop"
			// Create the yaml in v1alpha1
			buildTemplate := `kind: ConversionReview
apiVersion: %s
request:
  uid: 0000-0000-0000-0000
  desiredAPIVersion: %s
  objects:
    - apiVersion: shipwright.io/v1alpha1
      kind: Build
      metadata:
        name: buildkit-build
        annotations:
          build.shipwright.io/build-run-deletion: "true"
      spec:
        source:
          contextDir: %s
          bundleContainer:
            image: %s
            prune: %s
          credentials:
            name: %s
        dockerfile: Dockerfile
        trigger:
          when:
          - name:
            type: GitHub
            github:
              events:
              - Push
              branches:
              - %s
              - %s
          secretRef:
            name: %s
        timeout: 10m
        output:
          image: %s
          credentials:
            name: %s
          labels:
            foo: bar
          annotations:
            foo: bar
`
			o := fmt.Sprintf(buildTemplate, apiVersion,
				desiredAPIVersion, ctxDir,
				image, pruneOption,
				secretName, branchMain, branchDev,
				secretName, image, secretName,
			)

			// Invoke the /convert webhook endpoint
			conversionReview, err := getConversionReview(o)
			Expect(err).To(BeNil())
			Expect(conversionReview.Response.Result.Status).To(Equal(v1.StatusSuccess))

			convertedObj, err := ToUnstructured(conversionReview)
			Expect(err).To(BeNil())

			build, err := toV1Beta1BuildObject(convertedObj)
			Expect(err).To(BeNil())

			// Prepare our desired v1beta1 Build
			pruneNever := buildapi.PruneNever
			dockerfileVal := "Dockerfile"
			desiredBuild := buildapi.Build{
				TypeMeta: v1.TypeMeta{
					APIVersion: "shipwright.io/v1beta1",
					Kind:       "Build",
				},
				ObjectMeta: v1.ObjectMeta{
					Name: "buildkit-build",
				},
				Spec: buildapi.BuildSpec{
					Source: &buildapi.Source{
						Type:       buildapi.OCIArtifactType,
						ContextDir: &ctxDir,
						OCIArtifact: &buildapi.OCIArtifact{
							Image:      image,
							Prune:      &pruneNever,
							PullSecret: &secretName,
						},
					},
					ParamValues: []buildapi.ParamValue{
						{
							Name: "dockerfile",
							SingleValue: &buildapi.SingleValue{
								Value: &dockerfileVal,
							},
						},
					},
					Retention: &buildapi.BuildRetention{
						AtBuildDeletion: ptr.To(true),
					},
					Trigger: &buildapi.Trigger{
						When: []buildapi.TriggerWhen{
							{
								Name: "",
								Type: buildapi.GitHubWebHookTrigger,
								GitHub: &buildapi.WhenGitHub{
									Events: []buildapi.GitHubEventName{
										buildapi.GitHubPushEvent,
									},
									Branches: []string{
										branchMain,
										branchDev,
									},
								},
							},
						},
						TriggerSecret: &secretName,
					},
					Timeout: &v1.Duration{
						Duration: time.Minute * 10,
					},
					Output: buildapi.Image{
						Image:      image,
						PushSecret: &secretName,
						Labels: map[string]string{
							"foo": "bar",
						},
						Annotations: map[string]string{
							"foo": "bar",
						},
					},
				},
			}

			// Use ComparableTo and assert the whole object
			Expect(build).To(BeComparableTo(desiredBuild))

		})

		It("converts for spec url source type, and params", func() {
			// Create the yaml in v1alpha1
			buildTemplate := `kind: ConversionReview
apiVersion: %s
request:
  uid: 0000-0000-0000-0000
  desiredAPIVersion: %s
  objects:
  - apiVersion: shipwright.io/v1alpha1
    kind: Build
    metadata:
      name: buildkit-build
    spec:
      source:
        contextDir: %s
        revision: %s
        url: %s
        credentials:
          name: %s
      paramValues:
      - name: foo1
        value: disabled
      - name: foo2
        values:
        - secretValue:
            name: npm-registry-access
            key: npm-auth-token
            format: NPM_AUTH_TOKEN=${SECRET_VALUE}
`
			o := fmt.Sprintf(buildTemplate, apiVersion,
				desiredAPIVersion, ctxDir,
				revision, url,
				secretName)

			// Invoke the /convert webhook endpoint
			conversionReview, err := getConversionReview(o)

			Expect(err).To(BeNil())
			Expect(conversionReview.Response.Result.Status).To(Equal(v1.StatusSuccess))

			convertedObj, err := ToUnstructured(conversionReview)
			Expect(err).To(BeNil())

			build, err := toV1Beta1BuildObject(convertedObj)
			Expect(err).To(BeNil())

			// Prepare our desired v1beta1 Build
			valDisable := "disabled"
			b := "NPM_AUTH_TOKEN=${SECRET_VALUE}"
			desiredBuild := buildapi.Build{
				TypeMeta: v1.TypeMeta{
					APIVersion: "shipwright.io/v1beta1",
					Kind:       "Build",
				},
				ObjectMeta: v1.ObjectMeta{
					Name: "buildkit-build",
				},
				Spec: buildapi.BuildSpec{
					Source: &buildapi.Source{
						Type:       buildapi.GitType,
						ContextDir: &ctxDir,
						Git: &buildapi.Git{
							URL:         url,
							Revision:    &revision,
							CloneSecret: &secretName,
						},
					},
					ParamValues: []buildapi.ParamValue{
						{
							Name: "foo1",
							SingleValue: &buildapi.SingleValue{
								Value: &valDisable,
							},
						},
						{
							Name: "foo2",
							// todo: figure out why we need to set this one
							SingleValue: &buildapi.SingleValue{},
							Values: []buildapi.SingleValue{
								{
									SecretValue: &buildapi.ObjectKeyRef{
										Name:   "npm-registry-access",
										Key:    "npm-auth-token",
										Format: &b,
									},
								},
							},
						},
					},
				},
			}

			// Use ComparableTo and assert the whole object
			Expect(build).To(BeComparableTo(desiredBuild))
		})
	})

	Context("for a BuildRun CR from v1beta1 to v1alpha1", func() {
		var desiredAPIVersion = "shipwright.io/v1alpha1"

		It("converts for spec source", func() {
			// Create the yaml in v1beta1
			buildTemplate := `kind: ConversionReview
apiVersion: %s
request:
  uid: 0000-0000-0000-0000
  desiredAPIVersion: %s
  objects:
    - apiVersion: shipwright.io/v1beta1
      kind: BuildRun
      metadata:
        name: buildkit-run
      spec:
        build:
          name: a_build
        source:
          type: Local
          local:
            name: foobar_local
            timeout: 1m
`
			o := fmt.Sprintf(buildTemplate, apiVersion,
				desiredAPIVersion)

			// Invoke the /convert webhook endpoint
			conversionReview, err := getConversionReview(o)
			Expect(err).To(BeNil())
			Expect(conversionReview.Response.Result.Status).To(Equal(v1.StatusSuccess))

			convertedObj, err := ToUnstructured(conversionReview)
			Expect(err).To(BeNil())

			buildRun, err := toV1Alpha1BuildRunObject(convertedObj)
			Expect(err).To(BeNil())

			// Prepare our desired v1alpha1 BuildRun
			desiredBuildRun := buildapialpha.BuildRun{
				ObjectMeta: v1.ObjectMeta{
					Name: "buildkit-run",
				},
				TypeMeta: v1.TypeMeta{
					APIVersion: "shipwright.io/v1alpha1",
					Kind:       "BuildRun",
				},
				Spec: buildapialpha.BuildRunSpec{
					BuildRef: &buildapialpha.BuildRef{
						Name: "a_build",
					},
					Sources: []buildapialpha.BuildSource{
						{
							Name: "foobar_local",
							Type: buildapialpha.LocalCopy,
							Timeout: &v1.Duration{
								Duration: 1 * time.Minute,
							},
						},
					},
					ServiceAccount: &buildapialpha.ServiceAccount{},
				},
			}

			// Use ComparableTo and assert the whole object
			Expect(buildRun).To(BeComparableTo(desiredBuildRun))
		})

		It("converts for spec Build spec", func() {
			pruneOption := "AfterPull"
			sa := "foobar"
			// Create the yaml in v1beta1s
			buildTemplate := `kind: ConversionReview
apiVersion: %s
request:
  uid: 0000-0000-0000-0000
  desiredAPIVersion: %s
  objects:
    - apiVersion: shipwright.io/v1beta1
      kind: BuildRun
      metadata:
        name: buildkit-run
      spec:
        build:
          spec:
            source:
              type: OCI
              contextDir: %s
              ociArtifact:
                image: %s
                prune: %s
                pullSecret: %s
        serviceAccount: %s
        timeout: 10m
        paramValues:
        - name: foobar
          value: bar
        output:
          image: %s
          pushSecret: %s
          annotations:
            foo: bar
          labels:
            foo2: bar2
        env:
        - name: one
          value: two
        retention:
          ttlAfterFailed: 10m
        volumes:
        - name: volume1
          emptyDir: {}
`
			o := fmt.Sprintf(buildTemplate, apiVersion,
				desiredAPIVersion, ctxDir,
				image, pruneOption,
				secretName, sa, image,
				secretName,
			)

			// Invoke the /convert webhook endpoint
			conversionReview, err := getConversionReview(o)
			Expect(err).To(BeNil())
			Expect(conversionReview.Response.Result.Status).To(Equal(v1.StatusSuccess))

			convertedObj, err := ToUnstructured(conversionReview)
			Expect(err).To(BeNil())

			buildRun, err := toV1Alpha1BuildRunObject(convertedObj)
			Expect(err).To(BeNil())

			// Prepare our desired v1alpha1 BuildRun
			s := buildapialpha.PruneAfterPull
			paramVal := "bar"
			desiredBuildRun := buildapialpha.BuildRun{
				ObjectMeta: v1.ObjectMeta{
					Name: "buildkit-run",
				},
				TypeMeta: v1.TypeMeta{
					APIVersion: "shipwright.io/v1alpha1",
					Kind:       "BuildRun",
				},
				Spec: buildapialpha.BuildRunSpec{
					BuildSpec: &buildapialpha.BuildSpec{
						Source: buildapialpha.Source{
							BundleContainer: &buildapialpha.BundleContainer{
								Image: image,
								Prune: &s,
							},
							ContextDir: &ctxDir,
							Credentials: &corev1.LocalObjectReference{
								Name: secretName,
							},
						},
					},
					ServiceAccount: &buildapialpha.ServiceAccount{
						Name: &sa,
					},
					Timeout: &v1.Duration{
						Duration: time.Minute * 10,
					},
					ParamValues: []buildapialpha.ParamValue{
						{
							Name: "foobar",
							SingleValue: &buildapialpha.SingleValue{
								Value: &paramVal,
							},
						},
					},
					Output: &buildapialpha.Image{
						Image: image,
						Credentials: &corev1.LocalObjectReference{
							Name: secretName,
						},
						Annotations: map[string]string{
							"foo": "bar",
						},
						Labels: map[string]string{
							"foo2": "bar2",
						},
					},
					Env: []corev1.EnvVar{
						{
							Name:  "one",
							Value: "two",
						},
					},
					Retention: &buildapialpha.BuildRunRetention{
						TTLAfterFailed: &v1.Duration{
							Duration: time.Minute * 10,
						},
					},
					Volumes: []buildapialpha.BuildVolume{
						{
							Name: "volume1",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			}

			// Use ComparableTo and assert the whole object
			Expect(buildRun).To(BeComparableTo(desiredBuildRun))
		})

		It("converts for spec Build name ", func() {
			refBuild := "buildkit-build"
			sa := "foobar"
			// Create the yaml in v1beta1s
			buildTemplate := `kind: ConversionReview
apiVersion: %s
request:
  uid: 0000-0000-0000-0000
  desiredAPIVersion: %s
  objects:
    - apiVersion: shipwright.io/v1beta1
      kind: BuildRun
      metadata:
        name: buildkit-run
      spec:
        build:
          name: %s
        serviceAccount: %s
`
			o := fmt.Sprintf(buildTemplate, apiVersion,
				desiredAPIVersion, refBuild, sa)

			// Invoke the /convert webhook endpoint
			conversionReview, err := getConversionReview(o)
			Expect(err).To(BeNil())
			Expect(conversionReview.Response.Result.Status).To(Equal(v1.StatusSuccess))

			convertedObj, err := ToUnstructured(conversionReview)
			Expect(err).To(BeNil())

			buildRun, err := toV1Alpha1BuildRunObject(convertedObj)
			Expect(err).To(BeNil())

			// Prepare our desired v1alpha1 BuildRun
			desiredBuildRun := buildapialpha.BuildRun{
				ObjectMeta: v1.ObjectMeta{
					Name: "buildkit-run",
				},
				TypeMeta: v1.TypeMeta{
					APIVersion: "shipwright.io/v1alpha1",
					Kind:       "BuildRun",
				},
				Spec: buildapialpha.BuildRunSpec{
					BuildRef: &buildapialpha.BuildRef{
						Name: refBuild,
					},
					ServiceAccount: &buildapialpha.ServiceAccount{
						Name: &sa,
					},
				},
			}

			// Use ComparableTo and assert the whole object
			Expect(buildRun).To(BeComparableTo(desiredBuildRun))
		})

		It("converts for status", func() {
			// Create the yaml in v1beta1
			buildTemplate := `kind: ConversionReview
apiVersion: %s
request:
  uid: 0000-0000-0000-0000
  desiredAPIVersion: %s
  objects:
    - apiVersion: shipwright.io/v1beta1
      kind: BuildRun
      metadata:
        name: buildkit-run
      spec:
        build:
          name: a_build
        source:
          type: Local
          local:
            name: foobar_local
            timeout: 1m
      status:
        buildSpec:
          output:
            image: somewhere
            pushSecret: some-secret
          paramValues:
            - name: dockerfile
              value: Dockerfile
          source:
            git:
              url: https://github.com/shipwright-io/sample-go
            type: Git
          strategy:
            kind: ClusterBuildStrategy
            name: buildkit
          timeout: 10m0s
        completionTime: "2023-10-17T07:35:10Z"
        conditions:
          - lastTransitionTime: "2023-10-17T07:35:10Z"
            message: All Steps have completed executing
            reason: Succeeded
            status: "True"
            type: Succeeded
        output:
          digest: sha256:9befa6f5f7142a5bf92174b54bb6e0a1dd04e5252aa9dc8f6962f6da966f68a8
        source:
          git:
            commitAuthor: somebody
            commitSha: 6a45e68454ca0f319b1a82c65bea09a10fa9eec6
        startTime: "2023-10-17T07:31:55Z"
        taskRunName: buildkit-run-n5sxr
`
			o := fmt.Sprintf(buildTemplate, apiVersion,
				desiredAPIVersion)

			// Invoke the /convert webhook endpoint
			conversionReview, err := getConversionReview(o)
			Expect(err).To(BeNil())
			Expect(conversionReview.Response.Result.Status).To(Equal(v1.StatusSuccess))

			convertedObj, err := ToUnstructured(conversionReview)
			Expect(err).To(BeNil())

			buildRun, err := toV1Alpha1BuildRunObject(convertedObj)
			Expect(err).To(BeNil())

			// Prepare our desired v1alpha1 BuildRun
			completionTime, err := time.Parse(time.RFC3339, "2023-10-17T07:35:10Z")
			Expect(err).To(BeNil())
			startTime, err := time.Parse(time.RFC3339, "2023-10-17T07:31:55Z")
			Expect(err).To(BeNil())
			buildStrategyKind := buildapialpha.BuildStrategyKind(buildapialpha.ClusterBuildStrategyKind)
			desiredBuildRun := buildapialpha.BuildRun{
				ObjectMeta: v1.ObjectMeta{
					Name: "buildkit-run",
				},
				TypeMeta: v1.TypeMeta{
					APIVersion: "shipwright.io/v1alpha1",
					Kind:       "BuildRun",
				},
				Spec: buildapialpha.BuildRunSpec{
					BuildRef: &buildapialpha.BuildRef{
						Name: "a_build",
					},
					Sources: []buildapialpha.BuildSource{
						{
							Name: "foobar_local",
							Type: buildapialpha.LocalCopy,
							Timeout: &v1.Duration{
								Duration: 1 * time.Minute,
							},
						},
					},
					ServiceAccount: &buildapialpha.ServiceAccount{},
				},
				Status: buildapialpha.BuildRunStatus{
					BuildSpec: &buildapialpha.BuildSpec{
						Source: buildapialpha.Source{
							URL: ptr.To("https://github.com/shipwright-io/sample-go"),
						},
						Dockerfile: ptr.To("Dockerfile"),
						Output: buildapialpha.Image{
							Image: "somewhere",
							Credentials: &corev1.LocalObjectReference{
								Name: "some-secret",
							},
						},
						Strategy: buildapialpha.Strategy{
							Kind: &buildStrategyKind,
							Name: "buildkit",
						},
						Timeout: &v1.Duration{
							Duration: 10 * time.Minute,
						},
					},
					CompletionTime: &v1.Time{
						Time: completionTime,
					},
					Conditions: buildapialpha.Conditions{{
						LastTransitionTime: v1.Time{
							Time: completionTime,
						},
						Message: "All Steps have completed executing",
						Reason:  "Succeeded",
						Status:  corev1.ConditionTrue,
						Type:    buildapialpha.Succeeded,
					}},
					LatestTaskRunRef: ptr.To("buildkit-run-n5sxr"),
					Output: &buildapialpha.Output{
						Digest: "sha256:9befa6f5f7142a5bf92174b54bb6e0a1dd04e5252aa9dc8f6962f6da966f68a8",
					},
					Sources: []buildapialpha.SourceResult{{
						Name: "default",
						Git: &buildapialpha.GitSourceResult{
							CommitAuthor: "somebody",
							CommitSha:    "6a45e68454ca0f319b1a82c65bea09a10fa9eec6",
						},
					}},
					StartTime: &v1.Time{
						Time: startTime,
					},
				},
			}

			// Use ComparableTo and assert the whole object
			Expect(buildRun).To(BeComparableTo(desiredBuildRun))
		})
	})
	Context("for a BuildRun CR from v1alpha1 to v1beta1", func() {
		var desiredAPIVersion = "shipwright.io/v1beta1"

		It("converts for spec source", func() {
			// Create the yaml in v1alpha1
			buildTemplate := `kind: ConversionReview
apiVersion: %s
request:
  uid: 0000-0000-0000-0000
  desiredAPIVersion: %s
  objects:
    - apiVersion: shipwright.io/v1alpha1
      kind: BuildRun
      metadata:
        name: buildkit-run
      spec:
        buildRef:
          name: a_build
        sources:
        - name: foobar_local
          type: LocalCopy
          timeout: 1m
`
			o := fmt.Sprintf(buildTemplate, apiVersion,
				desiredAPIVersion)

			// Invoke the /convert webhook endpoint
			conversionReview, err := getConversionReview(o)
			Expect(err).To(BeNil())
			Expect(conversionReview.Response.Result.Status).To(Equal(v1.StatusSuccess))

			convertedObj, err := ToUnstructured(conversionReview)
			Expect(err).To(BeNil())

			buildRun, err := toV1Beta1BuildRunObject(convertedObj)
			Expect(err).To(BeNil())

			// Prepare our desired v1alpha1 BuildRun
			desiredBuildRun := buildapi.BuildRun{
				ObjectMeta: v1.ObjectMeta{
					Name: "buildkit-run",
				},
				TypeMeta: v1.TypeMeta{
					APIVersion: "shipwright.io/v1beta1",
					Kind:       "BuildRun",
				},
				Spec: buildapi.BuildRunSpec{
					Build: buildapi.ReferencedBuild{
						Name: ptr.To("a_build"),
					},
					Source: &buildapi.BuildRunSource{
						Type: buildapi.LocalType,
						Local: &buildapi.Local{
							Name: "foobar_local",
							Timeout: &v1.Duration{
								Duration: 1 * time.Minute,
							},
						},
					},
				},
			}

			// Use ComparableTo and assert the whole object
			Expect(buildRun).To(BeComparableTo(desiredBuildRun))
		})

		It("converts for spec a generated serviceAccount", func() {
			// Create the yaml in v1alpha1
			buildRunTemplate := `kind: ConversionReview
apiVersion: %s
request:
  uid: 0000-0000-0000-0000
  desiredAPIVersion: %s
  objects:
    - apiVersion: shipwright.io/v1alpha1
      kind: BuildRun
      metadata:
        name: buildkit-run
      spec:
        buildRef:
          name: a_build
        serviceAccount:
          generate: true
        output:
          image: foobar
`
			o := fmt.Sprintf(buildRunTemplate, apiVersion, desiredAPIVersion)

			// Invoke the /convert webhook endpoint
			conversionReview, err := getConversionReview(o)
			Expect(err).To(BeNil())
			Expect(conversionReview.Response.Result.Status).To(Equal(v1.StatusSuccess))

			convertedObj, err := ToUnstructured(conversionReview)
			Expect(err).To(BeNil())

			buildRun, err := toV1Beta1BuildRunObject(convertedObj)
			Expect(err).To(BeNil())

			// Prepare our desired v1beta1 BuildRun
			desiredBuildRun := buildapi.BuildRun{
				ObjectMeta: v1.ObjectMeta{
					Name: "buildkit-run",
				},
				TypeMeta: v1.TypeMeta{
					APIVersion: "shipwright.io/v1beta1",
					Kind:       "BuildRun",
				},
				Spec: buildapi.BuildRunSpec{
					Build: buildapi.ReferencedBuild{
						Name: ptr.To("a_build"),
					},
					ServiceAccount: ptr.To(".generate"),
					Output: &buildapi.Image{
						Image: "foobar",
					},
				},
			}

			// Use ComparableTo and assert the whole object
			Expect(buildRun).To(BeComparableTo(desiredBuildRun))
		})

		It("converts for spec Build buildref", func() {
			// Create the yaml in v1alpha1
			buildTemplate := `kind: ConversionReview
apiVersion: %s
request:
  uid: 0000-0000-0000-0000
  desiredAPIVersion: %s
  objects:
    - apiVersion: shipwright.io/v1alpha1
      kind: BuildRun
      metadata:
        name: buildkit-run
      spec:
        buildRef:
          name: a_build
        serviceAccount:
          name: foobar
        timeout: 10m
        paramValues:
        - name: cache
          value: registry
        volumes:
        - name: volume-name
          configMap:
            name: test-config
        retention:
          ttlAfterFailed: 10m
          ttlAfterSucceeded: 10m
        output:
          image: foobar
          credentials:
            name: foobar
          labels:
            foo: bar
        env:
        - name: foo
          value: bar
`
			o := fmt.Sprintf(buildTemplate, apiVersion,
				desiredAPIVersion)

			// Invoke the /convert webhook endpoint
			conversionReview, err := getConversionReview(o)
			Expect(err).To(BeNil())
			Expect(conversionReview.Response.Result.Status).To(Equal(v1.StatusSuccess))

			convertedObj, err := ToUnstructured(conversionReview)
			Expect(err).To(BeNil())

			buildRun, err := toV1Beta1BuildRunObject(convertedObj)
			Expect(err).To(BeNil())

			// Prepare our desired v1alpha1 BuildRun
			sa := "foobar"
			paramVal := "registry"
			desiredBuildRun := buildapi.BuildRun{
				ObjectMeta: v1.ObjectMeta{
					Name: "buildkit-run",
				},
				TypeMeta: v1.TypeMeta{
					APIVersion: "shipwright.io/v1beta1",
					Kind:       "BuildRun",
				},
				Spec: buildapi.BuildRunSpec{
					Build: buildapi.ReferencedBuild{
						Name: ptr.To("a_build"),
					},
					ServiceAccount: &sa,
					Timeout: &v1.Duration{
						Duration: 10 * time.Minute,
					},
					ParamValues: []buildapi.ParamValue{
						{
							Name: "cache",
							SingleValue: &buildapi.SingleValue{
								Value: &paramVal,
							},
						},
					},
					Output: &buildapi.Image{
						Image: "foobar",
						Labels: map[string]string{
							"foo": "bar",
						},
						PushSecret: &secretName,
					},
					Env: []corev1.EnvVar{
						{
							Name:  "foo",
							Value: "bar",
						},
					},
					Retention: &buildapi.BuildRunRetention{
						TTLAfterFailed: &v1.Duration{
							Duration: time.Minute * 10,
						},
						TTLAfterSucceeded: &v1.Duration{
							Duration: time.Minute * 10,
						},
					},
					Volumes: []buildapi.BuildVolume{
						{
							Name: "volume-name",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "test-config",
									},
								},
							},
						},
					},
				},
			}

			// Use ComparableTo and assert the whole object
			Expect(buildRun).To(BeComparableTo(desiredBuildRun))
		})
	})

	Context("for a BuildStrategy spec from v1beta1 to v1alpha1", func() {
		var desiredAPIVersion = "shipwright.io/v1alpha1"
		It("converts the strategy", func() {
			// Create the yaml in v1beta1
			buildTemplate := `kind: ConversionReview
apiVersion: %s
request:
  uid: 0000-0000-0000-0000
  desiredAPIVersion: %s
  objects:
    - apiVersion: shipwright.io/v1beta1
      kind: BuildStrategy
      metadata:
        name: buildkit
      spec:
        steps:
        - name: step-foobar
          image: foobar
          command:
          - some-command
          args:
          - $(params.dockerfile)
          securityContext:
            privileged: false
        parameters:
        - name: param_one
          description: foobar
          type: string
        - name: param_two
          description: foobar
          type: array
        - name: dockerfile
          description: The Dockerfile to build.
          type: string
          default: Dockerfile
        securityContext:
          runAsUser: 1000
          runAsGroup: 1000
        volumes:
        - name: foobar
          overridable: false
          description: nonedescription
`
			o := fmt.Sprintf(buildTemplate, apiVersion,
				desiredAPIVersion)

			// Invoke the /convert webhook endpoint
			conversionReview, err := getConversionReview(o)
			Expect(err).To(BeNil())
			Expect(conversionReview.Response.Result.Status).To(Equal(v1.StatusSuccess))

			convertedObj, err := ToUnstructured(conversionReview)
			Expect(err).To(BeNil())

			buildStrategy, err := toV1Alpha1BuildStrategyObject(convertedObj)
			Expect(err).To(BeNil())

			// Prepare our desired v1alpha1 BuildStrategy
			privileged := false
			volDescription := "nonedescription"
			desiredBuildStrategy := buildapialpha.BuildStrategy{
				ObjectMeta: v1.ObjectMeta{
					Name: "buildkit",
				},
				TypeMeta: v1.TypeMeta{
					APIVersion: "shipwright.io/v1alpha1",
					Kind:       "BuildStrategy",
				},
				Spec: buildapialpha.BuildStrategySpec{
					BuildSteps: []buildapialpha.BuildStep{
						{
							Container: corev1.Container{
								Name:    "step-foobar",
								Command: []string{"some-command"},
								Args:    []string{"$(params.DOCKERFILE)"},
								Image:   "foobar",
								SecurityContext: &corev1.SecurityContext{
									Privileged: &privileged,
								},
							},
						},
					},
					Parameters: []buildapialpha.Parameter{
						{
							Name:        "param_one",
							Description: "foobar",
							Type:        buildapialpha.ParameterTypeString,
						},
						{
							Name:        "param_two",
							Description: "foobar",
							Type:        buildapialpha.ParameterTypeArray,
						},
					},
					SecurityContext: &buildapialpha.BuildStrategySecurityContext{
						RunAsUser:  1000,
						RunAsGroup: 1000,
					},
					Volumes: []buildapialpha.BuildStrategyVolume{
						{
							Name:        "foobar",
							Overridable: &privileged,
							Description: &volDescription,
						},
					},
				},
			}

			// Use ComparableTo and assert the whole object
			Expect(buildStrategy).To(BeComparableTo(desiredBuildStrategy))
		})
	})
	Context("for a BuildStrategy spec from v1alpha1 to v1beta1", func() {
		var desiredAPIVersion = "shipwright.io/v1beta1"
		It("converts the strategy", func() {
			// Create the yaml in v1alpha1
			buildTemplate := `kind: ConversionReview
apiVersion: %s
request:
  uid: 0000-0000-0000-0000
  desiredAPIVersion: %s
  objects:
    - apiVersion: shipwright.io/v1alpha1
      kind: BuildStrategy
      metadata:
        name: buildkit
      spec:
        buildSteps:
        - name: step-foobar
          command:
          - some-command
          args:
          - $(params.DOCKERFILE)
          image: foobar
          securityContext:
            privileged: false
        parameters:
        - name: param_one
          description: foobar
          type: string
        - name: param_two
          description: foobar
          type: array
        securityContext:
          runAsUser: 1000
          runAsGroup: 1000
        volumes:
        - name: foobar
          overridable: false
          description: nonedescription
`
			o := fmt.Sprintf(buildTemplate, apiVersion,
				desiredAPIVersion)

			// Invoke the /convert webhook endpoint
			conversionReview, err := getConversionReview(o)
			Expect(err).To(BeNil())
			Expect(conversionReview.Response.Result.Status).To(Equal(v1.StatusSuccess))

			convertedObj, err := ToUnstructured(conversionReview)
			Expect(err).To(BeNil())

			buildStrategy, err := toV1Beta1BuildStrategyObject(convertedObj)
			Expect(err).To(BeNil())

			// Prepare our desired v1alpha1 BuildStrategy
			privileged := false
			volDescription := "nonedescription"
			desiredBuildStrategy := buildapi.BuildStrategy{
				ObjectMeta: v1.ObjectMeta{
					Name: "buildkit",
				},
				TypeMeta: v1.TypeMeta{
					APIVersion: "shipwright.io/v1beta1",
					Kind:       "BuildStrategy",
				},
				Spec: buildapi.BuildStrategySpec{
					Steps: []buildapi.Step{
						{
							Name:    "step-foobar",
							Command: []string{"some-command"},
							Args:    []string{"$(params.dockerfile)"},
							Image:   "foobar",
							SecurityContext: &corev1.SecurityContext{
								Privileged: &privileged,
							},
						},
					},
					Parameters: []buildapi.Parameter{
						{
							Name:        "param_one",
							Description: "foobar",
							Type:        buildapi.ParameterTypeString,
						},
						{
							Name:        "param_two",
							Description: "foobar",
							Type:        buildapi.ParameterTypeArray,
						},
						{
							Name:        "dockerfile",
							Description: "The Dockerfile to be built.",
							Type:        buildapi.ParameterTypeString,
							Default:     ptr.To("Dockerfile"),
						},
					},
					SecurityContext: &buildapi.BuildStrategySecurityContext{
						RunAsUser:  1000,
						RunAsGroup: 1000,
					},
					Volumes: []buildapi.BuildStrategyVolume{
						{
							Name:        "foobar",
							Overridable: &privileged,
							Description: &volDescription,
						},
					},
				},
			}

			// Use ComparableTo and assert the whole object
			Expect(buildStrategy).To(BeComparableTo(desiredBuildStrategy))
		})
	})
})

func ToUnstructured(conversionReview apiextensionsv1.ConversionReview) (unstructured.Unstructured, error) {
	convertedObj := unstructured.Unstructured{}

	scheme := runtime.NewScheme()
	yamlSerializer := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme, scheme)
	if _, _, err := yamlSerializer.Decode(conversionReview.Response.ConvertedObjects[0].Raw, nil, &convertedObj); err != nil {
		return convertedObj, err
	}
	return convertedObj, nil
}

func toV1Alpha1BuildObject(convertedObject unstructured.Unstructured) (buildapialpha.Build, error) {
	var build buildapialpha.Build
	u := convertedObject.UnstructuredContent()
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u, &build); err != nil {
		return build, err
	}
	return build, nil
}

func toV1Alpha1BuildRunObject(convertedObject unstructured.Unstructured) (buildapialpha.BuildRun, error) {
	var build buildapialpha.BuildRun
	u := convertedObject.UnstructuredContent()
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u, &build); err != nil {
		return build, err
	}
	return build, nil
}

func toV1Beta1BuildRunObject(convertedObject unstructured.Unstructured) (buildapi.BuildRun, error) {
	var build buildapi.BuildRun
	u := convertedObject.UnstructuredContent()
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u, &build); err != nil {
		return build, err
	}
	return build, nil
}

func toV1Beta1BuildStrategyObject(convertedObject unstructured.Unstructured) (buildapi.BuildStrategy, error) {
	var buildStrategy buildapi.BuildStrategy
	u := convertedObject.UnstructuredContent()
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u, &buildStrategy); err != nil {
		return buildStrategy, err
	}
	return buildStrategy, nil
}

func toV1Alpha1BuildStrategyObject(convertedObject unstructured.Unstructured) (buildapialpha.BuildStrategy, error) {
	var buildStrategy buildapialpha.BuildStrategy
	u := convertedObject.UnstructuredContent()
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u, &buildStrategy); err != nil {
		return buildStrategy, err
	}
	return buildStrategy, nil
}

func toV1Beta1BuildObject(convertedObject unstructured.Unstructured) (buildapi.Build, error) {
	var build buildapi.Build
	u := convertedObject.UnstructuredContent()
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u, &build); err != nil {
		return build, err
	}
	return build, nil
}

/**
* TODO's:
* - in the Build resource, replace the build.shipwright.io/build-run-deletion annotation in favor of .spec.retention.atBuildDeletion.
**/
