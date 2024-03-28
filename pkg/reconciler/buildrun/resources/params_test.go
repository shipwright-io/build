// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

var _ = Describe("Params overrides", func() {

	DescribeTable("original params can be overridden",
		func(buildParams []buildapi.ParamValue, buildRunParams []buildapi.ParamValue, expected types.GomegaMatcher) {
			Expect(OverrideParams(buildParams, buildRunParams)).To(expected)
		},

		Entry("override a single parameter",
			[]buildapi.ParamValue{
				{Name: "a", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("2"),
				}},
			}, []buildapi.ParamValue{
				{Name: "a", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("3"),
				}},
			}, ContainElements([]buildapi.ParamValue{
				{Name: "a", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("3"),
				}},
			})),

		Entry("override two parameters",
			[]buildapi.ParamValue{
				{Name: "a", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("2"),
				}},
				{Name: "b", SingleValue: &buildapi.SingleValue{
					SecretValue: &buildapi.ObjectKeyRef{
						Name: "a-secret",
						Key:  "a-key",
					},
				}},
			}, []buildapi.ParamValue{
				{Name: "a", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("3"),
				}},
				{Name: "b", SingleValue: &buildapi.SingleValue{
					ConfigMapValue: &buildapi.ObjectKeyRef{
						Name: "a-config-map",
						Key:  "a-cm-key",
					},
				}},
			}, ContainElements([]buildapi.ParamValue{
				{Name: "a", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("3"),
				}},
				{Name: "b", SingleValue: &buildapi.SingleValue{
					ConfigMapValue: &buildapi.ObjectKeyRef{
						Name: "a-config-map",
						Key:  "a-cm-key",
					},
				}},
			})),

		Entry("override multiple parameters",
			[]buildapi.ParamValue{
				{Name: "a", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("2"),
				}},
				{Name: "b", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("2"),
				}},
				{Name: "c", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("2"),
				}},
			}, []buildapi.ParamValue{
				{Name: "a", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("6"),
				}},
				{Name: "c", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("6"),
				}},
			}, ContainElements([]buildapi.ParamValue{
				{Name: "a", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("6"),
				}},
				{Name: "b", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("2"),
				}},
				{Name: "c", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("6"),
				}},
			})),

		Entry("dont override when second list is empty",
			[]buildapi.ParamValue{
				{Name: "t", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("2"),
				}},
				{Name: "z", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("2"),
				}},
				{Name: "g", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("2"),
				}},
			},
			[]buildapi.ParamValue{
				// no overrides
			},
			ContainElements([]buildapi.ParamValue{
				{Name: "t", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("2"),
				}},
				{Name: "z", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("2"),
				}},
				{Name: "g", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("2"),
				}},
			})),

		Entry("override when first list is empty but not the second list",
			[]buildapi.ParamValue{
				// no original values
			}, []buildapi.ParamValue{
				{Name: "a", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("6"),
				}},
				{Name: "c", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("6"),
				}},
			}, ContainElements([]buildapi.ParamValue{
				{Name: "a", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("6"),
				}},
				{Name: "c", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("6"),
				}},
			})),

		Entry("override multiple parameters if the match and add them if not present in first list",
			[]buildapi.ParamValue{
				{Name: "a", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("2"),
				}},
			}, []buildapi.ParamValue{
				{Name: "a", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("22"),
				}},
				{Name: "b", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("20"),
				}},
				{Name: "c", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("10"),
				}},
				{Name: "d", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("8"),
				}},
				{Name: "e", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("4"),
				}},
			}, ContainElements([]buildapi.ParamValue{
				{Name: "a", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("22"),
				}},
				{Name: "b", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("20"),
				}},
				{Name: "c", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("10"),
				}},
				{Name: "d", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("8"),
				}},
				{Name: "e", SingleValue: &buildapi.SingleValue{
					Value: pointer.String("4"),
				}},
			})),
	)
})

var _ = Describe("IsSystemReservedParameter", func() {

	Context("for a shp-prefixed parameter", func() {

		It("returns true", func() {
			Expect(IsSystemReservedParameter("shp-source-root")).To(BeTrue())
		})
	})

	Context("for a non shp-prefixed paramerer", func() {

		It("returns false", func() {
			Expect(IsSystemReservedParameter("custom-param")).To(BeFalse())
		})
	})
})

var _ = Describe("FindParameterByName", func() {

	Context("For a list of three parameters", func() {

		parameters := []buildapi.Parameter{{
			Name: "some-parameter",
			Type: "string",
		}, {
			Name: "another-parameter",
			Type: "array",
		}, {
			Name: "last-parameter",
			Type: "string",
		}}

		It("returns nil if no parameter with a matching name exists", func() {
			Expect(FindParameterByName(parameters, "non-existing-parameter")).To(BeNil())
		})

		It("returns the correct parameter with a matching name", func() {
			parameter := FindParameterByName(parameters, "another-parameter")
			Expect(parameter).ToNot(BeNil())
			Expect(parameter).To(BeEquivalentTo(&buildapi.Parameter{
				Name: "another-parameter",
				Type: "array",
			}))
		})
	})
})

var _ = Describe("FindParamValueByName", func() {

	Context("For a list of three parameter values", func() {

		paramValues := []buildapi.ParamValue{{
			Name: "some-parameter",
			SingleValue: &buildapi.SingleValue{
				Value: pointer.String("some-value"),
			},
		}, {
			Name: "another-parameter",
			Values: []buildapi.SingleValue{
				{
					Value: pointer.String("item"),
				},
				{
					ConfigMapValue: &buildapi.ObjectKeyRef{
						Name: "a-configmap",
						Key:  "a-key",
					},
				},
			},
		}, {
			Name: "last-parameter",
			SingleValue: &buildapi.SingleValue{
				Value: pointer.String("last-value"),
			},
		}}

		It("returns nil if no parameter with a matching name exists", func() {
			Expect(FindParamValueByName(paramValues, "non-existing-parameter")).To(BeNil())
		})

		It("returns the correct parameter with a matching name", func() {
			parameter := FindParamValueByName(paramValues, "another-parameter")
			Expect(parameter).ToNot(BeNil())
			Expect(parameter).To(BeEquivalentTo(&buildapi.ParamValue{
				Name: "another-parameter",
				Values: []buildapi.SingleValue{
					{
						Value: pointer.String("item"),
					},
					{
						ConfigMapValue: &buildapi.ObjectKeyRef{
							Name: "a-configmap",
							Key:  "a-key",
						},
					},
				},
			}))
		})
	})
})

var _ = Describe("generateEnvVarName", func() {

	Context("For a provided prefix", func() {

		It("returns a variable name with a random suffix", func() {
			name, err := generateEnvVarName(("MY_PREFIX_"))
			Expect(err).ToNot(HaveOccurred())
			Expect(name).To(HavePrefix("MY_PREFIX_"))
			Expect(len(name)).To(Equal(15))
		})
	})
})

var _ = Describe("isStepReferencingParameter", func() {

	Context("for a Step referencing parameters in different ways", func() {

		step := &pipelineapi.Step{
			Command: []string{
				"some-command",
				"$(params.first-param)",
			},
			Args: []string{
				"--flag=$(params['dot.param'])",
				"$(params.array-param[*])",
			},
			Env: []corev1.EnvVar{{
				Name:  "MY_ENV_VAR",
				Value: "hohe $(params[\"another.dot.param\"])",
			}},
		}

		It("returns true for a classical referenced parameter in the command", func() {
			Expect(isStepReferencingParameter(step, "first-param")).To(BeTrue())
		})

		It("returns true for a parameter referenced using brackets in an argument", func() {
			Expect(isStepReferencingParameter(step, "dot.param")).To(BeTrue())
		})

		It("returns true for a parameter referenced using brackets in an environment variable", func() {
			Expect(isStepReferencingParameter(step, "another.dot.param")).To(BeTrue())
		})

		It("returns true for an array referenced parameter in an argument", func() {
			Expect(isStepReferencingParameter(step, "array-param")).To(BeTrue())
		})

		It("returns false for a non-referenced parameter", func() {
			Expect(isStepReferencingParameter(step, "second-param")).To(BeFalse())
		})
	})
})

var _ = Describe("HandleTaskRunParam", func() {

	var taskRun *pipelineapi.TaskRun

	BeforeEach(func() {
		taskRun = &pipelineapi.TaskRun{
			Spec: pipelineapi.TaskRunSpec{
				TaskSpec: &pipelineapi.TaskSpec{
					Steps: []pipelineapi.Step{
						{
							Name: "first-container",
							Args: []string{
								"--an-argument=$(params.string-parameter)",
							},
						},
						{
							Name: "second-container",
							Args: []string{
								"$(params.array-parameter[*])",
							},
						},
					},
				},
			},
		}
	})

	Context("for a string parameter", func() {

		parameterDefinition := &buildapi.Parameter{
			Name: "string-parameter",
			Type: buildapi.ParameterTypeString,
		}

		It("adds a simple value", func() {
			err := HandleTaskRunParam(taskRun, parameterDefinition, buildapi.ParamValue{
				Name: "string-parameter",
				SingleValue: &buildapi.SingleValue{
					Value: pointer.String("My value"),
				},
			})
			Expect(err).ToNot(HaveOccurred())

			Expect(taskRun.Spec.Params).To(BeEquivalentTo([]pipelineapi.Param{
				{
					Name: "string-parameter",
					Value: pipelineapi.ParamValue{
						Type:      pipelineapi.ParamTypeString,
						StringVal: "My value",
					},
				},
			}))
		})

		It("adds a configmap value without a format", func() {
			err := HandleTaskRunParam(taskRun, parameterDefinition, buildapi.ParamValue{
				Name: "string-parameter",
				SingleValue: &buildapi.SingleValue{
					ConfigMapValue: &buildapi.ObjectKeyRef{
						Name: "config-map-name",
						Key:  "my-key",
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			// Verify the environment variable that is only added to the first step
			Expect(len(taskRun.Spec.TaskSpec.Steps[0].Env)).To(Equal(1))
			envVarName := taskRun.Spec.TaskSpec.Steps[0].Env[0].Name

			Expect(envVarName).To(HavePrefix("SHP_CONFIGMAP_PARAM_"))
			Expect(taskRun.Spec.TaskSpec.Steps[0].Env[0]).To(BeEquivalentTo(corev1.EnvVar{
				Name: envVarName,
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "config-map-name",
						},
						Key: "my-key",
					},
				},
			}))

			Expect(len(taskRun.Spec.TaskSpec.Steps[1].Env)).To(Equal(0))

			// Verify the parameters
			Expect(taskRun.Spec.Params).To(BeEquivalentTo([]pipelineapi.Param{
				{
					Name: "string-parameter",
					Value: pipelineapi.ParamValue{
						Type:      pipelineapi.ParamTypeString,
						StringVal: fmt.Sprintf("$(%s)", envVarName),
					},
				},
			}))
		})

		It("adds a configmap value with a format", func() {
			err := HandleTaskRunParam(taskRun, parameterDefinition, buildapi.ParamValue{
				Name: "string-parameter",
				SingleValue: &buildapi.SingleValue{
					ConfigMapValue: &buildapi.ObjectKeyRef{
						Name:   "config-map-name",
						Key:    "my-key",
						Format: pointer.String("The value from the config map is '${CONFIGMAP_VALUE}'."),
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			// Verify the environment variable that is only added to the first step
			Expect(len(taskRun.Spec.TaskSpec.Steps[0].Env)).To(Equal(1))
			envVarName := taskRun.Spec.TaskSpec.Steps[0].Env[0].Name

			Expect(envVarName).To(HavePrefix("SHP_CONFIGMAP_PARAM_"))
			Expect(taskRun.Spec.TaskSpec.Steps[0].Env[0]).To(BeEquivalentTo(corev1.EnvVar{
				Name: envVarName,
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "config-map-name",
						},
						Key: "my-key",
					},
				},
			}))

			Expect(len(taskRun.Spec.TaskSpec.Steps[1].Env)).To(Equal(0))

			// Verify the parameters
			Expect(taskRun.Spec.Params).To(BeEquivalentTo([]pipelineapi.Param{
				{
					Name: "string-parameter",
					Value: pipelineapi.ParamValue{
						Type:      pipelineapi.ParamTypeString,
						StringVal: fmt.Sprintf("The value from the config map is '$(%s)'.", envVarName),
					},
				},
			}))
		})

		It("adds a secret value without a format", func() {
			err := HandleTaskRunParam(taskRun, parameterDefinition, buildapi.ParamValue{
				Name: "string-parameter",
				SingleValue: &buildapi.SingleValue{
					SecretValue: &buildapi.ObjectKeyRef{
						Name: "secret-name",
						Key:  "secret-key",
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			// Verify the environment variable that is only added to the first step
			Expect(len(taskRun.Spec.TaskSpec.Steps[0].Env)).To(Equal(1))
			envVarName := taskRun.Spec.TaskSpec.Steps[0].Env[0].Name

			Expect(envVarName).To(HavePrefix("SHP_SECRET_PARAM_"))
			Expect(taskRun.Spec.TaskSpec.Steps[0].Env[0]).To(BeEquivalentTo(corev1.EnvVar{
				Name: envVarName,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "secret-name",
						},
						Key: "secret-key",
					},
				},
			}))

			Expect(len(taskRun.Spec.TaskSpec.Steps[1].Env)).To(Equal(0))

			// Verify the parameters
			Expect(taskRun.Spec.Params).To(BeEquivalentTo([]pipelineapi.Param{
				{
					Name: "string-parameter",
					Value: pipelineapi.ParamValue{
						Type:      pipelineapi.ParamTypeString,
						StringVal: fmt.Sprintf("$(%s)", envVarName),
					},
				},
			}))
		})

		It("adds a secret value with a format", func() {
			err := HandleTaskRunParam(taskRun, parameterDefinition, buildapi.ParamValue{
				Name: "string-parameter",
				SingleValue: &buildapi.SingleValue{
					SecretValue: &buildapi.ObjectKeyRef{
						Name:   "secret-name",
						Key:    "secret-key",
						Format: pointer.String("secret-value: ${SECRET_VALUE}"),
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			// Verify the environment variable that is only added to the first step
			Expect(len(taskRun.Spec.TaskSpec.Steps[0].Env)).To(Equal(1))
			envVarName := taskRun.Spec.TaskSpec.Steps[0].Env[0].Name

			Expect(envVarName).To(HavePrefix("SHP_SECRET_PARAM_"))
			Expect(taskRun.Spec.TaskSpec.Steps[0].Env[0]).To(BeEquivalentTo(corev1.EnvVar{
				Name: envVarName,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "secret-name",
						},
						Key: "secret-key",
					},
				},
			}))

			Expect(len(taskRun.Spec.TaskSpec.Steps[1].Env)).To(Equal(0))

			// Verify the parameters
			Expect(taskRun.Spec.Params).To(BeEquivalentTo([]pipelineapi.Param{
				{
					Name: "string-parameter",
					Value: pipelineapi.ParamValue{
						Type:      pipelineapi.ParamTypeString,
						StringVal: fmt.Sprintf("secret-value: $(%s)", envVarName),
					},
				},
			}))
		})
	})

	Context("for an array parameter", func() {

		parameterDefinition := &buildapi.Parameter{
			Name: "array-parameter",
			Type: buildapi.ParameterTypeArray,
		}

		It("adds simple values correctly", func() {
			err := HandleTaskRunParam(taskRun, parameterDefinition, buildapi.ParamValue{
				Name: "array-parameter",
				Values: []buildapi.SingleValue{
					{
						Value: pointer.String("first entry"),
					},
					{
						Value: pointer.String(""),
					},
					{
						Value: pointer.String("third entry"),
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			Expect(len(taskRun.Spec.TaskSpec.Steps[1].Env)).To(Equal(0))

			// Verify the parameters
			Expect(taskRun.Spec.Params).To(BeEquivalentTo([]pipelineapi.Param{
				{
					Name: "array-parameter",
					Value: pipelineapi.ParamValue{
						Type: pipelineapi.ParamTypeArray,
						ArrayVal: []string{
							"first entry",
							"",
							"third entry",
						},
					},
				},
			}))
		})

		It("adds values from different sources correctly", func() {
			err := HandleTaskRunParam(taskRun, parameterDefinition, buildapi.ParamValue{
				Name: "array-parameter",
				Values: []buildapi.SingleValue{
					{
						Value: pointer.String("first entry"),
					},
					{
						SecretValue: &buildapi.ObjectKeyRef{
							Name: "secret-name",
							Key:  "secret-key",
						},
					},
					{
						SecretValue: &buildapi.ObjectKeyRef{
							Name:   "secret-name",
							Key:    "secret-key",
							Format: pointer.String("The secret value is ${SECRET_VALUE}"),
						},
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			// Verify the environment variable that is only added to the second step
			Expect(len(taskRun.Spec.TaskSpec.Steps[0].Env)).To(Equal(0))

			Expect(len(taskRun.Spec.TaskSpec.Steps[1].Env)).To(Equal(1))
			envVarName := taskRun.Spec.TaskSpec.Steps[1].Env[0].Name

			Expect(envVarName).To(HavePrefix("SHP_SECRET_PARAM_"))
			Expect(taskRun.Spec.TaskSpec.Steps[1].Env[0]).To(BeEquivalentTo(corev1.EnvVar{
				Name: envVarName,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "secret-name",
						},
						Key: "secret-key",
					},
				},
			}))

			// Verify the parameters
			Expect(taskRun.Spec.Params).To(BeEquivalentTo([]pipelineapi.Param{
				{
					Name: "array-parameter",
					Value: pipelineapi.ParamValue{
						Type: pipelineapi.ParamTypeArray,
						ArrayVal: []string{
							"first entry",
							fmt.Sprintf("$(%s)", envVarName),
							fmt.Sprintf("The secret value is $(%s)", envVarName),
						},
					},
				},
			}))
		})
	})
})
