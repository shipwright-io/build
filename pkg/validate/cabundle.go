// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/cabundle"
)

// CABundle contains all required fields
// to validate a Build spec certificate definitions
type CABundle struct {
	Build  *buildapi.Build
	Client client.Client
}

func NewCABundle(client client.Client, build *buildapi.Build) *CABundle {
	return &CABundle{build, client}
}

// ValidatePath implements BuildPath interface and validates
// that all referenced secrets or configmaps under certificate exists
func (c *CABundle) ValidatePath(ctx context.Context) error {
	// Skip validation if no CA bundle is specified
	if c.Build.Spec.CABundle == nil {
		return nil
	}
	var statusError *apierrors.StatusError
	if err := cabundle.Validate(ctx, c.Client, c.Build.Spec.CABundle, c.Build.Namespace); err != nil {
		switch {
		case apierrors.IsNotFound(err):
			c.Build.Status.Reason = ptr.To[buildapi.BuildReason](buildapi.CABundleNotFound)
			c.Build.Status.Message = ptr.To(err.Error())
		case !errors.As(err, &statusError):
			c.Build.Status.Reason = ptr.To[buildapi.BuildReason](buildapi.CABundleNotValid)
			c.Build.Status.Message = ptr.To(err.Error())
		default:
			return err
		}
	}

	return nil
}
