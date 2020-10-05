// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package apis

import (
	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, v1alpha1.SchemeBuilder.AddToScheme)
}
