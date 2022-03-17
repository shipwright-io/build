// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/utils/pointer"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/validate"
)

var _ = Describe("ValidateBuildRunParameters", func() {
	Context("for a set of parameter definitions", func() {
		parameterDefinitions := []buildv1alpha1.Parameter{
			{
				Name: "string-param-no-default",
			},
			{
				Name:    "string-param-with-default",
				Type:    buildv1alpha1.ParameterTypeString,
				Default: pointer.String("default value"),
			},
			{
				Name: "array-param-no-defaults",
				Type: buildv1alpha1.ParameterTypeArray,
			},
			{
				Name:     "array-param-with-defaults",
				Type:     buildv1alpha1.ParameterTypeArray,
				Defaults: &[]string{},
			},
		}

		Context("for parameters just for the required fields", func() {
			buildParamValues := []buildv1alpha1.ParamValue{
				{
					Name: "string-param-no-default",
					SingleValue: &buildv1alpha1.SingleValue{
						Value: pointer.String("a value"),
					},
				},
			}

			buildRunParamValues := []buildv1alpha1.ParamValue{
				{
					Name:   "array-param-no-defaults",
					Values: []buildv1alpha1.SingleValue{},
				},
			}

			It("validates without an error", func() {
				valid, _, _ := validate.BuildRunParameters(parameterDefinitions, buildParamValues, buildRunParamValues)
				Expect(valid).To(BeTrue())
			})
		})

		Context("for parameter values from different sources", func() {
			buildParamValues := []buildv1alpha1.ParamValue{
				{
					Name: "string-param-no-default",
					SingleValue: &buildv1alpha1.SingleValue{
						ConfigMapValue: &buildv1alpha1.ObjectKeyRef{
							Name: "a-config-map",
							Key:  "some-key",
						},
					},
				},
				{
					Name: "string-param-with-default",
					// This is invalid but will be corrected in the BuildRun
					Values: []buildv1alpha1.SingleValue{},
				},
			}

			buildRunParamValues := []buildv1alpha1.ParamValue{
				{
					Name: "array-param-no-defaults",
					Values: []buildv1alpha1.SingleValue{
						{
							SecretValue: &buildv1alpha1.ObjectKeyRef{
								Name: "a-secret",
								Key:  "my-credential-key",
							},
						},
					},
				},
				{
					Name: "string-param-with-default",
					SingleValue: &buildv1alpha1.SingleValue{
						Value: pointer.String(""),
					},
				},
			}

			It("validates without an error", func() {
				valid, _, _ := validate.BuildRunParameters(parameterDefinitions, buildParamValues, buildRunParamValues)
				Expect(valid).To(BeTrue())
			})
		})

		Context("for parameter values that are missing", func() {

			It("validates with the correct validation error", func() {
				valid, reason, message := validate.BuildRunParameters(parameterDefinitions, []buildv1alpha1.ParamValue{}, []buildv1alpha1.ParamValue{})
				Expect(valid).To(BeFalse())
				Expect(reason).To(Equal("MissingParameterValues"))
				Expect(message).To(HavePrefix("The following parameters are required but no value has been provided:"))
				Expect(message).To(ContainSubstring("array-param-no-defaults"))
				Expect(message).To(ContainSubstring("string-param-no-default"))
			})
		})

		Context("for a parameter value that is defined but contains no value", func() {
			buildParamValues := []buildv1alpha1.ParamValue{
				{
					Name:        "string-param-no-default",
					SingleValue: &buildv1alpha1.SingleValue{},
				},
			}

			buildRunParamValues := []buildv1alpha1.ParamValue{
				{
					Name:   "array-param-no-defaults",
					Values: []buildv1alpha1.SingleValue{},
				},
			}

			It("validates with the correct validation error", func() {
				valid, reason, message := validate.BuildRunParameters(parameterDefinitions, buildParamValues, buildRunParamValues)
				Expect(valid).To(BeFalse())
				Expect(reason).To(Equal("MissingParameterValues"))
				Expect(message).To(Equal("The following parameters are required but no value has been provided: string-param-no-default"))
			})
		})

		Context("for parameter values that contain a value for a system parameter", func() {
			buildParamValues := []buildv1alpha1.ParamValue{
				{
					Name: "string-param-no-default",
					SingleValue: &buildv1alpha1.SingleValue{
						Value: pointer.String("a value"),
					},
				},
				{
					Name:   "array-param-no-defaults",
					Values: []buildv1alpha1.SingleValue{},
				},
			}

			buildRunParamValues := []buildv1alpha1.ParamValue{
				{
					Name: "shp-source-context",
					SingleValue: &buildv1alpha1.SingleValue{
						Value: pointer.String("/my-source"),
					},
				},
			}

			It("validates with the correct validation error", func() {
				valid, reason, message := validate.BuildRunParameters(parameterDefinitions, buildParamValues, buildRunParamValues)
				Expect(valid).To(BeFalse())
				Expect(reason).To(Equal("RestrictedParametersInUse"))
				Expect(message).To(Equal("The following parameters are restricted and cannot be set: shp-source-context"))
			})
		})

		Context("for parameter values that are not defined in the build strategy", func() {
			buildParamValues := []buildv1alpha1.ParamValue{
				{
					Name: "string-param-no-default",
					SingleValue: &buildv1alpha1.SingleValue{
						Value: pointer.String("a value"),
					},
				},
				{
					Name:   "array-param-no-defaults",
					Values: []buildv1alpha1.SingleValue{},
				},
				{
					Name:   "non-existing-parameter-on-build",
					Values: []buildv1alpha1.SingleValue{},
				},
			}

			buildRunParamValues := []buildv1alpha1.ParamValue{
				{
					Name: "non-existing-parameter",
					SingleValue: &buildv1alpha1.SingleValue{
						Value: pointer.String("my value"),
					},
				},
			}

			It("validates with the correct validation error", func() {
				valid, reason, message := validate.BuildRunParameters(parameterDefinitions, buildParamValues, buildRunParamValues)
				Expect(valid).To(BeFalse())
				Expect(reason).To(Equal("UndefinedParameter"))
				Expect(message).To(HavePrefix("The following parameters are not defined in the build strategy:"))
				Expect(message).To(ContainSubstring("non-existing-parameter-on-build"))
				Expect(message).To(ContainSubstring("non-existing-parameter"))
			})
		})

		Context("for parameter values that contain more than one value", func() {
			buildParamValues := []buildv1alpha1.ParamValue{
				{
					Name: "string-param-no-default",
					SingleValue: &buildv1alpha1.SingleValue{
						Value: pointer.String("a value"),
						ConfigMapValue: &buildv1alpha1.ObjectKeyRef{
							Name: "a-config-map",
							Key:  "a-key",
						},
					},
				},
			}

			buildRunParamValues := []buildv1alpha1.ParamValue{
				{
					Name: "array-param-no-defaults",
					Values: []buildv1alpha1.SingleValue{
						{
							Value: pointer.String("a good item"),
						},
						{
							ConfigMapValue: &buildv1alpha1.ObjectKeyRef{
								Name: "a-config-map",
								Key:  "a-key",
							},
							SecretValue: &buildv1alpha1.ObjectKeyRef{
								Name: "a-secret",
								Key:  "a-key",
							},
						},
					},
				},
			}

			It("validates with the correct validation error", func() {
				valid, reason, message := validate.BuildRunParameters(parameterDefinitions, buildParamValues, buildRunParamValues)
				Expect(valid).To(BeFalse())
				Expect(reason).To(Equal("InconsistentParameterValues"))
				Expect(message).To(HavePrefix("The following parameters have more than one of 'configMapValue', 'secretValue', and 'value' set:"))
				Expect(message).To(ContainSubstring("string-param-no-default"))
				Expect(message).To(ContainSubstring("array-param-no-defaults"))
			})
		})

		Context("for parameter values that use the wrong type", func() {
			buildParamValues := []buildv1alpha1.ParamValue{
				{
					Name: "string-param-no-default",
					Values: []buildv1alpha1.SingleValue{
						{
							Value: pointer.String("an item"),
						},
						{
							ConfigMapValue: &buildv1alpha1.ObjectKeyRef{
								Name: "a-config-map",
								Key:  "a-key",
							},
						},
					},
				},
			}

			buildRunParamValues := []buildv1alpha1.ParamValue{
				{
					Name: "array-param-no-defaults",
					SingleValue: &buildv1alpha1.SingleValue{
						Value: pointer.String("a value"),
					},
				},
			}

			It("validates with the correct validation error", func() {
				valid, reason, message := validate.BuildRunParameters(parameterDefinitions, buildParamValues, buildRunParamValues)
				Expect(valid).To(BeFalse())
				Expect(reason).To(Equal("WrongParameterValueType"))
				Expect(message).To(HavePrefix("The values for the following parameters are using the wrong type:"))
				Expect(message).To(ContainSubstring("string-param-no-default"))
				Expect(message).To(ContainSubstring("array-param-no-defaults"))
			})
		})

		Context("for array parameter values that contain empty items", func() {
			buildParamValues := []buildv1alpha1.ParamValue{
				{
					Name: "string-param-no-default",
					SingleValue: &buildv1alpha1.SingleValue{
						Value: pointer.String(" some value"),
					},
				},
				{
					Name: "array-param-with-defaults",
					Values: []buildv1alpha1.SingleValue{
						{
							// the bad item without any value
						},
					},
				},
			}

			buildRunParamValues := []buildv1alpha1.ParamValue{
				{
					Name: "array-param-no-defaults",
					Values: []buildv1alpha1.SingleValue{
						{
							Value: pointer.String("a good item"),
						},
						{
							// the bad item without any value
						},
						{
							ConfigMapValue: &buildv1alpha1.ObjectKeyRef{
								Name: "a-configmap",
								Key:  "a-key",
							},
						},
					},
				},
			}

			It("validates with the correct validation error", func() {
				valid, reason, message := validate.BuildRunParameters(parameterDefinitions, buildParamValues, buildRunParamValues)
				Expect(valid).To(BeFalse())
				Expect(reason).To(Equal("EmptyArrayItemParameterValues"))
				Expect(message).To(HavePrefix("The values for the following array parameters are containing at least one item where none of 'configMapValue', 'secretValue', and 'value' are set:"))
				Expect(message).To(ContainSubstring("array-param-with-defaults"))
				Expect(message).To(ContainSubstring("array-param-no-defaults"))
			})
		})

		Context("for parameter values that contain incomplete configMapValues", func() {
			buildParamValues := []buildv1alpha1.ParamValue{
				{
					Name: "string-param-no-default",
					SingleValue: &buildv1alpha1.SingleValue{
						ConfigMapValue: &buildv1alpha1.ObjectKeyRef{
							Name: "a-config-map",
						},
					},
				},
			}

			buildRunParamValues := []buildv1alpha1.ParamValue{
				{
					Name: "array-param-no-defaults",
					Values: []buildv1alpha1.SingleValue{
						{
							Value: pointer.String("an item"),
						},
						{
							ConfigMapValue: &buildv1alpha1.ObjectKeyRef{
								Key: "a-key",
							},
						},
					},
				},
			}

			It("validates with the correct validation error", func() {
				valid, reason, message := validate.BuildRunParameters(parameterDefinitions, buildParamValues, buildRunParamValues)
				Expect(valid).To(BeFalse())
				Expect(reason).To(Equal("IncompleteConfigMapValueParameterValues"))
				Expect(message).To(HavePrefix("The values for the following parameters are containing a 'configMapValue' with an empty 'name' or 'key':"))
				Expect(message).To(ContainSubstring("string-param-no-default"))
				Expect(message).To(ContainSubstring("array-param-no-defaults"))
			})
		})

		Context("for parameter values that contain incomplete secretValues", func() {
			buildParamValues := []buildv1alpha1.ParamValue{
				{
					Name: "string-param-no-default",
					SingleValue: &buildv1alpha1.SingleValue{
						SecretValue: &buildv1alpha1.ObjectKeyRef{
							Name: "a-secret",
						},
					},
				},
			}

			buildRunParamValues := []buildv1alpha1.ParamValue{
				{
					Name: "array-param-no-defaults",
					Values: []buildv1alpha1.SingleValue{
						{
							Value: pointer.String("an item"),
						},
						{
							SecretValue: &buildv1alpha1.ObjectKeyRef{
								Key: "a-key",
							},
						},
					},
				},
			}

			It("validates with the correct validation error", func() {
				valid, reason, message := validate.BuildRunParameters(parameterDefinitions, buildParamValues, buildRunParamValues)
				Expect(valid).To(BeFalse())
				Expect(reason).To(Equal("IncompleteSecretValueParameterValues"))
				Expect(message).To(HavePrefix("The values for the following parameters are containing a 'secretValue' with an empty 'name' or 'key':"))
				Expect(message).To(ContainSubstring("string-param-no-default"))
				Expect(message).To(ContainSubstring("array-param-no-defaults"))
			})
		})
	})
})
