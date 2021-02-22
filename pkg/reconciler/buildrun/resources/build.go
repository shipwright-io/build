// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetBuildObject retrieves an existing Build based on a name and namespace
func GetBuildObject(ctx context.Context, client client.Client, objectName string, objectNS string, build *buildv1alpha1.Build) error {
	return client.Get(ctx, types.NamespacedName{Name: objectName, Namespace: objectNS}, build)
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
