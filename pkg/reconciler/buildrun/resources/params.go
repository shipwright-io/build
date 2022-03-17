// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	corev1 "k8s.io/api/core/v1"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	pipeline "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
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

// OverrideParams allows to override an existing list of parameters with a second list,
// as long as their entry names matches
func OverrideParams(originalParams []buildv1alpha1.ParamValue, overrideParams []buildv1alpha1.ParamValue) []buildv1alpha1.ParamValue {
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
