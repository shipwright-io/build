// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"fmt"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RuntimeRef contains all required fields
// to validate a Build spec runtime definition
type RuntimeRef struct {
	Build  *build.Build
	Client client.Client
}

// ValidatePath implements BuildPath interface and validates
// that the Build spec runtime definition is properly populated
func (r RuntimeRef) ValidatePath(ctx context.Context) error {
	if resources.IsRuntimeDefined(r.Build) {
		if len(r.Build.Spec.Runtime.Paths) == 0 {
			r.Build.Status.Reason = build.RuntimePathsCanNotBeEmpty
			r.Build.Status.Message = "the property 'spec.runtime.paths' must not be empty"
			return fmt.Errorf("missing some") // TODO
		}
	}
	return nil
}
