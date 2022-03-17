// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"fmt"
	"strings"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
)

// BuildParameters validates that the parameter values specified in Build are suitable for what is defined in the BuildStrategy
func BuildParameters(parameterDefinitions []buildv1alpha1.Parameter, buildParamValues []buildv1alpha1.ParamValue) (bool, buildv1alpha1.BuildReason, string) {
	valid, reason, message := validateParameters(parameterDefinitions, buildParamValues, true)
	return valid, buildv1alpha1.BuildReason(reason), message
}

// BuildRunParameters validates that the parameter values specified in Build and BuildRun are suitable for what is defined in the BuildStrategy
func BuildRunParameters(parameterDefinitions []buildv1alpha1.Parameter, buildParamValues []buildv1alpha1.ParamValue, buildRunParamValues []buildv1alpha1.ParamValue) (bool, string, string) {
	paramValues := resources.OverrideParams(buildParamValues, buildRunParamValues)
	return validateParameters(parameterDefinitions, paramValues, false)
}

func validateParameters(parameterDefinitions []buildv1alpha1.Parameter, paramValues []buildv1alpha1.ParamValue, ignoreMissingParameters bool) (bool, string, string) {
	// list of params that collide with reserved system strategy parameters
	undesiredParams := []string{}

	// list of params that are not defined in the strategy
	undefinedParams := []string{}

	// first loop is through the values to catch those that should not be there
	for _, paramValue := range paramValues {
		if isReserved := resources.IsSystemReservedParameter(paramValue.Name); isReserved {
			undesiredParams = append(undesiredParams, paramValue.Name)
		} else {
			parameterDefinition := resources.FindParameterByName(parameterDefinitions, paramValue.Name)
			if parameterDefinition == nil {
				undefinedParams = append(undefinedParams, paramValue.Name)
			}
		}
	}

	if len(undesiredParams) > 0 {
		return false, resources.ConditionRestrictedParametersInUse, fmt.Sprintf("The following parameters are restricted and cannot be set: %s", strings.Join(undesiredParams, ", "))
	}

	if len(undefinedParams) > 0 {
		return false, resources.ConditionUndefinedParameter, fmt.Sprintf("The following parameters are not defined in the build strategy: %s", strings.Join(undefinedParams, ", "))
	}

	// list of parameters where the value is of the wrong type
	wrongValueTypeParameters := []string{}

	// list of parameters where there is no value
	missingParameters := []string{}

	// list of parameters where at least one array item is empty
	arrayItemEmptyParameters := []string{}

	// list of params that have multiple values set
	multiValueParams := []string{}

	// list of params that have incomplete ConfigMap values
	incompleteConfigMapValueParameters := []string{}

	// list of params that have incomplete Secret values
	incompleteSecretValueParameters := []string{}

	// second loop is through the strategy parameters to determine those with missing or incorrect values
	for _, parameterDefinition := range parameterDefinitions {
		paramValue := resources.FindParamValueByName(paramValues, parameterDefinition.Name)

		switch parameterDefinition.Type {
		case "": // string is default
			fallthrough
		case buildv1alpha1.ParameterTypeString:
			if paramValue != nil {
				// check if a string value contains array values
				if paramValue.Values != nil {
					wrongValueTypeParameters = append(wrongValueTypeParameters, parameterDefinition.Name)
				}

				if paramValue.SingleValue != nil {
					// check if a single value contains multiple values
					if hasMoreThanOneValue(*paramValue.SingleValue) {
						multiValueParams = append(multiValueParams, parameterDefinition.Name)
					}

					// check if a configmap value is incomplete
					if hasIncompleteConfigMapValue(*paramValue.SingleValue) {
						incompleteConfigMapValueParameters = append(incompleteConfigMapValueParameters, parameterDefinition.Name)
					}

					// check if a secret value is incomplete
					if hasIncompleteSecretValue(*paramValue.SingleValue) {
						incompleteSecretValueParameters = append(incompleteSecretValueParameters, parameterDefinition.Name)
					}
				}
			}

			// check if a string parameter without default has no value
			if parameterDefinition.Default == nil && (paramValue == nil || paramValue.SingleValue == nil || hasNoValue(*paramValue.SingleValue)) {
				missingParameters = append(missingParameters, parameterDefinition.Name)
				continue
			}

		case buildv1alpha1.ParameterTypeArray:
			if paramValue != nil {
				// check if an array value contains a single value
				if paramValue.SingleValue != nil {
					wrongValueTypeParameters = append(wrongValueTypeParameters, parameterDefinition.Name)
				}

				// check whether any array item contains no value
				for _, arrayItemParamValue := range paramValue.Values {
					if hasNoValue(arrayItemParamValue) {
						arrayItemEmptyParameters = append(arrayItemEmptyParameters, parameterDefinition.Name)
						break
					}
				}

				// check whether any array item has more than one value
				for _, arrayItemParamValue := range paramValue.Values {
					if hasMoreThanOneValue(arrayItemParamValue) {
						multiValueParams = append(multiValueParams, parameterDefinition.Name)
						break
					}
				}

				// check whether any array item has an incomplete configMapValue
				for _, arrayItemParamValue := range paramValue.Values {
					if hasIncompleteConfigMapValue(arrayItemParamValue) {
						incompleteConfigMapValueParameters = append(incompleteConfigMapValueParameters, parameterDefinition.Name)
					}
				}

				// check whether any array item has an incomplete secretValue
				for _, arrayItemParamValue := range paramValue.Values {
					if hasIncompleteSecretValue(arrayItemParamValue) {
						incompleteSecretValueParameters = append(incompleteSecretValueParameters, parameterDefinition.Name)
					}
				}
			}

			// check if an array parameter without defaults has no values
			if parameterDefinition.Defaults == nil && (paramValue == nil || paramValue.Values == nil) {
				missingParameters = append(missingParameters, parameterDefinition.Name)
			}
		}
	}

	if len(wrongValueTypeParameters) > 0 {
		return false, resources.ConditionWrongParameterValueType, fmt.Sprintf("The values for the following parameters are using the wrong type: %s", strings.Join(wrongValueTypeParameters, ", "))
	}

	if !ignoreMissingParameters && len(missingParameters) > 0 {
		return false, resources.ConditionMissingParameterValues, fmt.Sprintf("The following parameters are required but no value has been provided: %s", strings.Join(missingParameters, ", "))
	}

	if len(multiValueParams) > 0 {
		return false, resources.ConditionInconsistentParameterValues, fmt.Sprintf("The following parameters have more than one of 'configMapValue', 'secretValue', and 'value' set: %s", strings.Join(multiValueParams, ", "))
	}

	if len(arrayItemEmptyParameters) > 0 {
		return false, resources.ConditionEmptyArrayItemParameterValues, fmt.Sprintf("The values for the following array parameters are containing at least one item where none of 'configMapValue', 'secretValue', and 'value' are set: %s", strings.Join(arrayItemEmptyParameters, ", "))
	}

	if len(incompleteConfigMapValueParameters) > 0 {
		return false, resources.ConditionIncompleteConfigMapValueParameterValues, fmt.Sprintf("The values for the following parameters are containing a 'configMapValue' with an empty 'name' or 'key': %s", strings.Join(incompleteConfigMapValueParameters, ", "))
	}

	if len(incompleteSecretValueParameters) > 0 {
		return false, resources.ConditionIncompleteSecretValueParameterValues, fmt.Sprintf("The values for the following parameters are containing a 'secretValue' with an empty 'name' or 'key': %s", strings.Join(incompleteSecretValueParameters, ", "))
	}

	return true, "", ""
}

// hasMoreThanOneValue checks if a SingleValue has more than one value set (plain text, secret, and config map key reference)
func hasMoreThanOneValue(singleValue buildv1alpha1.SingleValue) bool {
	if singleValue.Value != nil && (singleValue.ConfigMapValue != nil || singleValue.SecretValue != nil) {
		return true
	}
	if singleValue.ConfigMapValue != nil && (singleValue.SecretValue != nil || singleValue.Value != nil) {
		return true
	}
	if singleValue.SecretValue != nil && (singleValue.ConfigMapValue != nil || singleValue.Value != nil) {
		return true
	}

	return false
}

// hasNoValue checks if a SingleValue has no value set (plain text, secret, and config map key reference)
func hasNoValue(singleValue buildv1alpha1.SingleValue) bool {
	if singleValue.ConfigMapValue == nil && singleValue.SecretValue == nil && singleValue.Value == nil {
		return true
	}

	return false
}

// hasIncompleteConfigMapValue checks if a SingleValue has a ConfigMap value with an empty name or key
func hasIncompleteConfigMapValue(singleValue buildv1alpha1.SingleValue) bool {
	if singleValue.ConfigMapValue != nil && (singleValue.ConfigMapValue.Name == "" || singleValue.ConfigMapValue.Key == "") {
		return true
	}

	return false
}

// hasIncompleteSecretValue checks if a SingleValue has a Secret value with an empty name or key
func hasIncompleteSecretValue(singleValue buildv1alpha1.SingleValue) bool {
	if singleValue.SecretValue != nil && (singleValue.SecretValue.Name == "" || singleValue.SecretValue.Key == "") {
		return true
	}

	return false
}
