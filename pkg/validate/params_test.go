// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/validate"
)

var _ = Describe("ValidateBuildRunParameters", func() {
	Context("for a set of parameter definitions", func() {
		parameterDefinitions := []buildapi.Parameter{
			{
				Name: "string-param-no-default",
			},
			{
				Name:    "string-param-with-default",
				Type:    buildapi.ParameterTypeString,
				Default: ptr.To("default value"),
			},
			{
				Name: "array-param-no-defaults",
				Type: buildapi.ParameterTypeArray,
			},
			{
				Name:     "array-param-with-defaults",
				Type:     buildapi.ParameterTypeArray,
				Defaults: &[]string{},
			},
		}

		Context("for parameters just for the required fields", func() {
			buildParamValues := []buildapi.ParamValue{
				{
					Name: "string-param-no-default",
					SingleValue: &buildapi.SingleValue{
						Value: ptr.To("a value"),
					},
				},
			}

			buildRunParamValues := []buildapi.ParamValue{
				{
					Name:   "array-param-no-defaults",
					Values: []buildapi.SingleValue{},
				},
			}

			It("validates without an error", func() {
				valid, _, _ := validate.BuildRunParameters(parameterDefinitions, buildParamValues, buildRunParamValues)
				Expect(valid).To(BeTrue())
			})
		})

		Context("for parameter values from different sources", func() {
			buildParamValues := []buildapi.ParamValue{
				{
					Name: "string-param-no-default",
					SingleValue: &buildapi.SingleValue{
						ConfigMapValue: &buildapi.ObjectKeyRef{
							Name: "a-config-map",
							Key:  "some-key",
						},
					},
				},
				{
					Name: "string-param-with-default",
					// This is invalid but will be corrected in the BuildRun
					Values: []buildapi.SingleValue{},
				},
			}

			buildRunParamValues := []buildapi.ParamValue{
				{
					Name: "array-param-no-defaults",
					Values: []buildapi.SingleValue{
						{
							SecretValue: &buildapi.ObjectKeyRef{
								Name: "a-secret",
								Key:  "my-credential-key",
							},
						},
					},
				},
				{
					Name: "string-param-with-default",
					SingleValue: &buildapi.SingleValue{
						Value: ptr.To(""),
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
				valid, reason, message := validate.BuildRunParameters(parameterDefinitions, []buildapi.ParamValue{}, []buildapi.ParamValue{})
				Expect(valid).To(BeFalse())
				Expect(reason).To(Equal("MissingParameterValues"))
				Expect(message).To(HavePrefix("The following parameters are required but no value has been provided:"))
				Expect(message).To(ContainSubstring("array-param-no-defaults"))
				Expect(message).To(ContainSubstring("string-param-no-default"))
			})
		})

		Context("for a parameter value that is defined but contains no value", func() {
			buildParamValues := []buildapi.ParamValue{
				{
					Name:        "string-param-no-default",
					SingleValue: &buildapi.SingleValue{},
				},
			}

			buildRunParamValues := []buildapi.ParamValue{
				{
					Name:   "array-param-no-defaults",
					Values: []buildapi.SingleValue{},
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
			buildParamValues := []buildapi.ParamValue{
				{
					Name: "string-param-no-default",
					SingleValue: &buildapi.SingleValue{
						Value: ptr.To("a value"),
					},
				},
				{
					Name:   "array-param-no-defaults",
					Values: []buildapi.SingleValue{},
				},
			}

			buildRunParamValues := []buildapi.ParamValue{
				{
					Name: "shp-source-context",
					SingleValue: &buildapi.SingleValue{
						Value: ptr.To("/my-source"),
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
			buildParamValues := []buildapi.ParamValue{
				{
					Name: "string-param-no-default",
					SingleValue: &buildapi.SingleValue{
						Value: ptr.To("a value"),
					},
				},
				{
					Name:   "array-param-no-defaults",
					Values: []buildapi.SingleValue{},
				},
				{
					Name:   "non-existing-parameter-on-build",
					Values: []buildapi.SingleValue{},
				},
			}

			buildRunParamValues := []buildapi.ParamValue{
				{
					Name: "non-existing-parameter",
					SingleValue: &buildapi.SingleValue{
						Value: ptr.To("my value"),
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
			buildParamValues := []buildapi.ParamValue{
				{
					Name: "string-param-no-default",
					SingleValue: &buildapi.SingleValue{
						Value: ptr.To("a value"),
						ConfigMapValue: &buildapi.ObjectKeyRef{
							Name: "a-config-map",
							Key:  "a-key",
						},
					},
				},
			}

			buildRunParamValues := []buildapi.ParamValue{
				{
					Name: "array-param-no-defaults",
					Values: []buildapi.SingleValue{
						{
							Value: ptr.To("a good item"),
						},
						{
							ConfigMapValue: &buildapi.ObjectKeyRef{
								Name: "a-config-map",
								Key:  "a-key",
							},
							SecretValue: &buildapi.ObjectKeyRef{
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
			buildParamValues := []buildapi.ParamValue{
				{
					Name: "string-param-no-default",
					Values: []buildapi.SingleValue{
						{
							Value: ptr.To("an item"),
						},
						{
							ConfigMapValue: &buildapi.ObjectKeyRef{
								Name: "a-config-map",
								Key:  "a-key",
							},
						},
					},
				},
			}

			buildRunParamValues := []buildapi.ParamValue{
				{
					Name: "array-param-no-defaults",
					SingleValue: &buildapi.SingleValue{
						Value: ptr.To("a value"),
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
			buildParamValues := []buildapi.ParamValue{
				{
					Name: "string-param-no-default",
					SingleValue: &buildapi.SingleValue{
						Value: ptr.To(" some value"),
					},
				},
				{
					Name: "array-param-with-defaults",
					Values: []buildapi.SingleValue{
						{
							// the bad item without any value
						},
					},
				},
			}

			buildRunParamValues := []buildapi.ParamValue{
				{
					Name: "array-param-no-defaults",
					Values: []buildapi.SingleValue{
						{
							Value: ptr.To("a good item"),
						},
						{
							// the bad item without any value
						},
						{
							ConfigMapValue: &buildapi.ObjectKeyRef{
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
			buildParamValues := []buildapi.ParamValue{
				{
					Name: "string-param-no-default",
					SingleValue: &buildapi.SingleValue{
						ConfigMapValue: &buildapi.ObjectKeyRef{
							Name: "a-config-map",
						},
					},
				},
			}

			buildRunParamValues := []buildapi.ParamValue{
				{
					Name: "array-param-no-defaults",
					Values: []buildapi.SingleValue{
						{
							Value: ptr.To("an item"),
						},
						{
							ConfigMapValue: &buildapi.ObjectKeyRef{
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
			buildParamValues := []buildapi.ParamValue{
				{
					Name: "string-param-no-default",
					SingleValue: &buildapi.SingleValue{
						SecretValue: &buildapi.ObjectKeyRef{
							Name: "a-secret",
						},
					},
				},
			}

			buildRunParamValues := []buildapi.ParamValue{
				{
					Name: "array-param-no-defaults",
					Values: []buildapi.SingleValue{
						{
							Value: ptr.To("an item"),
						},
						{
							SecretValue: &buildapi.ObjectKeyRef{
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
