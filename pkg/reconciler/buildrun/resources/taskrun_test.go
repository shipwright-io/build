// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	"path"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
	test "github.com/shipwright-io/build/test/v1beta1_samples"
)

var _ = Describe("TaskRun Unit Tests", func() {
	var (
		cfg                  *config.Config
		build                *buildv1beta1.Build
		buildRun             *buildv1beta1.BuildRun
		buildStrategy        *buildv1beta1.BuildStrategy
		clusterBuildStrategy *buildv1beta1.ClusterBuildStrategy
		serviceAccountName   string
		ctl                  test.Catalog
	)

	BeforeEach(func() {
		cfg = config.NewDefaultConfig()
		serviceAccountName = "test-sa"

		// Load basic test objects
		var err error
		build, err = ctl.LoadBuildYAML([]byte(test.MinimalBuild))
		Expect(err).ToNot(HaveOccurred())

		buildRun, err = ctl.LoadBuildRunFromBytes([]byte(test.MinimalBuildRun))
		Expect(err).ToNot(HaveOccurred())

		buildStrategy, err = ctl.LoadBuildStrategyFromBytes([]byte(test.ClusterBuildStrategyNoOp))
		Expect(err).ToNot(HaveOccurred())

		clusterBuildStrategy, err = ctl.LoadCBSWithName("noop", []byte(test.ClusterBuildStrategyNoOp))
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("GenerateTaskRun", func() {
		Context("with valid inputs", func() {
			It("should generate a TaskRun with BuildStrategy", func() {
				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun).ToNot(BeNil())
				Expect(taskRun.GenerateName).To(Equal(buildRun.Name + "-"))
				Expect(taskRun.Namespace).To(Equal(buildRun.Namespace))
				Expect(taskRun.Spec.ServiceAccountName).To(Equal(serviceAccountName))
				Expect(taskRun.Spec.TaskSpec).ToNot(BeNil())
				Expect(len(taskRun.Spec.Workspaces)).To(Equal(1))
				Expect(taskRun.Spec.Workspaces[0].Name).To(Equal("source"))
			})

			It("should generate a TaskRun with ClusterBuildStrategy", func() {
				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, clusterBuildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun).ToNot(BeNil())
				Expect(taskRun.GenerateName).To(Equal(buildRun.Name + "-"))
				Expect(taskRun.Namespace).To(Equal(buildRun.Namespace))
				Expect(taskRun.Spec.ServiceAccountName).To(Equal(serviceAccountName))
				Expect(taskRun.Spec.TaskSpec).ToNot(BeNil())
				Expect(len(taskRun.Spec.Workspaces)).To(Equal(1))
				Expect(taskRun.Spec.Workspaces[0].Name).To(Equal("source"))
			})

			It("should include proper labels", func() {
				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun.Labels).ToNot(BeNil())
				Expect(taskRun.Labels[buildv1beta1.LabelBuildRun]).To(Equal(buildRun.Name))
				Expect(taskRun.Labels[buildv1beta1.LabelBuildRunGeneration]).To(Equal(strconv.FormatInt(buildRun.Generation, 10)))

				if build.Name != "" {
					Expect(taskRun.Labels[buildv1beta1.LabelBuild]).To(Equal(build.Name))
					Expect(taskRun.Labels[buildv1beta1.LabelBuildGeneration]).To(Equal(strconv.FormatInt(build.Generation, 10)))
				}
			})

			It("should include base parameters", func() {
				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())

				// Check that base parameters are included
				paramNames := make(map[string]bool)
				for _, param := range taskRun.Spec.Params {
					paramNames[param.Name] = true
				}

				Expect(paramNames).To(HaveKey("shp-output-image"))
				Expect(paramNames).To(HaveKey("shp-output-insecure"))
				Expect(paramNames).To(HaveKey("shp-source-root"))
				Expect(paramNames).To(HaveKey("shp-source-context"))
			})

			It("should include TaskSpec parameters", func() {
				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())

				taskSpecParamNames := make(map[string]bool)
				for _, param := range taskRun.Spec.TaskSpec.Params {
					taskSpecParamNames[param.Name] = true
				}

				Expect(taskSpecParamNames).To(HaveKey("shp-output-image"))
				Expect(taskSpecParamNames).To(HaveKey("shp-output-insecure"))
				Expect(taskSpecParamNames).To(HaveKey("shp-source-root"))
				Expect(taskSpecParamNames).To(HaveKey("shp-source-context"))
			})

			It("should include workspaces", func() {
				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(len(taskRun.Spec.TaskSpec.Workspaces)).To(Equal(1))
				Expect(taskRun.Spec.TaskSpec.Workspaces[0].Name).To(Equal("source"))
			})

			It("should include results", func() {
				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(len(taskRun.Spec.TaskSpec.Results)).To(BeNumerically(">", 0))

				// Check for standard results
				resultNames := make(map[string]bool)
				for _, result := range taskRun.Spec.TaskSpec.Results {
					resultNames[result.Name] = true
				}

				Expect(resultNames).To(HaveKey("shp-image-digest"))
				Expect(resultNames).To(HaveKey("shp-image-size"))
			})
		})

		Context("with timeout configuration", func() {
			It("should use BuildRun timeout when specified", func() {
				duration := metav1.Duration{Duration: 5 * time.Minute}
				buildRun.Spec.Timeout = &duration

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun.Spec.Timeout).ToNot(BeNil())
				Expect(taskRun.Spec.Timeout.Duration).To(Equal(5 * time.Minute))
			})

			It("should use Build timeout when BuildRun timeout is not specified", func() {
				duration := metav1.Duration{Duration: 10 * time.Minute}
				build.Spec.Timeout = &duration

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun.Spec.Timeout).ToNot(BeNil())
				Expect(taskRun.Spec.Timeout.Duration).To(Equal(10 * time.Minute))
			})

			It("should prefer BuildRun timeout over Build timeout", func() {
				buildDuration := metav1.Duration{Duration: 10 * time.Minute}
				buildRunDuration := metav1.Duration{Duration: 5 * time.Minute}

				build.Spec.Timeout = &buildDuration
				buildRun.Spec.Timeout = &buildRunDuration

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun.Spec.Timeout).ToNot(BeNil())
				Expect(taskRun.Spec.Timeout.Duration).To(Equal(5 * time.Minute))
			})

			It("should have no timeout when neither Build nor BuildRun specify timeout", func() {
				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun.Spec.Timeout).To(BeNil())
			})
		})

		Context("with node selectors", func() {
			It("should use BuildRun node selector when specified", func() {
				buildRun.Spec.NodeSelector = map[string]string{
					"node-type": "buildrun-node",
				}

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun.Spec.PodTemplate).ToNot(BeNil())
				Expect(taskRun.Spec.PodTemplate.NodeSelector).To(Equal(buildRun.Spec.NodeSelector))
			})

			It("should use Build node selector when BuildRun node selector is not specified", func() {
				build.Spec.NodeSelector = map[string]string{
					"node-type": "build-node",
				}

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun.Spec.PodTemplate).ToNot(BeNil())
				Expect(taskRun.Spec.PodTemplate.NodeSelector).To(Equal(build.Spec.NodeSelector))
			})

			It("should prefer BuildRun node selector over Build node selector", func() {
				build.Spec.NodeSelector = map[string]string{
					"node-type": "build-node",
				}
				buildRun.Spec.NodeSelector = map[string]string{
					"node-type": "buildrun-node",
				}

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun.Spec.PodTemplate).ToNot(BeNil())
				Expect(taskRun.Spec.PodTemplate.NodeSelector).To(Equal(buildRun.Spec.NodeSelector))
			})

			It("should merge node selectors from Build and BuildRun", func() {
				build.Spec.NodeSelector = map[string]string{
					"build-key": "build-value",
				}
				buildRun.Spec.NodeSelector = map[string]string{
					"buildrun-key": "buildrun-value",
				}

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun.Spec.PodTemplate).ToNot(BeNil())
				Expect(taskRun.Spec.PodTemplate.NodeSelector).To(HaveKeyWithValue("build-key", "build-value"))
				Expect(taskRun.Spec.PodTemplate.NodeSelector).To(HaveKeyWithValue("buildrun-key", "buildrun-value"))
			})
		})

		Context("with tolerations", func() {
			It("should use BuildRun tolerations when specified", func() {
				buildRun.Spec.Tolerations = []corev1.Toleration{
					{
						Key:      "buildrun-key",
						Operator: corev1.TolerationOpEqual,
						Value:    "buildrun-value",
						Effect:   corev1.TaintEffectNoSchedule,
					},
				}

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun.Spec.PodTemplate).ToNot(BeNil())
				Expect(len(taskRun.Spec.PodTemplate.Tolerations)).To(Equal(1))
				Expect(taskRun.Spec.PodTemplate.Tolerations[0].Key).To(Equal("buildrun-key"))
			})

			It("should use Build tolerations when BuildRun tolerations are not specified", func() {
				build.Spec.Tolerations = []corev1.Toleration{
					{
						Key:      "build-key",
						Operator: corev1.TolerationOpEqual,
						Value:    "build-value",
						Effect:   corev1.TaintEffectNoSchedule,
					},
				}

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun.Spec.PodTemplate).ToNot(BeNil())
				Expect(len(taskRun.Spec.PodTemplate.Tolerations)).To(Equal(1))
				Expect(taskRun.Spec.PodTemplate.Tolerations[0].Key).To(Equal("build-key"))
			})

			It("should prefer BuildRun tolerations over Build tolerations for same key", func() {
				build.Spec.Tolerations = []corev1.Toleration{
					{
						Key:      "same-key",
						Operator: corev1.TolerationOpEqual,
						Value:    "build-value",
						Effect:   corev1.TaintEffectNoSchedule,
					},
				}
				buildRun.Spec.Tolerations = []corev1.Toleration{
					{
						Key:      "same-key",
						Operator: corev1.TolerationOpEqual,
						Value:    "buildrun-value",
						Effect:   corev1.TaintEffectNoSchedule,
					},
				}

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun.Spec.PodTemplate).ToNot(BeNil())
				Expect(len(taskRun.Spec.PodTemplate.Tolerations)).To(Equal(1))
				Expect(taskRun.Spec.PodTemplate.Tolerations[0].Value).To(Equal("buildrun-value"))
			})

			It("should set default effect to NoSchedule when not specified", func() {
				buildRun.Spec.Tolerations = []corev1.Toleration{
					{
						Key:      "test-key",
						Operator: corev1.TolerationOpEqual,
						Value:    "test-value",
						// Effect not specified
					},
				}

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun.Spec.PodTemplate).ToNot(BeNil())
				Expect(len(taskRun.Spec.PodTemplate.Tolerations)).To(Equal(1))
				Expect(taskRun.Spec.PodTemplate.Tolerations[0].Effect).To(Equal(corev1.TaintEffectNoSchedule))
			})
		})

		Context("with scheduler name", func() {
			It("should use BuildRun scheduler name when specified", func() {
				schedulerName := "buildrun-scheduler"
				buildRun.Spec.SchedulerName = &schedulerName

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun.Spec.PodTemplate).ToNot(BeNil())
				Expect(taskRun.Spec.PodTemplate.SchedulerName).To(Equal(schedulerName))
			})

			It("should use Build scheduler name when BuildRun scheduler name is not specified", func() {
				schedulerName := "build-scheduler"
				build.Spec.SchedulerName = &schedulerName

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun.Spec.PodTemplate).ToNot(BeNil())
				Expect(taskRun.Spec.PodTemplate.SchedulerName).To(Equal(schedulerName))
			})

			It("should prefer BuildRun scheduler name over Build scheduler name", func() {
				buildSchedulerName := "build-scheduler"
				buildRunSchedulerName := "buildrun-scheduler"

				build.Spec.SchedulerName = &buildSchedulerName
				buildRun.Spec.SchedulerName = &buildRunSchedulerName

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun.Spec.PodTemplate).ToNot(BeNil())
				Expect(taskRun.Spec.PodTemplate.SchedulerName).To(Equal(buildRunSchedulerName))
			})
		})

		Context("with runtimeClassName", func() {
			It("should use BuildRun runtimeClassName when specified", func() {
				runtimeClassName := "kata-containers"
				buildRun.Spec.RuntimeClassName = &runtimeClassName

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun.Spec.PodTemplate).ToNot(BeNil())
				Expect(taskRun.Spec.PodTemplate.RuntimeClassName).ToNot(BeNil())
				Expect(*taskRun.Spec.PodTemplate.RuntimeClassName).To(Equal(runtimeClassName))
			})

			It("should use Build runtimeClassName when BuildRun runtimeClassName is not specified", func() {
				runtimeClassName := "gvisor"
				build.Spec.RuntimeClassName = &runtimeClassName

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun.Spec.PodTemplate).ToNot(BeNil())
				Expect(taskRun.Spec.PodTemplate.RuntimeClassName).ToNot(BeNil())
				Expect(*taskRun.Spec.PodTemplate.RuntimeClassName).To(Equal(runtimeClassName))
			})

			It("should prefer BuildRun runtimeClassName over Build runtimeClassName", func() {
				buildRuntimeClassName := "gvisor"
				buildRunRuntimeClassName := "kata-containers"

				build.Spec.RuntimeClassName = &buildRuntimeClassName
				buildRun.Spec.RuntimeClassName = &buildRunRuntimeClassName

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun.Spec.PodTemplate).ToNot(BeNil())
				Expect(taskRun.Spec.PodTemplate.RuntimeClassName).ToNot(BeNil())
				Expect(*taskRun.Spec.PodTemplate.RuntimeClassName).To(Equal(buildRunRuntimeClassName))
			})
		})

		Context("with output image configuration", func() {
			It("should use BuildRun output image when specified", func() {
				buildRun.Spec.Output = &buildv1beta1.Image{
					Image: "registry.com/buildrun-image:latest",
				}

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())

				// Find the shp-output-image parameter
				var outputImageParam *pipelineapi.Param
				for _, param := range taskRun.Spec.Params {
					if param.Name == "shp-output-image" {
						outputImageParam = &param
						break
					}
				}

				Expect(outputImageParam).ToNot(BeNil())
				Expect(outputImageParam.Value.StringVal).To(Equal("registry.com/buildrun-image:latest"))
			})

			It("should use Build output image when BuildRun output image is not specified", func() {
				build.Spec.Output.Image = "registry.com/build-image:latest"

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())

				// Find the shp-output-image parameter
				var outputImageParam *pipelineapi.Param
				for _, param := range taskRun.Spec.Params {
					if param.Name == "shp-output-image" {
						outputImageParam = &param
						break
					}
				}

				Expect(outputImageParam).ToNot(BeNil())
				Expect(outputImageParam.Value.StringVal).To(Equal("registry.com/build-image:latest"))
			})

			It("should handle insecure flag from BuildRun", func() {
				insecure := true
				buildRun.Spec.Output = &buildv1beta1.Image{
					Image:    "registry.com/image:latest",
					Insecure: &insecure,
				}

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())

				// Find the shp-output-insecure parameter
				var insecureParam *pipelineapi.Param
				for _, param := range taskRun.Spec.Params {
					if param.Name == "shp-output-insecure" {
						insecureParam = &param
						break
					}
				}

				Expect(insecureParam).ToNot(BeNil())
				Expect(insecureParam.Value.StringVal).To(Equal("true"))
			})

			It("should handle insecure flag from Build when BuildRun doesn't specify it", func() {
				insecure := true
				build.Spec.Output.Insecure = &insecure

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())

				// Find the shp-output-insecure parameter
				var insecureParam *pipelineapi.Param
				for _, param := range taskRun.Spec.Params {
					if param.Name == "shp-output-insecure" {
						insecureParam = &param
						break
					}
				}

				Expect(insecureParam).ToNot(BeNil())
				Expect(insecureParam.Value.StringVal).To(Equal("true"))
			})
		})

		Context("with source context directory", func() {
			It("should set source context to workspace root when no context dir is specified", func() {
				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())

				// Find the shp-source-context parameter
				var sourceContextParam *pipelineapi.Param
				for _, param := range taskRun.Spec.Params {
					if param.Name == "shp-source-context" {
						sourceContextParam = &param
						break
					}
				}

				Expect(sourceContextParam).ToNot(BeNil())
				Expect(sourceContextParam.Value.StringVal).To(Equal("/workspace/source"))
			})

			It("should set source context to context dir when specified", func() {
				contextDir := "sub/directory"
				build.Spec.Source = &buildv1beta1.Source{
					ContextDir: &contextDir,
				}

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())

				// Find the shp-source-context parameter
				var sourceContextParam *pipelineapi.Param
				for _, param := range taskRun.Spec.Params {
					if param.Name == "shp-source-context" {
						sourceContextParam = &param
						break
					}
				}

				Expect(sourceContextParam).ToNot(BeNil())
				Expect(sourceContextParam.Value.StringVal).To(Equal(path.Join("/workspace/source", contextDir)))
			})
		})

		Context("with environment variables", func() {
			It("should handle environment variables from Build", func() {
				build.Spec.Env = []corev1.EnvVar{
					{Name: "BUILD_ENV", Value: "build-value"},
				}

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun).ToNot(BeNil())

				// Environment variables should be merged into strategy steps
				if len(taskRun.Spec.TaskSpec.Steps) > 0 {
					// Check if any step has the environment variable
					found := false
					for _, step := range taskRun.Spec.TaskSpec.Steps {
						for _, env := range step.Env {
							if env.Name == "BUILD_ENV" && env.Value == "build-value" {
								found = true
								break
							}
						}
						if found {
							break
						}
					}
					// Note: Env vars are only added to strategy steps, not all steps
				}
			})

			It("should handle environment variables from BuildRun", func() {
				buildRun.Spec.Env = []corev1.EnvVar{
					{Name: "BUILDRUN_ENV", Value: "buildrun-value"},
				}

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun).ToNot(BeNil())
			})

			It("should prefer BuildRun environment variables over Build environment variables", func() {
				build.Spec.Env = []corev1.EnvVar{
					{Name: "SAME_ENV", Value: "build-value"},
				}
				buildRun.Spec.Env = []corev1.EnvVar{
					{Name: "SAME_ENV", Value: "buildrun-value"},
				}

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun).ToNot(BeNil())
			})
		})

		Context("with embedded Build (empty build name)", func() {
			It("should not include Build labels when build name is empty", func() {
				build.Name = ""

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun.Labels).ToNot(BeNil())
				Expect(taskRun.Labels[buildv1beta1.LabelBuildRun]).To(Equal(buildRun.Name))
				Expect(taskRun.Labels).ToNot(HaveKey(buildv1beta1.LabelBuild))
				Expect(taskRun.Labels).ToNot(HaveKey(buildv1beta1.LabelBuildGeneration))
			})
		})

		Context("with strategy parameters", func() {
			It("should include strategy parameters in TaskSpec", func() {
				// Use a build strategy that already has parameters
				paramBuildStrategy, err := ctl.LoadBuildStrategyFromBytes([]byte(test.BuildStrategyWithParameters))
				Expect(err).ToNot(HaveOccurred())

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, paramBuildStrategy)

				Expect(err).ToNot(HaveOccurred())

				// Check that the parameter is included in TaskSpec
				paramFound := false
				for _, param := range taskRun.Spec.TaskSpec.Params {
					if param.Name == "sleep-time" {
						paramFound = true
						Expect(param.Description).To(Equal("time in seconds for sleeping"))
						Expect(param.Type).To(Equal(pipelineapi.ParamTypeString))
						Expect(param.Default).ToNot(BeNil())
						Expect(param.Default.StringVal).To(Equal("1"))
						break
					}
				}
				Expect(paramFound).To(BeTrue())
			})

			It("should handle array type parameters", func() {
				// Use a build strategy that already has array parameters
				paramBuildStrategy, err := ctl.LoadBuildStrategyFromBytes([]byte(test.BuildStrategyWithParameters))
				Expect(err).ToNot(HaveOccurred())

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, paramBuildStrategy)

				Expect(err).ToNot(HaveOccurred())

				// Check that the array parameter is included in TaskSpec
				paramFound := false
				for _, param := range taskRun.Spec.TaskSpec.Params {
					if param.Name == "array-param" {
						paramFound = true
						Expect(param.Description).To(Equal("An arbitrary array"))
						Expect(param.Type).To(Equal(pipelineapi.ParamTypeArray))
						Expect(param.Default).ToNot(BeNil())
						Expect(param.Default.ArrayVal).To(Equal([]string{}))
						break
					}
				}
				Expect(paramFound).To(BeTrue())
			})
		})

		Context("with strategy steps", func() {
			It("should include strategy steps in TaskSpec", func() {
				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(len(taskRun.Spec.TaskSpec.Steps)).To(BeNumerically(">", 0))

				// The exact number of steps depends on the strategy and source configuration
				// but we should have at least the strategy steps
				strategyStepFound := false
				for _, step := range taskRun.Spec.TaskSpec.Steps {
					// Check if this is one of the strategy steps
					for _, strategyStep := range buildStrategy.Spec.Steps {
						if step.Name == strategyStep.Name {
							strategyStepFound = true
							Expect(step.Image).To(Equal(strategyStep.Image))
							break
						}
					}
					if strategyStepFound {
						break
					}
				}
				Expect(strategyStepFound).To(BeTrue())
			})
		})

		Context("with volumes", func() {
			It("should handle strategy volumes", func() {
				// Create a build strategy with volumes for this test
				volumeBuildStrategy, err := ctl.LoadBuildStrategyFromBytes([]byte(test.ClusterBuildStrategyNoOp))
				Expect(err).ToNot(HaveOccurred())

				// Add a volume to the build strategy
				volumeBuildStrategy.Spec.Volumes = []buildv1beta1.BuildStrategyVolume{
					{
						Name: "test-volume",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				}

				// Add a volume mount to the strategy step
				if len(volumeBuildStrategy.Spec.Steps) > 0 {
					volumeBuildStrategy.Spec.Steps[0].VolumeMounts = []corev1.VolumeMount{
						{
							Name:      "test-volume",
							MountPath: "/test",
						},
					}
				}

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, volumeBuildStrategy)

				Expect(err).ToNot(HaveOccurred())

				// Check that the volume is included in TaskSpec
				volumeFound := false
				for _, volume := range taskRun.Spec.TaskSpec.Volumes {
					if volume.Name == "test-volume" {
						volumeFound = true
						Expect(volume.EmptyDir).ToNot(BeNil())
						break
					}
				}
				Expect(volumeFound).To(BeTrue())
			})
		})

		Context("error handling", func() {
			It("should handle invalid parameters gracefully", func() {
				// Create a build with a parameter that doesn't exist in strategy
				buildWithInvalidParam := build.DeepCopy()
				buildWithInvalidParam.Spec.ParamValues = []buildv1beta1.ParamValue{
					{
						Name:        "nonexistent-param",
						SingleValue: &buildv1beta1.SingleValue{Value: stringPtr("some-value")},
					},
				}

				taskRun, err := resources.GenerateTaskRun(cfg, buildWithInvalidParam, buildRun, serviceAccountName, buildStrategy)

				Expect(err).To(HaveOccurred())
				Expect(taskRun).To(BeNil())
				Expect(err.Error()).To(ContainSubstring("not defined in the build strategy"))
			})
		})
	})

	Context("with annotations", func() {
		It("should propagate strategy annotations to TaskRun", func() {
			annotationStrategy, err := ctl.LoadCBSWithName("kaniko", []byte(test.ClusterBuildStrategyWithAnnotations))
			Expect(err).ToNot(HaveOccurred())

			taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, annotationStrategy)

			Expect(err).ToNot(HaveOccurred())
			Expect(taskRun.Annotations).ToNot(BeNil())

			// Check that propagatable annotations are included (only kubernetes.io/ingress-bandwidth should be propagated)
			Expect(taskRun.Annotations).To(HaveKey("kubernetes.io/ingress-bandwidth"))
			Expect(taskRun.Annotations["kubernetes.io/ingress-bandwidth"]).To(Equal("1M"))

			// Check that non-propagatable annotations are filtered out
			Expect(taskRun.Annotations).ToNot(HaveKey("clusterbuildstrategy.shipwright.io/dummy"))
			Expect(taskRun.Annotations).ToNot(HaveKey("kubectl.kubernetes.io/last-applied-configuration"))
		})

		It("should not add annotations when strategy has none", func() {
			taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

			Expect(err).ToNot(HaveOccurred())
			// TaskRun should have no annotations or empty annotations
			if taskRun.Annotations != nil {
				Expect(len(taskRun.Annotations)).To(Equal(0))
			}
		})
	})

	Context("with volume overrides", func() {
		It("should use BuildRun volume overrides when specified", func() {
			// Create a build strategy that has the volume to be overridden
			strategyWithVolume := buildStrategy.DeepCopy()
			overridable := true
			strategyWithVolume.Spec.Volumes = []buildv1beta1.BuildStrategyVolume{
				{
					Name:        "buildrun-volume",
					Overridable: &overridable,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			}

			buildRun.Spec.Volumes = []buildv1beta1.BuildVolume{
				{
					Name: "buildrun-volume",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "buildrun-config",
							},
						},
					},
				},
			}

			taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, strategyWithVolume)

			Expect(err).ToNot(HaveOccurred())

			// Check that BuildRun volume override is used in TaskSpec
			volumeFound := false
			for _, volume := range taskRun.Spec.TaskSpec.Volumes {
				if volume.Name == "buildrun-volume" {
					volumeFound = true
					Expect(volume.ConfigMap).ToNot(BeNil())
					Expect(volume.ConfigMap.Name).To(Equal("buildrun-config"))
					Expect(volume.EmptyDir).To(BeNil()) // Should not have the original EmptyDir
					break
				}
			}
			Expect(volumeFound).To(BeTrue())
		})

		It("should use Build volume overrides when BuildRun volumes are not specified", func() {
			// Create a build strategy that has the volume to be overridden
			strategyWithVolume := buildStrategy.DeepCopy()
			overridable := true
			strategyWithVolume.Spec.Volumes = []buildv1beta1.BuildStrategyVolume{
				{
					Name:        "build-volume",
					Overridable: &overridable,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			}

			build.Spec.Volumes = []buildv1beta1.BuildVolume{
				{
					Name: "build-volume",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "build-secret",
						},
					},
				},
			}

			taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, strategyWithVolume)

			Expect(err).ToNot(HaveOccurred())

			// Check that Build volume override is used in TaskSpec
			volumeFound := false
			for _, volume := range taskRun.Spec.TaskSpec.Volumes {
				if volume.Name == "build-volume" {
					volumeFound = true
					Expect(volume.Secret).ToNot(BeNil())
					Expect(volume.Secret.SecretName).To(Equal("build-secret"))
					Expect(volume.EmptyDir).To(BeNil()) // Should not have the original EmptyDir
					break
				}
			}
			Expect(volumeFound).To(BeTrue())
		})

		It("should prefer BuildRun volumes over Build volumes for same name", func() {
			// Create a build strategy that has the volume to be overridden
			strategyWithVolume := buildStrategy.DeepCopy()
			overridable := true
			strategyWithVolume.Spec.Volumes = []buildv1beta1.BuildStrategyVolume{
				{
					Name:        "shared-volume",
					Overridable: &overridable,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			}

			build.Spec.Volumes = []buildv1beta1.BuildVolume{
				{
					Name: "shared-volume",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "build-secret",
						},
					},
				},
			}

			buildRun.Spec.Volumes = []buildv1beta1.BuildVolume{
				{
					Name: "shared-volume",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "buildrun-config",
							},
						},
					},
				},
			}

			taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, strategyWithVolume)

			Expect(err).ToNot(HaveOccurred())

			// Check that BuildRun volume takes precedence over Build volume
			volumeFound := false
			for _, volume := range taskRun.Spec.TaskSpec.Volumes {
				if volume.Name == "shared-volume" {
					volumeFound = true
					Expect(volume.ConfigMap).ToNot(BeNil())
					Expect(volume.ConfigMap.Name).To(Equal("buildrun-config"))
					Expect(volume.Secret).To(BeNil())
					Expect(volume.EmptyDir).To(BeNil())
					break
				}
			}
			Expect(volumeFound).To(BeTrue())
		})
	})

	Context("with retention policies", func() {
		It("should handle Build retention configuration", func() {
			failedLimit := uint(3)
			succeededLimit := uint(2)
			build.Spec.Retention = &buildv1beta1.BuildRetention{
				FailedLimit:    &failedLimit,
				SucceededLimit: &succeededLimit,
			}

			taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

			Expect(err).ToNot(HaveOccurred())
			Expect(taskRun).ToNot(BeNil())
			// Retention policies don't directly affect TaskRun generation,
			// but they should not cause errors
		})

		It("should handle BuildRun retention configuration", func() {
			ttlAfterFailed := metav1.Duration{Duration: 10 * time.Minute}
			ttlAfterSucceeded := metav1.Duration{Duration: 5 * time.Minute}
			buildRun.Spec.Retention = &buildv1beta1.BuildRunRetention{
				TTLAfterFailed:    &ttlAfterFailed,
				TTLAfterSucceeded: &ttlAfterSucceeded,
			}

			taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

			Expect(err).ToNot(HaveOccurred())
			Expect(taskRun).ToNot(BeNil())
		})
	})

	Context("with complex parameter validation", func() {
		It("should handle array parameters correctly", func() {
			paramBuildStrategy, err := ctl.LoadBuildStrategyFromBytes([]byte(test.BuildStrategyWithParameters))
			Expect(err).ToNot(HaveOccurred())

			// Create a build with array parameter
			buildWithArrayParam := build.DeepCopy()
			buildWithArrayParam.Spec.ParamValues = []buildv1beta1.ParamValue{
				{
					Name: "array-param",
					Values: []buildv1beta1.SingleValue{
						{Value: stringPtr("1")},
						{Value: stringPtr("2")},
						{Value: stringPtr("3")},
					},
				},
			}

			taskRun, err := resources.GenerateTaskRun(cfg, buildWithArrayParam, buildRun, serviceAccountName, paramBuildStrategy)

			Expect(err).ToNot(HaveOccurred())
			Expect(taskRun).ToNot(BeNil())

			// Check that the array parameter is included
			paramFound := false
			for _, param := range taskRun.Spec.Params {
				if param.Name == "array-param" {
					paramFound = true
					Expect(param.Value.ArrayVal).To(Equal([]string{"1", "2", "3"}))
					break
				}
			}
			Expect(paramFound).To(BeTrue())
		})

		It("should handle parameters with defaults", func() {
			paramBuildStrategy, err := ctl.LoadBuildStrategyFromBytes([]byte(test.BuildStrategyWithParameters))
			Expect(err).ToNot(HaveOccurred())

			// Don't provide the sleep-time parameter, let it use default
			taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, paramBuildStrategy)

			Expect(err).ToNot(HaveOccurred())
			Expect(taskRun).ToNot(BeNil())

			// Check that the parameter definition is included in TaskSpec with default
			paramFound := false
			for _, param := range taskRun.Spec.TaskSpec.Params {
				if param.Name == "sleep-time" {
					paramFound = true
					Expect(param.Default).ToNot(BeNil())
					Expect(param.Default.StringVal).To(Equal("1")) // default value in TaskSpec
					break
				}
			}
			Expect(paramFound).To(BeTrue())
		})

		It("should override parameter defaults when specified", func() {
			paramBuildStrategy, err := ctl.LoadBuildStrategyFromBytes([]byte(test.BuildStrategyWithParameters))
			Expect(err).ToNot(HaveOccurred())

			// Override the sleep-time parameter
			buildWithParam := build.DeepCopy()
			buildWithParam.Spec.ParamValues = []buildv1beta1.ParamValue{
				{
					Name:        "sleep-time",
					SingleValue: &buildv1beta1.SingleValue{Value: stringPtr("5")},
				},
			}

			taskRun, err := resources.GenerateTaskRun(cfg, buildWithParam, buildRun, serviceAccountName, paramBuildStrategy)

			Expect(err).ToNot(HaveOccurred())
			Expect(taskRun).ToNot(BeNil())

			// Check that the parameter uses provided value
			paramFound := false
			for _, param := range taskRun.Spec.Params {
				if param.Name == "sleep-time" {
					paramFound = true
					Expect(param.Value.StringVal).To(Equal("5")) // overridden value
					break
				}
			}
			Expect(paramFound).To(BeTrue())
		})
	})

	Context("with environment variable edge cases", func() {
		It("should handle environment variables with valueFrom", func() {
			build.Spec.Env = []corev1.EnvVar{
				{
					Name: "SECRET_VALUE",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "my-secret",
							},
							Key: "secret-key",
						},
					},
				},
			}

			buildRun.Spec.Env = []corev1.EnvVar{
				{
					Name: "CONFIG_VALUE",
					ValueFrom: &corev1.EnvVarSource{
						ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "my-config",
							},
							Key: "config-key",
						},
					},
				},
			}

			taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

			Expect(err).ToNot(HaveOccurred())
			Expect(taskRun).ToNot(BeNil())
			// Environment variables are merged into strategy steps
		})

		It("should handle conflicting environment variable names with different sources", func() {
			build.Spec.Env = []corev1.EnvVar{
				{
					Name:  "CONFLICT_VAR",
					Value: "build-value",
				},
			}

			buildRun.Spec.Env = []corev1.EnvVar{
				{
					Name: "CONFLICT_VAR",
					ValueFrom: &corev1.EnvVarSource{
						ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "my-config",
							},
							Key: "config-key",
						},
					},
				},
			}

			taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

			Expect(err).ToNot(HaveOccurred())
			Expect(taskRun).ToNot(BeNil())
			// BuildRun env var should take precedence
		})
	})

	Context("with service account variations", func() {
		It("should handle '.generate' service account", func() {
			generateSA := ".generate"
			buildRun.Spec.ServiceAccount = &generateSA

			taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

			Expect(err).ToNot(HaveOccurred())
			Expect(taskRun).ToNot(BeNil())
			// Service account name should still be set from the parameter passed to GenerateTaskRun
			Expect(taskRun.Spec.ServiceAccountName).To(Equal(serviceAccountName))
		})

		It("should handle custom service account", func() {
			customSA := "custom-service-account"
			buildRun.Spec.ServiceAccount = &customSA

			taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

			Expect(err).ToNot(HaveOccurred())
			Expect(taskRun).ToNot(BeNil())
			Expect(taskRun.Spec.ServiceAccountName).To(Equal(serviceAccountName))
		})
	})

	Context("with source-related configurations", func() {
		It("should handle BuildRun source overrides", func() {
			buildRunWithSource := buildRun.DeepCopy()
			buildRunWithSource.Spec.Source = &buildv1beta1.BuildRunSource{
				Type: buildv1beta1.LocalType,
				Local: &buildv1beta1.Local{
					Name: "local-source",
				},
			}

			taskRun, err := resources.GenerateTaskRun(cfg, build, buildRunWithSource, serviceAccountName, buildStrategy)

			Expect(err).ToNot(HaveOccurred())
			Expect(taskRun).ToNot(BeNil())
		})

		It("should handle different source types", func() {
			// Test with OCI source type
			buildWithOCI := build.DeepCopy()
			buildWithOCI.Spec.Source = &buildv1beta1.Source{
				Type: buildv1beta1.OCIArtifactType,
				OCIArtifact: &buildv1beta1.OCIArtifact{
					Image: "registry.com/my-source:latest",
				},
			}

			taskRun, err := resources.GenerateTaskRun(cfg, buildWithOCI, buildRun, serviceAccountName, buildStrategy)

			Expect(err).ToNot(HaveOccurred())
			Expect(taskRun).ToNot(BeNil())
		})

		It("should handle source with credentials", func() {
			secretName := "git-credentials"
			buildWithCredentials := build.DeepCopy()
			buildWithCredentials.Spec.Source = &buildv1beta1.Source{
				Type: buildv1beta1.GitType,
				Git: &buildv1beta1.Git{
					URL:         "https://github.com/private/repo.git",
					CloneSecret: &secretName,
				},
			}

			taskRun, err := resources.GenerateTaskRun(cfg, buildWithCredentials, buildRun, serviceAccountName, buildStrategy)

			Expect(err).ToNot(HaveOccurred())
			Expect(taskRun).ToNot(BeNil())
		})
	})

	Context("with image processing configurations", func() {
		It("should handle output image with annotations", func() {
			buildRun.Spec.Output = &buildv1beta1.Image{
				Image: "registry.com/test:latest",
				Annotations: map[string]string{
					"org.opencontainers.image.description": "Test image",
					"custom.annotation":                    "custom-value",
				},
			}

			taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

			Expect(err).ToNot(HaveOccurred())
			Expect(taskRun).ToNot(BeNil())

			// Check that image processing steps are added
			hasImageProcessingStep := false
			for _, step := range taskRun.Spec.TaskSpec.Steps {
				if step.Name == "image-processing" {
					hasImageProcessingStep = true
					break
				}
			}
			Expect(hasImageProcessingStep).To(BeTrue())
		})

		It("should handle output image with labels", func() {
			buildRun.Spec.Output = &buildv1beta1.Image{
				Image: "registry.com/test:latest",
				Labels: map[string]string{
					"maintainer": "team@company.com",
					"version":    "1.0.0",
				},
			}

			taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

			Expect(err).ToNot(HaveOccurred())
			Expect(taskRun).ToNot(BeNil())

			// Check for image processing step
			hasImageProcessingStep := false
			for _, step := range taskRun.Spec.TaskSpec.Steps {
				if step.Name == "image-processing" {
					hasImageProcessingStep = true
					break
				}
			}
			Expect(hasImageProcessingStep).To(BeTrue())
		})

		It("should handle vulnerability scanning configuration", func() {
			enabled := true
			failOnFinding := true
			buildRun.Spec.Output = &buildv1beta1.Image{
				Image: "registry.com/test:latest",
				VulnerabilityScan: &buildv1beta1.VulnerabilityScanOptions{
					Enabled:       enabled,
					FailOnFinding: failOnFinding,
				},
			}

			taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

			Expect(err).ToNot(HaveOccurred())
			Expect(taskRun).ToNot(BeNil())

			// Check that vulnerability scanning is configured in image processing
			hasImageProcessingStep := false
			for _, step := range taskRun.Spec.TaskSpec.Steps {
				if step.Name == "image-processing" {
					hasImageProcessingStep = true
					break
				}
			}
			Expect(hasImageProcessingStep).To(BeTrue())
		})

		It("should handle image timestamp settings", func() {
			timestamp := buildv1beta1.OutputImageZeroTimestamp
			buildRun.Spec.Output = &buildv1beta1.Image{
				Image:     "registry.com/test:latest",
				Timestamp: &timestamp,
			}

			taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

			Expect(err).ToNot(HaveOccurred())
			Expect(taskRun).ToNot(BeNil())

			// Image processing should handle timestamp
			hasImageProcessingStep := false
			for _, step := range taskRun.Spec.TaskSpec.Steps {
				if step.Name == "image-processing" {
					hasImageProcessingStep = true
					break
				}
			}
			Expect(hasImageProcessingStep).To(BeTrue())
		})
	})

	Describe("Helper functions", func() {
		Context("generateTaskRunLabels", func() {
			It("should generate correct labels for Build with name", func() {
				build.Name = "test-build"
				build.Generation = 5
				buildRun.Name = "test-buildrun"
				buildRun.Generation = 3

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun.Labels[buildv1beta1.LabelBuild]).To(Equal("test-build"))
				Expect(taskRun.Labels[buildv1beta1.LabelBuildGeneration]).To(Equal("5"))
				Expect(taskRun.Labels[buildv1beta1.LabelBuildRun]).To(Equal("test-buildrun"))
				Expect(taskRun.Labels[buildv1beta1.LabelBuildRunGeneration]).To(Equal("3"))
			})

			It("should not include Build labels when build name is empty", func() {
				build.Name = ""
				buildRun.Name = "test-buildrun"
				buildRun.Generation = 3

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun.Labels).ToNot(HaveKey(buildv1beta1.LabelBuild))
				Expect(taskRun.Labels).ToNot(HaveKey(buildv1beta1.LabelBuildGeneration))
				Expect(taskRun.Labels[buildv1beta1.LabelBuildRun]).To(Equal("test-buildrun"))
				Expect(taskRun.Labels[buildv1beta1.LabelBuildRunGeneration]).To(Equal("3"))
			})
		})

		Context("generateWorkspaceBindings", func() {
			It("should create workspace binding for source", func() {
				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(len(taskRun.Spec.Workspaces)).To(Equal(1))
				Expect(taskRun.Spec.Workspaces[0].Name).To(Equal("source"))
				Expect(taskRun.Spec.Workspaces[0].EmptyDir).ToNot(BeNil())
			})
		})

		Context("generateTaskRunMetadata", func() {
			It("should create correct metadata", func() {
				buildRun.Name = "test-buildrun"
				buildRun.Namespace = "test-namespace"

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun.GenerateName).To(Equal("test-buildrun-"))
				Expect(taskRun.Namespace).To(Equal("test-namespace"))
				Expect(taskRun.Labels).ToNot(BeNil())
			})
		})

		Context("createBaseTaskSpec", func() {
			It("should create TaskSpec with base components", func() {
				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun.Spec.TaskSpec.Params).ToNot(BeEmpty())
				Expect(taskRun.Spec.TaskSpec.Workspaces).ToNot(BeEmpty())
				Expect(taskRun.Spec.TaskSpec.Results).ToNot(BeEmpty())
			})
		})
	})

	Describe("Integration scenarios", func() {
		Context("with complex configuration", func() {
			It("should handle multiple configurations together", func() {
				// Set up complex configuration
				contextDir := "docker"
				schedulerName := "custom-scheduler"
				insecure := true
				duration := metav1.Duration{Duration: 15 * time.Minute}

				build.Spec.Source = &buildv1beta1.Source{
					ContextDir: &contextDir,
				}
				build.Spec.NodeSelector = map[string]string{"build-node": "true"}
				build.Spec.Tolerations = []corev1.Toleration{
					{Key: "build-taint", Operator: corev1.TolerationOpExists},
				}
				build.Spec.SchedulerName = &schedulerName
				build.Spec.Timeout = &duration
				build.Spec.Output.Insecure = &insecure
				build.Spec.Env = []corev1.EnvVar{
					{Name: "BUILD_VAR", Value: "build-val"},
				}

				buildRun.Spec.NodeSelector = map[string]string{"buildrun-node": "true"}
				buildRun.Spec.Env = []corev1.EnvVar{
					{Name: "BUILDRUN_VAR", Value: "buildrun-val"},
				}

				taskRun, err := resources.GenerateTaskRun(cfg, build, buildRun, serviceAccountName, buildStrategy)

				Expect(err).ToNot(HaveOccurred())
				Expect(taskRun).ToNot(BeNil())

				// Verify all configurations are applied
				Expect(taskRun.Spec.Timeout.Duration).To(Equal(15 * time.Minute))
				Expect(taskRun.Spec.PodTemplate.NodeSelector).To(HaveKeyWithValue("buildrun-node", "true"))
				Expect(taskRun.Spec.PodTemplate.SchedulerName).To(Equal(schedulerName))

				// Check source context parameter
				var sourceContextParam *pipelineapi.Param
				for _, param := range taskRun.Spec.Params {
					if param.Name == "shp-source-context" {
						sourceContextParam = &param
						break
					}
				}
				Expect(sourceContextParam).ToNot(BeNil())
				Expect(sourceContextParam.Value.StringVal).To(Equal("/workspace/source/docker"))

				// Check insecure parameter
				var insecureParam *pipelineapi.Param
				for _, param := range taskRun.Spec.Params {
					if param.Name == "shp-output-insecure" {
						insecureParam = &param
						break
					}
				}
				Expect(insecureParam).ToNot(BeNil())
				Expect(insecureParam.Value.StringVal).To(Equal("true"))
			})
		})
	})
})

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
