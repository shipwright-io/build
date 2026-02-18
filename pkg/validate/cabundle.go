// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/cabundle"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CABundle contains all required fields
// to validate a Build spec certificate definitions
type CABundle struct {
	Build  *buildv1beta1.Build
	Client client.Client
}

func NewCABundle(client client.Client, build *buildv1beta1.Build) *CABundle {
	return &CABundle{build, client}
}

// ValidatePath implements BuildPath interface and validates
// that all referenced secrets or configmaps under certificate exists
func (c *CABundle) ValidatePath(ctx context.Context) error {
	if err := cabundle.Validate(ctx, c.Client, c.Build.Spec.CABundle, c.Build.Namespace); apierrors.IsNotFound(err) {
		c.Build.Status.Reason = ptr.To[buildv1beta1.BuildReason](buildv1beta1.CABundleReferenceNotFound)
		c.Build.Status.Message = ptr.To(err.Error())
	} else if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}
