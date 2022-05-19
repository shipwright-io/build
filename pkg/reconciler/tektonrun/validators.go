package tektonrun

import (
	tektonv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateTektonRun validates that the provided Tekton Run object correctly specifies the
// Shipwright Build custom task for Tekton.
func ValidateTektonRun(tektonRun *tektonv1alpha1.Run) error {
	var allErrs field.ErrorList

	path := field.NewPath("spec")

	if err := validateRunEmbeddedSpec(tektonRun.Spec.Spec, path.Child("spec")); err != nil {
		allErrs = append(allErrs, err)
	}

	if errs := validateRunEmbeddedRef(tektonRun.Spec.Ref, path.Child("ref")); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	if err := validateRunTimeout(tektonRun.Spec.Timeout, path.Child("timeout")); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := validateRunRetires(tektonRun.Spec.Retries, path.Child("retries")); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := validateRunParameters(tektonRun.Spec.Params, path.Child("params")); err != nil {
		allErrs = append(allErrs, err...)
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.ParseGroupKind("Run.tekton.dev"),
		tektonRun.Name,
		allErrs,
	)
}

func validateRunEmbeddedSpec(embeddedSpec *tektonv1alpha1.EmbeddedRunSpec, path *field.Path) *field.Error {
	if embeddedSpec != nil {
		return field.Invalid(path, "<object>", "embedded custom task spec is not supported")
	}
	return nil
}

func validateRunEmbeddedRef(embeddedRef *tektonv1beta1.TaskRef, path *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	if embeddedRef == nil {
		allErrs = append(allErrs, field.Required(path, "custom task reference must be provided"))
		return allErrs
	}

	if err := validateAPIVersion(embeddedRef.APIVersion, path.Child("apiVersion")); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := validateKind(embeddedRef.Kind, path.Child("kind")); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := validateName(embeddedRef.Name, path.Child("name")); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) > 0 {
		return allErrs
	}
	return nil
}

func validateAPIVersion(apiVersion string, path *field.Path) *field.Error {
	if apiVersion != "shipwright.io/v1alpha1" {
		return field.Invalid(path, apiVersion, "apiVersion must be shipwright.io/v1alpha1")
	}
	return nil
}

func validateKind(kind tektonv1beta1.TaskKind, path *field.Path) *field.Error {
	if kind != "Build" {
		return field.Invalid(path, kind, "kind must be Build")
	}
	return nil
}

func validateName(name string, path *field.Path) *field.Error {
	if len(name) == 0 {
		return field.Required(path, "build name is required")
	}
	return nil
}

func validateRunTimeout(timeout *metav1.Duration, path *field.Path) *field.Error {
	// TODO: Timeouts are effectively ignored by custom task implementations, but can be populated
	// default in a pipeline. Provide a warning that the timeout is ignored?
	return nil
}

func validateRunRetires(retries int, path *field.Path) *field.Error {
	if retries != 0 {
		return field.Invalid(path, retries, "retries are not supported")
	}
	return nil
}

func validateRunParameters(params []tektonv1beta1.Param, path *field.Path) field.ErrorList {
	// Tekton alpha APIs introduce implicit parameters, which need to be passed through
	return nil
}
