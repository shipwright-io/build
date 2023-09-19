// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"context"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/ctxlog"
	"github.com/shipwright-io/build/pkg/webhook"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// ensure v1beta1 implements the Conversion interface
var _ webhook.Conversion = (*ClusterBuildStrategy)(nil)

// ConvertTo converts this object to its v1alpha1 equivalent
func (src *ClusterBuildStrategy) ConvertTo(ctx context.Context, obj *unstructured.Unstructured) error {
	ctxlog.Debug(ctx, "Converting ClusterBuildStrategy from beta to alpha", "namespace", src.Namespace, "name", src.Name)

	var bs v1alpha1.ClusterBuildStrategy
	bs.TypeMeta = src.TypeMeta
	bs.TypeMeta.APIVersion = alphaGroupVersion
	bs.ObjectMeta = src.ObjectMeta

	src.Spec.ConvertTo(&bs.Spec)

	mapito, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&bs)
	if err != nil {
		ctxlog.Error(ctx, err, "failed structuring the newObject")
	}
	obj.Object = mapito

	return nil
}

// ConvertFrom converts v1alpha1.ClusterBuildStrategy into this object
func (src *ClusterBuildStrategy) ConvertFrom(ctx context.Context, obj *unstructured.Unstructured) error {
	var cbs v1alpha1.ClusterBuildStrategy

	unstructured := obj.UnstructuredContent()
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured, &cbs)
	if err != nil {
		ctxlog.Error(ctx, err, "failed unstructuring the buildrun convertedObject")
	}

	ctxlog.Debug(ctx, "Converting ClusterBuildStrategy from alpha to beta", "namespace", cbs.Namespace, "name", cbs.Name)

	src.ObjectMeta = cbs.ObjectMeta
	src.TypeMeta = cbs.TypeMeta
	src.TypeMeta.APIVersion = betaGroupVersion

	src.Spec.ConvertFrom(cbs.Spec)

	return nil
}
