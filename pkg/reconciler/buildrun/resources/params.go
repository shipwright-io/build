// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	pipeline "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

const (
	envVarNameSuffixChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

var (
	envVarNameSuffixCharsCount = big.NewInt(int64(len(envVarNameSuffixChars)))
	systemReservedParamKeys    = map[string]bool{
		"BUILDER_IMAGE": true,
		"DOCKERFILE":    true,
		"CONTEXT_DIR":   true,
	}
)

// overrideParams allows to override an existing list of parameters with a second list,
// as long as their entry names matches
func overrideParams(originalParams []buildv1alpha1.ParamValue, overrideParams []buildv1alpha1.ParamValue) []buildv1alpha1.ParamValue {
	if len(overrideParams) == 0 {
		return originalParams
	}

	if len(originalParams) == 0 && len(overrideParams) > 0 {
		return overrideParams
	}

	// Build a map from originalParams
	originalMap := make(map[string]buildv1alpha1.ParamValue)
	for _, p := range originalParams {
		originalMap[p.Name] = p
	}

	// Override param in map where the name matches in both originalParams and overrideParams.
	// Extend map to add parameters that only the overrideParams list contains.
	for _, p := range overrideParams {
		originalMap[p.Name] = p
	}

	// drop results on a slice and return
	paramsList := []buildv1alpha1.ParamValue{}

	for k := range originalMap {
		paramsList = append(paramsList, originalMap[k])
	}

	return paramsList
}

// IsSystemReservedParameter verifies if we are using a system reserved parameter name
func IsSystemReservedParameter(param string) bool {
	return systemReservedParamKeys[param] || strings.HasPrefix(param, "shp-")
}

// FindParameterByName returns the first entry in a Parameter array with a specified name, or nil
func FindParameterByName(parameters []buildv1alpha1.Parameter, name string) *buildv1alpha1.Parameter {
	for _, candidate := range parameters {
		if candidate.Name == name {
			return &candidate
		}
	}

	return nil
}

// FindParamValueByName returns the first entry in a ParamValue array with a specified name, or nil
func FindParamValueByName(paramValues []buildv1alpha1.ParamValue, name string) *buildv1alpha1.ParamValue {
	for _, candidate := range paramValues {
		if candidate.Name == name {
			return &candidate
		}
	}

	return nil
}

// HandleTaskRunParam makes the necessary changes to a TaskRun for a parameter
func HandleTaskRunParam(taskRun *pipeline.TaskRun, parameterDefinition *buildv1alpha1.Parameter, paramValue buildv1alpha1.ParamValue) error {
	taskRunParam := pipeline.Param{
		Name:  paramValue.Name,
		Value: pipeline.ArrayOrString{},
	}

	switch parameterDefinition.Type {
	case "": // string is default
		fallthrough
	case buildv1alpha1.ParameterTypeString:
		taskRunParam.Value.Type = pipeline.ParamTypeString

		switch {
		case paramValue.SingleValue == nil && parameterDefinition.Default == nil:
			// this error should never happen because we validate this upfront in ValidateBuildRunParameters
			return fmt.Errorf("unexpected parameter without any value: %s", parameterDefinition.Name)

		case paramValue.SingleValue == nil:
			// we tolerate this for optional parameters, we enter this code path if a user provides a paramValue without any value. The theoretic but
			// not documented use case is that a Build specifies a value for an optional parameter (which has a default in the strategy). Then one can
			// set a paramValue without a value in the BuildRun to get back to the default value.
			return nil

		case paramValue.SingleValue.ConfigMapValue != nil:
			envVarName, err := addConfigMapEnvVar(taskRun, paramValue.Name, paramValue.SingleValue.ConfigMapValue.Name, paramValue.SingleValue.ConfigMapValue.Key)

			if err != nil {
				return err
			}

			envVarExpression := fmt.Sprintf("$(%s)", envVarName)
			if paramValue.SingleValue.ConfigMapValue.Format != nil {
				taskRunParam.Value.StringVal = strings.ReplaceAll(*paramValue.SingleValue.ConfigMapValue.Format, "${CONFIGMAP_VALUE}", envVarExpression)
			} else {
				taskRunParam.Value.StringVal = envVarExpression
			}

		case paramValue.SingleValue.SecretValue != nil:
			envVarName, err := addSecretEnvVar(taskRun, paramValue.Name, paramValue.SingleValue.SecretValue.Name, paramValue.SingleValue.SecretValue.Key)
			if err != nil {
				return err
			}

			envVarExpression := fmt.Sprintf("$(%s)", envVarName)
			if paramValue.SingleValue.SecretValue.Format != nil {
				taskRunParam.Value.StringVal = strings.ReplaceAll(*paramValue.SingleValue.SecretValue.Format, "${SECRET_VALUE}", envVarExpression)
			} else {
				taskRunParam.Value.StringVal = envVarExpression
			}

		case paramValue.SingleValue.Value != nil:
			taskRunParam.Value.StringVal = *paramValue.SingleValue.Value

		}

	case buildv1alpha1.ParameterTypeArray:
		taskRunParam.Value.Type = pipeline.ParamTypeArray

		switch {
		case paramValue.Values == nil && parameterDefinition.Defaults == nil:
			// this error should never happen because we validate this upfront in ValidateBuildRunParameters
			return fmt.Errorf("unexpected parameter without any value: %s", parameterDefinition.Name)

		case paramValue.Values == nil:
			// we tolerate this for optional parameters, we enter this code path if a user provides a paramValue without any values. The theoretic but
			// not documented use case is that a Build specifies values for an optional parameter (which has defaults in the strategy). Then one can
			// set a paramValue without values in the BuildRun to get back to the default values.
			return nil

		default:
			for index, value := range paramValue.Values {
				switch {
				case value.ConfigMapValue != nil:
					envVarName, err := addConfigMapEnvVar(taskRun, paramValue.Name, value.ConfigMapValue.Name, value.ConfigMapValue.Key)
					if err != nil {
						return err
					}

					envVarExpression := fmt.Sprintf("$(%s)", envVarName)
					if value.ConfigMapValue.Format != nil {
						taskRunParam.Value.ArrayVal = append(taskRunParam.Value.ArrayVal, strings.ReplaceAll(*value.ConfigMapValue.Format, "${CONFIGMAP_VALUE}", envVarExpression))
					} else {
						taskRunParam.Value.ArrayVal = append(taskRunParam.Value.ArrayVal, envVarExpression)
					}

				case value.SecretValue != nil:
					envVarName, err := addSecretEnvVar(taskRun, paramValue.Name, value.SecretValue.Name, value.SecretValue.Key)
					if err != nil {
						return err
					}

					envVarExpression := fmt.Sprintf("$(%s)", envVarName)
					if value.SecretValue.Format != nil {
						taskRunParam.Value.ArrayVal = append(taskRunParam.Value.ArrayVal, strings.ReplaceAll(*value.SecretValue.Format, "${SECRET_VALUE}", envVarExpression))
					} else {
						taskRunParam.Value.ArrayVal = append(taskRunParam.Value.ArrayVal, envVarExpression)
					}
				case value.Value != nil:
					taskRunParam.Value.ArrayVal = append(taskRunParam.Value.ArrayVal, *value.Value)

				default:
					// this error should never happen because we validate this upfront in ValidateBuildRunParameters
					return fmt.Errorf("unexpected parameter without any value: %s[%d]", parameterDefinition.Name, index)

				}
			}
		}
	}

	taskRun.Spec.Params = append(taskRun.Spec.Params, taskRunParam)

	return nil
}

// generateEnvVarName adds a random suffix of five characters or digits to a given prefix
func generateEnvVarName(prefix string) (string, error) {
	result := prefix

	// We add five random characters or digits out of the envVarNameSuffixChars pool which contains
	// capital latin characters and digits as they are allowed in env vars.
	// For this we run a loop which creates a random index from 0 to the number of possible characters,
	// and then appends this character to the result.
	for i := 0; i < 5; i++ {
		num, err := rand.Int(rand.Reader, envVarNameSuffixCharsCount)
		if err != nil {
			return "", err
		}
		result += string(envVarNameSuffixChars[num.Int64()])
	}

	return result, nil
}

// addConfigMapEnvVar modifies all steps which are referencing a parameter name in their command, args, or environment variable values,
// to contain a mapped environment variable for the ConfigMap key. It returns the name of the environment variable name.
func addConfigMapEnvVar(taskRun *pipeline.TaskRun, paramName string, configMapName string, configMapKey string) (string, error) {
	envVarName := ""

	// In this first loop, we check whether any step already references the same ConfigMap key. This can
	// happen when multiple paramValues reference the same ConfigMap key. We then reuse the environment
	// variable name. This ensures that we do not have multiple environment variables that reference the
	// same key. This is done for efficiency.
stepLookupLoop:
	for _, step := range taskRun.Spec.TaskSpec.Steps {
		for _, env := range step.Env {
			if strings.HasPrefix(env.Name, "SHP_CONFIGMAP_PARAM_") && env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil && env.ValueFrom.ConfigMapKeyRef.LocalObjectReference.Name == configMapName && env.ValueFrom.ConfigMapKeyRef.Key == configMapKey {
				envVarName = env.Name
				break stepLookupLoop
			}
		}
	}

	// generate an environment variable name if there is not yet one
	if envVarName == "" {
		generatedEnvVarName, err := generateEnvVarName("SHP_CONFIGMAP_PARAM_")
		if err != nil {
			return "", err
		}
		envVarName = generatedEnvVarName
	}

	// In this second loop, we iterate all the steps and add the environment variable to all steps
	// where the parameter is referenced.
stepModifyLoop:
	for i, step := range taskRun.Spec.TaskSpec.Steps {
		if isStepReferencingParameter(&step, paramName) {
			// make sure we don't add a duplicate environment variable to a step
			for _, env := range step.Env {
				if env.Name == envVarName {
					continue stepModifyLoop
				}
			}

			// we need to prepend our environment variables, this allows environment variables defined in the step of a build strategy to reference our environment variables
			// From: https://kubernetes.io/docs/tasks/inject-data-application/define-environment-variable-container/:
			// > Environment variables may reference each other, however ordering is important. Variables making use of others defined in the same context must come later in the list.
			taskRun.Spec.TaskSpec.Steps[i].Env = append([]corev1.EnvVar{{
				Name: envVarName,
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: configMapName,
						},
						Key: configMapKey,
					},
				},
			}}, taskRun.Spec.TaskSpec.Steps[i].Env...)
		}
	}

	return envVarName, nil
}

// addSecretEnvVar modifies all steps which are referencing a parameter name in their command, args, or environment variable values,
// to contain a mapped environment variable for the Secret key. It returns the name of the environment variable name.
func addSecretEnvVar(taskRun *pipeline.TaskRun, paramName string, secretName string, secretKey string) (string, error) {
	envVarName := ""

	// In this first loop, we check whether any step already references the same Secret key. This can
	// happen when multiple paramValues reference the same Secret key. We then reuse the environment
	// variable name. This ensures that we do not have multiple environment variables that reference the
	// same key. This is done for efficiency.
stepLookupLoop:
	for _, step := range taskRun.Spec.TaskSpec.Steps {
		for _, env := range step.Env {
			if strings.HasPrefix(env.Name, "SHP_SECRET_PARAM_") && env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil && env.ValueFrom.SecretKeyRef.LocalObjectReference.Name == secretName && env.ValueFrom.SecretKeyRef.Key == secretKey {
				envVarName = env.Name
				break stepLookupLoop
			}
		}
	}

	// generate an environment variable name if there is not yet one
	if envVarName == "" {
		generatedEnvVarName, err := generateEnvVarName("SHP_SECRET_PARAM_")
		if err != nil {
			return "", err
		}
		envVarName = generatedEnvVarName
	}

	// In this second loop, we iterate all the steps and add the environment variable to all steps
	// where the parameter is referenced.
stepModifyLoop:
	for i, step := range taskRun.Spec.TaskSpec.Steps {
		if isStepReferencingParameter(&step, paramName) {
			// make sure we don't add a duplicate environment variable to a step
			for _, env := range step.Env {
				if env.Name == envVarName {
					continue stepModifyLoop
				}
			}

			// we need to prepend our environment variables, this allows environment variables defined in the step of a build strategy to reference our environment variables
			// From: https://kubernetes.io/docs/tasks/inject-data-application/define-environment-variable-container/:
			// > Environment variables may reference each other, however ordering is important. Variables making use of others defined in the same context must come later in the list.
			taskRun.Spec.TaskSpec.Steps[i].Env = append([]corev1.EnvVar{{
				Name: envVarName,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: secretName,
						},
						Key: secretKey,
					},
				},
			}}, taskRun.Spec.TaskSpec.Steps[i].Env...)
		}
	}

	return envVarName, nil
}

// isStepReferencingParameter checks if a step is referencing a parameter in its command, args, or environment variable values
func isStepReferencingParameter(step *pipeline.Step, paramName string) bool {
	searchStrings := []string{
		// the trailing ) is intentionally omitted because of arrays
		// Tekton reference: https://github.com/tektoncd/pipeline/blob/main/docs/tasks.md#using-variable-substitution
		fmt.Sprintf("$(params.%s", paramName),
		fmt.Sprintf("$(params['%s']", paramName),
		fmt.Sprintf("$(params[\"%s\"]", paramName),
	}

	for _, command := range step.Command {
		if isStringContainingAnySearchString(command, searchStrings) {
			return true
		}
	}
	for _, arg := range step.Args {
		if isStringContainingAnySearchString(arg, searchStrings) {
			return true
		}
	}
	for _, env := range step.Env {
		if isStringContainingAnySearchString(env.Value, searchStrings) {
			return true
		}
	}

	return false
}

func isStringContainingAnySearchString(aString string, searchStrings []string) bool {
	for _, searchString := range searchStrings {
		if strings.Contains(aString, searchString) {
			return true
		}
	}
	return false
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

// ValidateBuildParameters validates that the parameter values specified in Build are suitable for what is defined in the BuildStrategy
func ValidateBuildParameters(parameterDefinitions []buildv1alpha1.Parameter, buildParamValues []buildv1alpha1.ParamValue) (bool, buildv1alpha1.BuildReason, string) {
	valid, reason, message := validateParameters(parameterDefinitions, buildParamValues, true)
	return valid, buildv1alpha1.BuildReason(reason), message
}

// ValidateBuildRunParameters validates that the parameter values specified in Build and BuildRun
// are suitable for what is defined in the BuildStrategy
func ValidateBuildRunParameters(parameterDefinitions []buildv1alpha1.Parameter, buildParamValues []buildv1alpha1.ParamValue, buildRunParamValues []buildv1alpha1.ParamValue) (bool, string, string) {
	paramValues := overrideParams(buildParamValues, buildRunParamValues)
	return validateParameters(parameterDefinitions, paramValues, false)
}

func validateParameters(parameterDefinitions []buildv1alpha1.Parameter, paramValues []buildv1alpha1.ParamValue, ignoreMissingParameters bool) (bool, string, string) {
	// list of params that collide with reserved system strategy parameters
	undesiredParams := []string{}

	// list of params that are not defined in the strategy
	undefinedParams := []string{}

	// first loop is through the values to catch those that should not be there
	for _, paramValue := range paramValues {
		if isReserved := IsSystemReservedParameter(paramValue.Name); isReserved {
			undesiredParams = append(undesiredParams, paramValue.Name)
		} else {
			parameterDefinition := FindParameterByName(parameterDefinitions, paramValue.Name)
			if parameterDefinition == nil {
				undefinedParams = append(undefinedParams, paramValue.Name)
			}
		}
	}

	if len(undesiredParams) > 0 {
		return false, ConditionRestrictedParametersInUse, fmt.Sprintf("The following parameters are restricted and cannot be set: %s", strings.Join(undesiredParams, ", "))
	}

	if len(undefinedParams) > 0 {
		return false, ConditionUndefinedParameter, fmt.Sprintf("The following parameters are not defined in the build strategy: %s", strings.Join(undefinedParams, ", "))
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
		paramValue := FindParamValueByName(paramValues, parameterDefinition.Name)

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
		return false, ConditionWrongParameterValueType, fmt.Sprintf("The values for the following parameters are using the wrong type: %s", strings.Join(wrongValueTypeParameters, ", "))
	}

	if !ignoreMissingParameters && len(missingParameters) > 0 {
		return false, ConditionMissingParameterValues, fmt.Sprintf("The following parameters are required but no value has been provided: %s", strings.Join(missingParameters, ", "))
	}

	if len(multiValueParams) > 0 {
		return false, ConditionInconsistentParameterValues, fmt.Sprintf("The following parameters have more than one of 'configMapValue', 'secretValue', and 'value' set: %s", strings.Join(multiValueParams, ", "))
	}

	if len(arrayItemEmptyParameters) > 0 {
		return false, ConditionEmptyArrayItemParameterValues, fmt.Sprintf("The values for the following array parameters are containing at least one item where none of 'configMapValue', 'secretValue', and 'value' are set: %s", strings.Join(arrayItemEmptyParameters, ", "))
	}

	if len(incompleteConfigMapValueParameters) > 0 {
		return false, ConditionIncompleteConfigMapValueParameterValues, fmt.Sprintf("The values for the following parameters are containing a 'configMapValue' with an empty 'name' or 'key': %s", strings.Join(incompleteConfigMapValueParameters, ", "))
	}

	if len(incompleteSecretValueParameters) > 0 {
		return false, ConditionIncompleteSecretValueParameterValues, fmt.Sprintf("The values for the following parameters are containing a 'secretValue' with an empty 'name' or 'key': %s", strings.Join(incompleteSecretValueParameters, ", "))
	}

	return true, "", ""
}
