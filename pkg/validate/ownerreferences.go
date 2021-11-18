// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"fmt"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/ctxlog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// OwnerRef contains all required fields
// to validate a Build OwnerReference definition
type OwnerRef struct {
	Build  *build.Build
	Client client.Client
	Scheme *runtime.Scheme
}

// ValidatePath implements BuildPath interface and validates
// setting the ownershipReference between a Build and a BuildRun
func (o OwnerRef) ValidatePath(ctx context.Context) error {

	buildRunList, err := o.retrieveBuildRunsfromBuild(ctx)
	if err != nil {
		return err
	}

	switch o.Build.GetAnnotations()[build.AnnotationBuildRunDeletion] {
	case "true":
		// if the buildRun does not have an ownerreference to the Build, lets add it.
		for _, buildRun := range buildRunList.Items {
			if index := o.validateBuildOwnerReference(buildRun.OwnerReferences); index == -1 {
				if err := controllerutil.SetControllerReference(o.Build, &buildRun, o.Scheme); err != nil {
					o.Build.Status.Reason = build.BuildReasonPtr(build.SetOwnerReferenceFailed)
					o.Build.Status.Message = pointer.StringPtr(fmt.Sprintf("unexpected error when trying to set the ownerreference: %v", err))
				}
				if err = o.Client.Update(ctx, &buildRun); err != nil {
					return err
				}
				ctxlog.Info(ctx, fmt.Sprintf("successfully updated BuildRun %s", buildRun.Name), namespace, buildRun.Namespace, name, buildRun.Name)
			}
		}
	case "", "false":
		// if the buildRun have an ownerreference to the Build, lets remove it
		for _, buildRun := range buildRunList.Items {
			if index := o.validateBuildOwnerReference(buildRun.OwnerReferences); index != -1 {
				buildRun.OwnerReferences = removeOwnerReferenceByIndex(buildRun.OwnerReferences, index)
				if err := o.Client.Update(ctx, &buildRun); err != nil {
					return err
				}
				ctxlog.Info(ctx, fmt.Sprintf("successfully updated BuildRun %s", buildRun.Name), namespace, buildRun.Namespace, name, buildRun.Name)
			}
		}

	default:
		ctxlog.Info(ctx, fmt.Sprintf("the annotation %s was not properly defined, supported values are true or false", build.AnnotationBuildRunDeletion), namespace, o.Build.Namespace, name, o.Build.Name)
		return fmt.Errorf("the annotation %s was not properly defined, supported values are true or false", build.AnnotationBuildRunDeletion)
	}

	return nil
}

// retrieveBuildRunsfromBuild returns a list of BuildRuns that are owned by a Build in the same namespace
func (o OwnerRef) retrieveBuildRunsfromBuild(ctx context.Context) (*build.BuildRunList, error) {
	buildRunList := &build.BuildRunList{}

	lbls := map[string]string{
		build.LabelBuild: o.Build.Name,
	}
	opts := client.ListOptions{
		Namespace:     o.Build.Namespace,
		LabelSelector: labels.SelectorFromSet(lbls),
	}

	err := o.Client.List(ctx, buildRunList, &opts)
	return buildRunList, err
}

// validateOwnerReferences returns an index value if a Build is owning a reference or -1 if this is not the case
func (o OwnerRef) validateBuildOwnerReference(references []metav1.OwnerReference) int {
	for i, ownerRef := range references {
		if ownerRef.Kind == o.Build.Kind && ownerRef.Name == o.Build.Name {
			return i
		}
	}
	return -1
}

// removeOwnerReferenceByIndex removes the entry by index, this will not keep the same
// order in the slice
func removeOwnerReferenceByIndex(references []metav1.OwnerReference, i int) []metav1.OwnerReference {
	return append(references[:i], references[i+1:]...)
}
