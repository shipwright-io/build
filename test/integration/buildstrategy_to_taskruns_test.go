// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	utils "github.com/shipwright-io/build/test/utils/v1beta1"
	test "github.com/shipwright-io/build/test/v1beta1_samples"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Integration tests BuildStrategies and TaskRuns", func() {
	var (
		bsObject       *v1beta1.BuildStrategy
		buildObject    *v1beta1.Build
		buildRunObject *v1beta1.BuildRun
		secret         *corev1.Secret
		configMap      *corev1.ConfigMap
		buildSample    []byte
		buildRunSample []byte
	)

	// Load the BuildStrategies before each test case
	BeforeEach(func() {
		bsObject, err = tb.Catalog.LoadBuildStrategyFromBytes([]byte(test.BuildahBuildStrategySingleStep))
		Expect(err).To(BeNil())

		err = tb.CreateBuildStrategy(bsObject)
		Expect(err).To(BeNil())
	})

	// Delete the Build and BuildStrategy after each test case
	AfterEach(func() {
		_, err = tb.GetBuild(buildObject.Name)
		if err == nil {
			Expect(tb.DeleteBuild(buildObject.Name)).To(BeNil())
		}

		err := tb.DeleteBuildStrategy(bsObject.Name)
		Expect(err).To(BeNil())

		if configMap != nil {
			Expect(tb.DeleteConfigMap(configMap.Name)).NotTo(HaveOccurred())
			configMap = nil
		}

		if secret != nil {
			Expect(tb.DeleteSecret(secret.Name)).NotTo(HaveOccurred())
			secret = nil
		}
	})

	// Override the Build and BuildRun CRD instances to use
	// before an It() statement is executed
	JustBeforeEach(func() {
		if buildSample != nil {
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(BUILD+tb.Namespace, bsObject.Name, buildSample)
			Expect(err).To(BeNil())
		}

		if buildRunSample != nil {
			buildRunObject, err = tb.Catalog.LoadBRWithNameAndRef(BUILDRUN+tb.Namespace, BUILD+tb.Namespace, buildRunSample)
			Expect(err).To(BeNil())
		}
	})

	Context("when a buildrun is created", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildBSMinimal)
			buildRunSample = []byte(test.MinimalBuildRun)
		})

		It("should create a taskrun with the correct annotations", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			_, err = tb.GetBRTillStartTime(buildRunObject.Name)
			Expect(err).To(BeNil())

			taskRun, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())

			Expect(taskRun.Annotations["kubernetes.io/egress-bandwidth"]).To(Equal("1M"))
			Expect(taskRun.Annotations["kubernetes.io/ingress-bandwidth"]).To(Equal("1M"))
			_, containsKey := taskRun.Annotations["clusterbuildstrategy.shipwright.io/dummy"]
			Expect(containsKey).To(BeFalse())
			_, containsKey = taskRun.Annotations["kubectl.kubernetes.io/last-applied-configuration"]
			Expect(containsKey).To(BeFalse())
		})
	})

	Context("buildstrategy with defined parameters", func() {

		BeforeEach(func() {
			// Create a Strategy with parameters
			bsObject, err = tb.Catalog.LoadBuildStrategyFromBytes(
				[]byte(test.BuildStrategyWithParameters),
			)
			Expect(err).To(BeNil())

			err = tb.CreateBuildStrategy(bsObject)
			Expect(err).To(BeNil())

			// Create a minimal BuildRun
			buildRunObject, err = tb.Catalog.LoadBRWithNameAndRef(
				BUILDRUN+tb.Namespace,
				BUILD+tb.Namespace,
				[]byte(test.MinimalBuildRun),
			)
			Expect(err).To(BeNil())
		})

		var constructStringParam = func(paramName string, val string) pipelineapi.Param {
			return pipelineapi.Param{
				Name: paramName,
				Value: pipelineapi.ParamValue{
					Type:      pipelineapi.ParamTypeString,
					StringVal: val,
				},
			}
		}

		var constructArrayParam = func(paramName string, values ...string) pipelineapi.Param {
			return pipelineapi.Param{
				Name: paramName,
				Value: pipelineapi.ParamValue{
					Type:     pipelineapi.ParamTypeArray,
					ArrayVal: values,
				},
			}
		}

		var constructBuildObjectAndWait = func(b *v1beta1.Build) {
			// Create the Build object in-cluster
			Expect(tb.CreateBuild(b)).To(BeNil())

			// Wait until the Build object is validated
			_, err = tb.GetBuildTillValidation(b.Name)
			Expect(err).To(BeNil())
		}

		var constructBuildRunObjectAndWait = func(br *v1beta1.BuildRun) {
			// Create the BuildRun object in-cluster
			Expect(tb.CreateBR(br)).To(BeNil())

			// Wait until the BuildRun is registered
			_, err = tb.GetBRTillStartTime(br.Name)
			Expect(err).To(BeNil())
		}

		It("uses sleep-time param if specified in the Build with buildstrategy", func() {
			// Set BuildWithSleepTimeParam with a value of 30
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				BUILD+tb.Namespace,
				bsObject.Name,
				[]byte(test.BuildWithSleepTimeParam),
			)
			Expect(err).To(BeNil())

			constructBuildObjectAndWait(buildObject)

			constructBuildRunObjectAndWait(buildRunObject)

			taskRun, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())

			Expect(taskRun.Spec.Params).To(ContainElement(constructStringParam("sleep-time", "30")))
		})

		It("overrides sleep-time param if specified in the BuildRun", func() {
			// Set BuildWithSleepTimeParam with a value of 30
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				BUILD+tb.Namespace,
				bsObject.Name,
				[]byte(test.BuildWithSleepTimeParam),
			)
			Expect(err).To(BeNil())

			constructBuildObjectAndWait(buildObject)

			// Construct a BuildRun object that references the previous Build
			// without parameters definitions
			buildRunObject, err = tb.Catalog.LoadBRWithNameAndRef(
				BUILDRUN+tb.Namespace,
				BUILD+tb.Namespace,
				[]byte(test.MinimalBuildRunWithParams),
			)
			Expect(err).To(BeNil())

			constructBuildRunObjectAndWait(buildRunObject)

			taskRun, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())

			Expect(taskRun.Spec.Params).To(ContainElement(constructStringParam("sleep-time", "15")))
		})

		It("uses array-param if specified in the Build with buildstrategy", func() {
			// Set BuildWithArrayParam with an array value of "3" and "-1"
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				BUILD+tb.Namespace,
				bsObject.Name,
				[]byte(test.BuildWithArrayParam),
			)
			Expect(err).To(BeNil())

			constructBuildObjectAndWait(buildObject)

			constructBuildRunObjectAndWait(buildRunObject)

			taskRun, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())

			Expect(taskRun.Spec.Params).To(ContainElement(constructArrayParam("array-param", "3", "-1")))
		})

		It("uses params with references to ConfigMaps and Secrets correctly", func() {
			// prepare a ConfigMap
			configMap = tb.Catalog.ConfigMapWithData("a-configmap", tb.Namespace, map[string]string{
				"a-cm-key": "configmap-data",
			})
			err = tb.CreateConfigMap(configMap)
			Expect(err).ToNot(HaveOccurred())

			// prepare a secret
			secret = tb.Catalog.SecretWithStringData("a-secret", tb.Namespace, map[string]string{
				"a-secret-key": "a-value",
			})
			err = tb.CreateSecret(secret)
			Expect(err).ToNot(HaveOccurred())

			// create the build
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				BUILD+tb.Namespace,
				bsObject.Name,
				[]byte(test.BuildWithConfigMapSecretParams),
			)
			Expect(err).ToNot(HaveOccurred())

			constructBuildObjectAndWait(buildObject)

			constructBuildRunObjectAndWait(buildRunObject)

			taskRun, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).ToNot(HaveOccurred())

			// the sleep30 step should have an env var for the secret
			Expect(len(taskRun.Spec.TaskSpec.Steps)).To(Equal(3))
			Expect(len(taskRun.Spec.TaskSpec.Steps[1].Env)).To(Equal(1))
			Expect(taskRun.Spec.TaskSpec.Steps[1].Env[0].ValueFrom).NotTo(BeNil())
			Expect(taskRun.Spec.TaskSpec.Steps[1].Env[0].ValueFrom.SecretKeyRef).NotTo(BeNil())
			Expect(taskRun.Spec.TaskSpec.Steps[1].Env[0].ValueFrom.SecretKeyRef.Name).To(Equal("a-secret"))
			Expect(taskRun.Spec.TaskSpec.Steps[1].Env[0].ValueFrom.SecretKeyRef.Key).To(Equal("a-secret-key"))
			envVarNameSecret := taskRun.Spec.TaskSpec.Steps[1].Env[0].Name
			Expect(envVarNameSecret).To(HavePrefix("SHP_SECRET_PARAM_"))

			// the echo-array-sum step should have an env var for the ConfigMap
			Expect(len(taskRun.Spec.TaskSpec.Steps[2].Env)).To(Equal(1))
			Expect(taskRun.Spec.TaskSpec.Steps[2].Env[0].ValueFrom).NotTo(BeNil())
			Expect(taskRun.Spec.TaskSpec.Steps[2].Env[0].ValueFrom.ConfigMapKeyRef).NotTo(BeNil())
			Expect(taskRun.Spec.TaskSpec.Steps[2].Env[0].ValueFrom.ConfigMapKeyRef.Name).To(Equal("a-configmap"))
			Expect(taskRun.Spec.TaskSpec.Steps[2].Env[0].ValueFrom.ConfigMapKeyRef.Key).To(Equal("a-cm-key"))
			envVarNameConfigMap := taskRun.Spec.TaskSpec.Steps[2].Env[0].Name
			Expect(envVarNameConfigMap).To(HavePrefix("SHP_CONFIGMAP_PARAM_"))

			// verify the parameters
			Expect(taskRun.Spec.Params).To(ContainElement(constructStringParam("sleep-time", fmt.Sprintf("$(%s)", envVarNameSecret))))
			Expect(taskRun.Spec.Params).To(ContainElement(constructArrayParam("array-param", "3", fmt.Sprintf("$(%s)", envVarNameConfigMap), "-1")))
		})

		It("fails the TaskRun generation if the buildRun specifies a reserved system parameter", func() {
			// Build without params
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				BUILD+tb.Namespace,
				bsObject.Name,
				[]byte(test.BuildBSMinimal),
			)
			Expect(err).To(BeNil())

			constructBuildObjectAndWait(buildObject)

			// Construct a BuildRun object that references the previous Build
			// without usage of reserved params
			buildRunObjectWithReservedParams, err := tb.Catalog.LoadBRWithNameAndRef(
				BUILDRUN+tb.Namespace,
				BUILD+tb.Namespace,
				[]byte(test.MinimalBuildRunWithReservedParams),
			)
			Expect(err).To(BeNil())

			// Create the BuildRun object in-cluster
			Expect(tb.CreateBR(buildRunObjectWithReservedParams)).To(BeNil())

			// Wait until the BuildRun is registered
			br, err := tb.GetBRTillCompletion(buildRunObjectWithReservedParams.Name)
			Expect(err).To(BeNil())

			Expect(br.Status.GetCondition(v1beta1.Succeeded).GetReason()).To(Equal("RestrictedParametersInUse"))
			Expect(br.Status.GetCondition(v1beta1.Succeeded).GetMessage()).To(HavePrefix("The following parameters are restricted and cannot be set"))
			Expect(br.Status.GetCondition(v1beta1.Succeeded).GetMessage()).To(ContainSubstring("shp-sleep-time"))
		})

		It("add params from buildRun if they are not defined in the Build", func() {
			// Build without params
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				BUILD+tb.Namespace,
				bsObject.Name,
				[]byte(test.BuildBSMinimal),
			)
			Expect(err).To(BeNil())

			constructBuildObjectAndWait(buildObject)

			// Construct a BuildRun object that references the previous Build
			// without parameters definitions
			buildRunObject, err := tb.Catalog.LoadBRWithNameAndRef(
				BUILDRUN+tb.Namespace,
				BUILD+tb.Namespace,
				[]byte(test.MinimalBuildRunWithParams),
			)
			Expect(err).To(BeNil())

			constructBuildRunObjectAndWait(buildRunObject)

			_, err = tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())
		})

		It("fails the Build due to the usage of a restricted parameter name", func() {
			// Build using shipwright restricted params
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				BUILD+tb.Namespace,
				bsObject.Name,
				[]byte(test.BuildWithRestrictedParam),
			)
			Expect(err).To(BeNil())

			// Create the Build object in-cluster
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// Wait until the Build object is validated
			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(*buildObject.Status.Reason).To(Equal(v1beta1.RestrictedParametersInUse))
			Expect(*buildObject.Status.Message).To(HavePrefix("The following parameters are restricted and cannot be set:"))
			Expect(*buildObject.Status.Message).To(ContainSubstring("shp-something"))
		})

		It("fails the Build due to the definition of an undefined param in the strategy", func() {
			// Build using undefined parameter in the referenced strategy
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				BUILD+tb.Namespace,
				bsObject.Name,
				[]byte(test.BuildWithUndefinedParam),
			)
			Expect(err).To(BeNil())

			// Create the Build object in-cluster
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// Wait until the Build object is validated
			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(*buildObject.Status.Reason).To(Equal(v1beta1.UndefinedParameter))
			Expect(*buildObject.Status.Message).To(Equal("The following parameters are not defined in the build strategy: sleep-not"))
		})

		It("allows a user to set an empty string on parameter without default", func() {

			// Create a BuildStrategy with a parameter without default value
			bsObjectOverride, err := tb.Catalog.LoadBuildStrategyFromBytes(
				[]byte(test.BuildStrategyWithoutDefaultInParameter),
			)
			Expect(err).To(BeNil())

			err = tb.CreateBuildStrategy(bsObjectOverride)
			Expect(err).To(BeNil())

			// Build that uses an empty string on the single param
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				BUILD+tb.Namespace,
				bsObjectOverride.Name,
				[]byte(test.BuildWithEmptyStringParam),
			)
			Expect(err).To(BeNil())

			// Create the Build object in-cluster
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// Wait until the Build object is validated
			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			// Construct a BuildRun object that references the previous Build
			// without parameters definitions
			buildRunObject, err := tb.Catalog.LoadBRWithNameAndRef(
				BUILDRUN+tb.Namespace,
				BUILD+tb.Namespace,
				[]byte(test.MinimalBuildRun),
			)
			Expect(err).To(BeNil())

			constructBuildRunObjectAndWait(buildRunObject)

			_, err = tb.GetBRTillStartTime(buildRunObject.Name)
			Expect(err).To(BeNil())

			taskRun, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())

			// Validate that the TaskSpec parameter have no default value
			Expect(taskRun.Spec.TaskSpec.Params).To(ContainElement(pipelineapi.ParamSpec{
				Name:        "sleep-time",
				Type:        pipelineapi.ParamTypeString,
				Description: "time in seconds for sleeping",
				Default:     nil,
			}))

			// Validate that the TaskRun param have an empty string as the value
			Expect(taskRun.Spec.Params).To(ContainElement(pipelineapi.Param{
				Name: "sleep-time",
				Value: pipelineapi.ParamValue{
					Type:      pipelineapi.ParamTypeString,
					StringVal: "",
				},
			}))
		})

		It("fails the taskrun when a strategy parameter value is never specified", func() {

			// Create a BuildStrategy with a parameter without default value
			bsObjectOverride, err := tb.Catalog.LoadBuildStrategyFromBytes(
				[]byte(test.BuildStrategyWithoutDefaultInParameter),
			)
			Expect(err).To(BeNil())

			err = tb.CreateBuildStrategy(bsObjectOverride)
			Expect(err).To(BeNil())

			// Build that does not define a param
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
				BUILD+tb.Namespace,
				bsObjectOverride.Name,
				[]byte(test.BuildBSMinimal),
			)
			Expect(err).To(BeNil())

			// Create the Build object in-cluster
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			// Wait until the Build object is validated
			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			// Construct a BuildRun object that references the previous Build
			// without parameters definitions
			buildRunObject, err := tb.Catalog.LoadBRWithNameAndRef(
				BUILDRUN+tb.Namespace,
				BUILD+tb.Namespace,
				[]byte(test.MinimalBuildRun),
			)
			Expect(err).To(BeNil())

			// Create the BuildRun object in-cluster
			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			br, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())

			Expect(br.Status.GetCondition(v1beta1.Succeeded).GetReason()).To(Equal("MissingParameterValues"))
			Expect(br.Status.GetCondition(v1beta1.Succeeded).GetMessage()).To(HavePrefix("The following parameters are required but no value has been provided:"))
			Expect(br.Status.GetCondition(v1beta1.Succeeded).GetMessage()).To(ContainSubstring("sleep-time"))
		})

		Context("when a taskrun fails with an error result", func() {
			It("surfaces the result to the buildrun", func() {
				// Create a BuildStrategy that guarantees a failure
				cbsObject, err := tb.Catalog.LoadBuildStrategyFromBytes(
					[]byte(test.BuildStrategyWithErrorResult),
				)
				Expect(err).To(BeNil())

				err = tb.CreateBuildStrategy(cbsObject)
				Expect(err).To(BeNil())

				buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(
					BUILD+tb.Namespace,
					cbsObject.Name,
					[]byte(test.BuildBSMinimal),
				)
				Expect(err).To(BeNil())

				Expect(tb.CreateBuild(buildObject)).To(BeNil())

				buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
				Expect(err).To(BeNil())

				buildRunObject, err = tb.Catalog.LoadBRWithNameAndRef(
					BUILDRUN+tb.Namespace,
					BUILD+tb.Namespace,
					[]byte(test.MinimalBuildRun),
				)
				Expect(err).To(BeNil())

				Expect(tb.CreateBR(buildRunObject)).To(BeNil())

				buildRun, err := tb.GetBRTillCompletion(buildRunObject.Name)
				Expect(err).To(BeNil())

				Expect(buildRun.Status.FailureDetails).ToNot(BeNil())
				Expect(buildRun.Status.FailureDetails.Location.Container).To(Equal("step-fail-with-error-result"))
				Expect(buildRun.Status.FailureDetails.Message).To(Equal("integration test error message"))
				Expect(buildRun.Status.FailureDetails.Reason).To(Equal("integration test error reason"))
			})
		})
	})

	Context("buildstrategy without push operation", Label("managed-push"), func() {

		BeforeEach(func() {
			// Create a Strategy with parameters
			bsObject, err = tb.Catalog.LoadBuildStrategyFromBytes(
				[]byte(test.BuildStrategyWithoutPush),
			)
			Expect(err).ToNot(HaveOccurred())

			err = tb.CreateBuildStrategy(bsObject)
			Expect(err).ToNot(HaveOccurred())

			buildRunSample = []byte(test.MinimalBuildRun)
			buildSample = []byte(test.BuildBSMinimalNoSource)
		})

		Context("build output without secret", func() {

			It("creates a TaskRun with Shipwright managed push", func() {
				Expect(tb.CreateBuild(buildObject)).ToNot(HaveOccurred())

				buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
				Expect(err).ToNot(HaveOccurred())

				Expect(tb.CreateBR(buildRunObject)).ToNot(HaveOccurred())

				buildRunObject, err = tb.GetBRTillStartTime(buildRunObject.Name)
				Expect(err).ToNot(HaveOccurred())

				taskRun, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
				Expect(err).ToNot(HaveOccurred())

				// Verify the steps
				Expect(taskRun.Spec.TaskSpec.Steps).To(HaveLen(2))
				Expect(taskRun.Spec.TaskSpec.Steps[0].Name).To(Equal("store-tarball"))
				Expect(taskRun.Spec.TaskSpec.Steps[1].Name).To(Equal("image-processing"))

				// Verify the volume for the output directory
				Expect(taskRun.Spec.TaskSpec.Volumes).To(utils.ContainNamedElement("shp-output-directory"))
				Expect(taskRun.Spec.TaskSpec.Steps[0].VolumeMounts).To(utils.ContainNamedElement("shp-output-directory"))
				Expect(taskRun.Spec.TaskSpec.Steps[1].VolumeMounts).To(utils.ContainNamedElement("shp-output-directory"))

				// Verify the param
				Expect(taskRun.Spec.TaskSpec.Params).To(utils.ContainNamedElement("shp-output-directory"))
				Expect(taskRun.Spec.Params).To(utils.ContainNamedElement("shp-output-directory"))

				// Verify the step args
				Expect(taskRun.Spec.TaskSpec.Steps[1].Args).To(BeEquivalentTo([]string{
					"--push",
					"$(params.shp-output-directory)",
					"--image",
					"$(params.shp-output-image)",
					"--insecure=$(params.shp-output-insecure)",
					"--result-file-image-digest",
					"$(results.shp-image-digest.path)",
					"--result-file-image-size",
					"$(results.shp-image-size.path)",
					"--result-file-image-vulnerabilities",
					"$(results.shp-image-vulnerabilities.path)",
				}))
			})
		})

		Context("build output with secret", func() {

			BeforeEach(func() {
				secret = tb.Catalog.SecretWithDockerConfigJson("registry", tb.Namespace, "registry.abc", "some-user", "some-password")
				err := tb.CreateSecret(secret)
				Expect(err).ToNot(HaveOccurred())
			})

			It("creates a TaskRun with Shipwright managed push", func() {
				buildObject.Spec.Output.PushSecret = &secret.Name
				Expect(tb.CreateBuild(buildObject)).ToNot(HaveOccurred())

				buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
				Expect(err).ToNot(HaveOccurred())

				Expect(tb.CreateBR(buildRunObject)).ToNot(HaveOccurred())

				buildRunObject, err = tb.GetBRTillStartTime(buildRunObject.Name)
				Expect(err).ToNot(HaveOccurred())

				taskRun, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
				Expect(err).ToNot(HaveOccurred())

				// Verify the steps
				Expect(taskRun.Spec.TaskSpec.Steps).To(HaveLen(2))
				Expect(taskRun.Spec.TaskSpec.Steps[0].Name).To(Equal("store-tarball"))
				Expect(taskRun.Spec.TaskSpec.Steps[1].Name).To(Equal("image-processing"))

				// Verify the volume for the output directory
				Expect(taskRun.Spec.TaskSpec.Volumes).To(utils.ContainNamedElement("shp-output-directory"))
				Expect(taskRun.Spec.TaskSpec.Steps[0].VolumeMounts).To(utils.ContainNamedElement("shp-output-directory"))
				Expect(taskRun.Spec.TaskSpec.Steps[1].VolumeMounts).To(utils.ContainNamedElement("shp-output-directory"))

				// Verify the volume for the output secret
				Expect(taskRun.Spec.TaskSpec.Volumes).To(utils.ContainNamedElement(fmt.Sprintf("shp-%s", secret.Name)))
				Expect(taskRun.Spec.TaskSpec.Steps[1].VolumeMounts).To(utils.ContainNamedElement(fmt.Sprintf("shp-%s", secret.Name)))

				// Verify the param
				Expect(taskRun.Spec.TaskSpec.Params).To(utils.ContainNamedElement("shp-output-directory"))
				Expect(taskRun.Spec.Params).To(utils.ContainNamedElement("shp-output-directory"))

				// Verify the step args
				Expect(taskRun.Spec.TaskSpec.Steps[1].Args).To(BeEquivalentTo([]string{
					"--push",
					"$(params.shp-output-directory)",
					"--image",
					"$(params.shp-output-image)",
					"--insecure=$(params.shp-output-insecure)",
					"--result-file-image-digest",
					"$(results.shp-image-digest.path)",
					"--result-file-image-size",
					"$(results.shp-image-size.path)",
					"--result-file-image-vulnerabilities",
					"$(results.shp-image-vulnerabilities.path)",
					"--secret-path",
					"/workspace/shp-push-secret",
				}))
			})
		})
	})

	Context("when a build with stepResources is created", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildWithStepResourcesBuildStrategy)
			buildRunSample = []byte(test.MinimalBuildRun)

			// Delete the single-step BuildStrategy created in the global BeforeEach
			// and create a multi-step strategy that has buildah-bud and buildah-push steps
			Expect(tb.DeleteBuildStrategy(bsObject.Name)).To(BeNil())
			bsObject, err = tb.Catalog.LoadBuildStrategyFromBytes([]byte(test.MinimalBuildahBuildStrategy))
			Expect(err).To(BeNil())
			err = tb.CreateBuildStrategy(bsObject)
			Expect(err).To(BeNil())
		})

		It("should create a taskrun with the correct stepResources", func() {
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			_, err = tb.GetBRTillStartTime(buildRunObject.Name)
			Expect(err).To(BeNil())

			taskRun, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())

			// Find the buildah-bud step and verify its resources
			var budStep *pipelineapi.Step
			for i := range taskRun.Spec.TaskSpec.Steps {
				if taskRun.Spec.TaskSpec.Steps[i].Name == "buildah-bud" {
					budStep = &taskRun.Spec.TaskSpec.Steps[i]
					break
				}
			}

			Expect(budStep).ToNot(BeNil())
			// Verify that the step resources were overridden
			Expect(budStep.ComputeResources.Requests.Cpu().String()).To(Equal("250m"))
			Expect(budStep.ComputeResources.Requests.Memory().String()).To(Equal("256Mi"))
			Expect(budStep.ComputeResources.Limits.Cpu().String()).To(Equal("1"))
			Expect(budStep.ComputeResources.Limits.Memory().String()).To(Equal("2Gi"))

			// Find the buildah-push step and verify it retains strategy defaults
			var pushStep *pipelineapi.Step
			for i := range taskRun.Spec.TaskSpec.Steps {
				if taskRun.Spec.TaskSpec.Steps[i].Name == "buildah-push" {
					pushStep = &taskRun.Spec.TaskSpec.Steps[i]
					break
				}
			}

			Expect(pushStep).ToNot(BeNil())
			// Verify that the push step retains strategy defaults (100m, 65Mi)
			Expect(pushStep.ComputeResources.Requests.Cpu().String()).To(Equal("100m"))
			Expect(pushStep.ComputeResources.Requests.Memory().String()).To(Equal("65Mi"))
		})

		It("should fail validation when a stepResources override references an unknown step", func() {
			buildSample = []byte(test.BuildWithStepResourcesBuildStrategy)
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(BUILD+tb.Namespace, bsObject.Name, buildSample)
			Expect(err).To(BeNil())

			// Mutate the valid Build to reference a non-existent step
			buildObject.Spec.Strategy.StepResources[0].Name = "non-existent-step"

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(buildObject.Status.Reason).ToNot(BeNil())
			Expect(*buildObject.Status.Reason).To(Equal(v1beta1.UndefinedStepResource))
			Expect(*buildObject.Status.Message).To(ContainSubstring("non-existent-step"))
		})
	})

	Context("when a buildrun with stepResources override is created", func() {

		BeforeEach(func() {
			// Use a Build with stepResources, then override with BuildRun.Spec.StepResources
			buildSample = []byte(test.BuildWithStepResourcesBuildStrategy)
			buildRunSample = []byte(test.MinimalBuildRunWithStepResourcesOverride)

			// Delete the single-step BuildStrategy created in the global BeforeEach
			// and create a multi-step strategy that has buildah-bud and buildah-push steps
			Expect(tb.DeleteBuildStrategy(bsObject.Name)).To(BeNil())
			bsObject, err = tb.Catalog.LoadBuildStrategyFromBytes([]byte(test.MinimalBuildahBuildStrategy))
			Expect(err).To(BeNil())
			err = tb.CreateBuildStrategy(bsObject)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			buildRunSample = nil
		})

		It("should create a taskrun with BuildRun stepResources taking precedence over Build stepResources", func() {
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			_, err = tb.GetBRTillStartTime(buildRunObject.Name)
			Expect(err).To(BeNil())

			taskRun, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())

			// Find the buildah-bud step and verify its resources
			var budStep *pipelineapi.Step
			for i := range taskRun.Spec.TaskSpec.Steps {
				if taskRun.Spec.TaskSpec.Steps[i].Name == "buildah-bud" {
					budStep = &taskRun.Spec.TaskSpec.Steps[i]
					break
				}
			}

			Expect(budStep).ToNot(BeNil())
			// Verify that BuildRun.Spec.StepResources took precedence over Build.Spec.Strategy.StepResources
			// Build has: 250m/256Mi requests, 1/2Gi limits
			// BuildRun has: 500m/512Mi requests, 2/4Gi limits
			Expect(budStep.ComputeResources.Requests.Cpu().String()).To(Equal("500m"))
			Expect(budStep.ComputeResources.Requests.Memory().String()).To(Equal("512Mi"))
			Expect(budStep.ComputeResources.Limits.Cpu().String()).To(Equal("2"))
			Expect(budStep.ComputeResources.Limits.Memory().String()).To(Equal("4Gi"))

			// Find the buildah-push step and verify it retains strategy defaults (not overridden by either)
			var pushStep *pipelineapi.Step
			for i := range taskRun.Spec.TaskSpec.Steps {
				if taskRun.Spec.TaskSpec.Steps[i].Name == "buildah-push" {
					pushStep = &taskRun.Spec.TaskSpec.Steps[i]
					break
				}
			}

			Expect(pushStep).ToNot(BeNil())
			// Verify that the push step retains strategy defaults (100m, 65Mi)
			Expect(pushStep.ComputeResources.Requests.Cpu().String()).To(Equal("100m"))
			Expect(pushStep.ComputeResources.Requests.Memory().String()).To(Equal("65Mi"))
		})

		It("should fail validation when BuildRun stepResources reference an unknown step", func() {
			buildRunSample = []byte(test.MinimalBuildRunWithStepResourcesOverride)
			buildRunObject, err = tb.Catalog.LoadBRWithNameAndRef(BUILDRUN+tb.Namespace, buildObject.Name, buildRunSample)
			Expect(err).To(BeNil())

			// Mutate the valid BuildRun to reference a non-existent step
			buildRunObject.Spec.StepResources[0].Name = "non-existent-step"

			Expect(tb.CreateBuild(buildObject)).To(BeNil())
			_, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			br, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())

			condition := br.Status.GetCondition(v1beta1.Succeeded)
			Expect(condition.Status).To(Equal(corev1.ConditionFalse))
			Expect(condition.Reason).To(Equal(string(v1beta1.UndefinedStepResource)))
			Expect(condition.Message).To(ContainSubstring("does not exist"))
		})
	})
})
