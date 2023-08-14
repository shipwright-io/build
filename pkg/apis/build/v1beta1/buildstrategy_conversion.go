// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"context"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/ctxlog"
	"github.com/shipwright-io/build/pkg/webhook"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// ensure v1beta1 implements the Conversion interface
var _ webhook.Conversion = (*BuildStrategy)(nil)

// ConvertTo converts this object to its v1alpha1 equivalent
func (src *BuildStrategy) ConvertTo(ctx context.Context, obj *unstructured.Unstructured) error {
	var bs v1alpha1.BuildStrategy
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

func (src *BuildStrategySpec) ConvertTo(bs *v1alpha1.BuildStrategySpec) {

	bs.BuildSteps = []v1alpha1.BuildStep{}
	for _, step := range src.Steps {

		buildStep := v1alpha1.BuildStep{
			Container: corev1.Container{
				Name:            step.Name,
				Image:           step.Image,
				Command:         step.Command,
				Args:            step.Args,
				WorkingDir:      step.WorkingDir,
				Env:             step.Env,
				Resources:       step.Resources,
				VolumeMounts:    step.VolumeMounts,
				ImagePullPolicy: step.ImagePullPolicy,
			},
		}

		if step.SecurityContext != nil {
			buildStep.SecurityContext = step.SecurityContext
		}

		bs.BuildSteps = append(bs.BuildSteps, buildStep)
	}

	bs.Parameters = []v1alpha1.Parameter{}
	for _, param := range src.Parameters {
		bs.Parameters = append(bs.Parameters, v1alpha1.Parameter{
			Name:        param.Name,
			Description: param.Description,
			Type:        v1alpha1.ParameterType(param.Type),
			Default:     param.Default,
			Defaults:    param.Defaults,
		})
	}

	if src.SecurityContext != nil {
		bs.SecurityContext = (*v1alpha1.BuildStrategySecurityContext)(src.SecurityContext)
	}

	bs.Volumes = []v1alpha1.BuildStrategyVolume{}
	for _, vol := range src.Volumes {
		bs.Volumes = append(bs.Volumes, v1alpha1.BuildStrategyVolume{
			Overridable:  vol.Overridable,
			Name:         vol.Name,
			Description:  vol.Description,
			VolumeSource: vol.VolumeSource,
		})
	}
}

// ConvertFrom converts from v1alpha1.BuildStrategy into this object.
func (src *BuildStrategy) ConvertFrom(ctx context.Context, obj *unstructured.Unstructured) error {
	var br v1alpha1.BuildStrategy

	unstructured := obj.UnstructuredContent()
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured, &br)
	if err != nil {
		ctxlog.Error(ctx, err, "failed unstructuring the buildrun convertedObject")
	}

	src.ObjectMeta = br.ObjectMeta
	src.TypeMeta = br.TypeMeta
	src.TypeMeta.APIVersion = betaGroupVersion

	src.Spec.ConvertFrom(br.Spec)

	return nil
}

func (src *BuildStrategySpec) ConvertFrom(bs v1alpha1.BuildStrategySpec) {
	src.Steps = []Step{}
	for _, brStep := range bs.BuildSteps {
		step := Step{
			Name:            brStep.Name,
			Image:           brStep.Image,
			Command:         brStep.Command,
			Args:            brStep.Args,
			WorkingDir:      brStep.WorkingDir,
			Env:             brStep.Env,
			Resources:       brStep.Resources,
			VolumeMounts:    brStep.VolumeMounts,
			ImagePullPolicy: brStep.ImagePullPolicy,
			SecurityContext: brStep.SecurityContext,
		}
		src.Steps = append(src.Steps, step)
	}

	src.Parameters = []Parameter{}
	for _, param := range bs.Parameters {
		src.Parameters = append(src.Parameters, Parameter{
			Name:        param.Name,
			Description: param.Description,
			Type:        ParameterType(param.Type),
			Default:     param.Default,
			Defaults:    param.Defaults,
		})
	}

	src.SecurityContext = (*BuildStrategySecurityContext)(bs.SecurityContext)

	src.Volumes = []BuildStrategyVolume{}
	for _, vol := range bs.Volumes {
		src.Volumes = append(src.Volumes, BuildStrategyVolume{
			Overridable:  vol.Overridable,
			Name:         vol.Name,
			Description:  vol.Description,
			VolumeSource: vol.VolumeSource,
		})
	}
}
