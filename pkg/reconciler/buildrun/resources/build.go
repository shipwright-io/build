// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"
	"fmt"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetBuildObject retrieves an existing Build based on a name and namespace
func GetBuildObject(ctx context.Context, client client.Client, buildRun *buildv1alpha1.BuildRun, build *buildv1alpha1.Build) error {
	err := client.Get(ctx, types.NamespacedName{Name: buildRun.Spec.BuildRef.Name, Namespace: buildRun.Namespace}, build)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// stop reconciling and mark the BuildRun as Failed
			// we only reconcile again if the status.Update call fails
			if updateErr := UpdateConditionWithFalseStatus(ctx, client, buildRun, fmt.Sprintf("build.shipwright.io \"%s\" not found", buildRun.Spec.BuildRef.Name), ConditionBuildNotFound); updateErr != nil {
				return HandleError("build object not found", err, updateErr)
			}
		}
	}

	return err
}

// IsOwnedByBuild checks if the controllerReferences contains a well known owner Kind
func IsOwnedByBuild(build *buildv1alpha1.Build, controlledReferences []metav1.OwnerReference) bool {
	for _, ref := range controlledReferences {
		if ref.Kind == build.Kind && ref.Name == build.Name {
			return true
		}
	}

	return false
}
