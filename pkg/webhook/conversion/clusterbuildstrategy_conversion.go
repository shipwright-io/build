// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package conversion

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	runtime "k8s.io/apimachinery/pkg/runtime"

	buildapialpha "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/ctxlog"
)

// convertClusterBuildStrategyBetaToAlpha converts this object to its v1alpha1 equivalent
func convertClusterBuildStrategyBetaToAlpha(ctx context.Context, src *buildapi.ClusterBuildStrategy, obj *unstructured.Unstructured) error {
	ctxlog.Info(ctx, "converting ClusterBuildStrategy from beta to alpha", "namespace", src.Namespace, "name", src.Name)

	var bs buildapialpha.ClusterBuildStrategy
	bs.TypeMeta = src.TypeMeta
	bs.APIVersion = alphaGroupVersion
	bs.ObjectMeta = src.ObjectMeta

	convertBuildStrategySpecBetaToAlpha((*buildapi.BuildStrategySpec)(&src.Spec), (*buildapialpha.BuildStrategySpec)(&bs.Spec))

	mapito, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&bs)
	if err != nil {
		ctxlog.Error(ctx, err, "failed structuring the newObject")
		return err
	}
	obj.Object = mapito

	return nil
}

// convertClusterBuildStrategyAlphaToBeta converts buildapialpha.ClusterBuildStrategy into this object
func convertClusterBuildStrategyAlphaToBeta(ctx context.Context, src *buildapi.ClusterBuildStrategy, obj *unstructured.Unstructured) error {
	var cbs buildapialpha.ClusterBuildStrategy

	unstructured := obj.UnstructuredContent()
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured, &cbs)
	if err != nil {
		ctxlog.Error(ctx, err, "failed unstructuring the buildrun convertedObject")
		return err
	}

	ctxlog.Info(ctx, "converting ClusterBuildStrategy from alpha to beta", "namespace", cbs.Namespace, "name", cbs.Name)

	src.ObjectMeta = cbs.ObjectMeta
	src.TypeMeta = cbs.TypeMeta
	src.APIVersion = betaGroupVersion

	convertBuildStrategySpecAlphaToBeta((*buildapi.BuildStrategySpec)(&src.Spec), (buildapialpha.BuildStrategySpec)(cbs.Spec))

	return nil
}
