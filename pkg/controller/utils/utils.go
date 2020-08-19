package utils

import (
	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

// IsRuntimeDefined inspect if build has `.spec.runtime` defined, checking intermediary attributes
// and making sure ImageURL is informed.
func IsRuntimeDefined(b *buildv1alpha1.Build) bool {
	if b.Spec.Runtime == nil {
		return false
	}
	if b.Spec.Runtime.Base.ImageURL == "" {
		return false
	}
	return true
}

// IsIsBuilderImageDefined inspect if build contains `.spec.BuilderImage` defined.
func IsBuilderImageDefined(b *buildv1alpha1.Build) bool {
	if b.Spec.BuilderImage == nil {
		return false
	}
	if b.Spec.BuilderImage.ImageURL == "" {
		return false
	}
	return true
}
