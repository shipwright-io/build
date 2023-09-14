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
var _ webhook.Conversion = (*BuildRun)(nil)

// To Alpha
func (src *BuildRun) ConvertTo(ctx context.Context, obj *unstructured.Unstructured) error {
	ctxlog.Debug(ctx, "Converting BuildRun from beta to alpha", "namespace", src.Namespace, "name", src.Name)

	var alphaBuildRun v1alpha1.BuildRun

	alphaBuildRun.TypeMeta = src.TypeMeta
	alphaBuildRun.TypeMeta.APIVersion = alphaGroupVersion
	alphaBuildRun.ObjectMeta = src.ObjectMeta

	// BuildRunSpec BuildSpec
	if src.Spec.Build.Build != nil {
		newBuildSpec := v1alpha1.BuildSpec{}
		if err := src.Spec.Build.Build.ConvertTo(&newBuildSpec); err != nil {
			return err
		}
		alphaBuildRun.Spec.BuildSpec = &newBuildSpec
	} else {
		alphaBuildRun.Spec.BuildRef = &v1alpha1.BuildRef{
			Name: src.Spec.Build.Name,
		}
	}

	// BuildRunSpec ServiceAccount
	alphaBuildRun.Spec.ServiceAccount = &v1alpha1.ServiceAccount{
		Name: src.Spec.ServiceAccount,
	}

	// BuildRunSpec Timeout
	alphaBuildRun.Spec.Timeout = src.Spec.Timeout

	// BuildRunSpec ParamValues
	alphaBuildRun.Spec.ParamValues = nil
	for _, p := range src.Spec.ParamValues {
		param := v1alpha1.ParamValue{}
		p.convertToAlpha(&param)
		alphaBuildRun.Spec.ParamValues = append(alphaBuildRun.Spec.ParamValues, param)
	}

	// BuildRunSpec Image

	if src.Spec.Output != nil {
		alphaBuildRun.Spec.Output = &v1alpha1.Image{
			Image:       src.Spec.Output.Image,
			Annotations: src.Spec.Output.Annotations,
			Labels:      src.Spec.Output.Labels,
		}
		if src.Spec.Output.PushSecret != nil {
			alphaBuildRun.Spec.Output.Credentials = &corev1.LocalObjectReference{
				Name: *src.Spec.Output.PushSecret,
			}
		}
	}

	// BuildRunSpec State
	alphaBuildRun.Spec.State = (*v1alpha1.BuildRunRequestedState)(src.Spec.State)

	// BuildRunSpec Env
	alphaBuildRun.Spec.Env = src.Spec.Env

	// BuildRunSpec Retention
	alphaBuildRun.Spec.Retention = (*v1alpha1.BuildRunRetention)(src.Spec.Retention)

	// BuildRunSpec Volumes
	alphaBuildRun.Spec.Volumes = []v1alpha1.BuildVolume{}
	for _, vol := range src.Spec.Volumes {
		alphaBuildRun.Spec.Volumes = append(alphaBuildRun.Spec.Volumes, v1alpha1.BuildVolume{
			Name:         vol.Name,
			VolumeSource: vol.VolumeSource,
		})
	}

	mapito, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&alphaBuildRun)
	if err != nil {
		ctxlog.Error(ctx, err, "failed structuring the newObject")
	}
	obj.Object = mapito

	return nil

}

// From Alpha
func (src *BuildRun) ConvertFrom(ctx context.Context, obj *unstructured.Unstructured) error {

	var alphaBuildRun v1alpha1.BuildRun

	unstructured := obj.UnstructuredContent()
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured, &alphaBuildRun)
	if err != nil {
		ctxlog.Error(ctx, err, "failed unstructuring the buildrun convertedObject")
	}

	ctxlog.Debug(ctx, "Converting BuildRun from alpha to beta", "namespace", alphaBuildRun.Namespace, "name", alphaBuildRun.Name)

	src.ObjectMeta = alphaBuildRun.ObjectMeta
	src.TypeMeta = alphaBuildRun.TypeMeta
	src.TypeMeta.APIVersion = betaGroupVersion

	src.Spec.ConvertFrom(&alphaBuildRun.Spec)

	sources := []SourceResult{}
	for _, s := range alphaBuildRun.Status.Sources {
		sr := SourceResult{
			Name:        s.Name,
			Git:         (*GitSourceResult)(s.Git),
			OciArtifact: (*OciArtifactSourceResult)(s.Bundle),
		}
		sources = append(sources, sr)
	}

	conditions := []Condition{}

	for _, c := range alphaBuildRun.Status.Conditions {
		ct := Condition{
			Type:               Type(c.Type),
			Status:             c.Status,
			LastTransitionTime: c.LastTransitionTime,
			Reason:             c.Reason,
			Message:            c.Message,
		}
		conditions = append(conditions, ct)
	}

	src.Status = BuildRunStatus{
		Sources:        sources,
		Output:         (*Output)(alphaBuildRun.Status.Output),
		Conditions:     conditions,
		TaskRunName:    alphaBuildRun.Status.LatestTaskRunRef,
		StartTime:      alphaBuildRun.Status.StartTime,
		CompletionTime: alphaBuildRun.Status.CompletionTime,
		FailureDetails: src.Status.FailureDetails,
	}

	buildBeta := Build{}
	if alphaBuildRun.Status.BuildSpec != nil {
		buildBeta.Spec.ConvertFrom(alphaBuildRun.Status.BuildSpec)
		src.Status.BuildSpec = &buildBeta.Spec
	}

	return nil
}

func (dest *BuildRunSpec) ConvertFrom(orig *v1alpha1.BuildRunSpec) error {

	// BuildRunSpec BuildSpec
	dest.Build = &ReferencedBuild{}
	if orig.BuildSpec != nil {
		if dest.Build.Build != nil {
			dest.Build.Build.ConvertFrom(orig.BuildSpec)
		}
	}
	if orig.BuildRef != nil {
		dest.Build.Name = orig.BuildRef.Name
	}

	if orig.ServiceAccount != nil {
		dest.ServiceAccount = orig.ServiceAccount.Name
	}

	dest.Timeout = orig.Timeout

	// BuildRunSpec ParamValues
	dest.ParamValues = []ParamValue{}
	for _, p := range orig.ParamValues {
		param := convertBetaParamValue(p)
		dest.ParamValues = append(dest.ParamValues, param)
	}

	// Handle BuildSpec Output
	dest.Output = &Image{}
	if orig.Output != nil {
		dest.Output.Image = orig.Output.Image
		dest.Output.Annotations = orig.Output.Annotations
		dest.Output.Labels = orig.Output.Labels
	}

	if orig.Output != nil && orig.Output.Credentials != nil {
		dest.Output.PushSecret = &orig.Output.Credentials.Name
	}

	// BuildRunSpec State
	dest.State = (*BuildRunRequestedState)(orig.State)

	// BuildRunSpec Env
	dest.Env = orig.Env

	// BuildRunSpec Retention
	dest.Retention = (*BuildRunRetention)(orig.Retention)

	// BuildRunSpec Volumes
	dest.Volumes = []BuildVolume{}
	for _, vol := range orig.Volumes {
		dest.Volumes = append(dest.Volumes, BuildVolume{
			Name:         vol.Name,
			VolumeSource: vol.VolumeSource,
		})
	}
	return nil
}
