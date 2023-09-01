// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package webhook

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Conversion interface {
	ConvertFrom(context.Context, *unstructured.Unstructured) error
	ConvertTo(context.Context, *unstructured.Unstructured) error
}
