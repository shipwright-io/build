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
	"k8s.io/utils/ptr"
)

// ensure v1beta1 implements the Conversion interface
var _ webhook.Conversion = (*BuildRun)(nil)

// To Alpha
func (src *BuildRun) ConvertTo(ctx context.Context, obj *unstructured.Unstructured) error {
	ctxlog.Info(ctx, "converting BuildRun from beta to alpha", "namespace", src.Namespace, "name", src.Name)

	var alphaBuildRun v1alpha1.BuildRun

	alphaBuildRun.TypeMeta = src.TypeMeta
	alphaBuildRun.TypeMeta.APIVersion = alphaGroupVersion
	alphaBuildRun.ObjectMeta = src.ObjectMeta

	// BuildRunSpec BuildSpec
	if src.Spec.Build.Spec != nil {
		newBuildSpec := v1alpha1.BuildSpec{}
		if err := src.Spec.Build.Spec.ConvertTo(&newBuildSpec); err != nil {
			return err
		}
		alphaBuildRun.Spec.BuildSpec = &newBuildSpec
	} else if src.Spec.Build.Name != nil {
		alphaBuildRun.Spec.BuildRef = &v1alpha1.BuildRef{
			Name: *src.Spec.Build.Name,
		}
	}

	// BuildRunSpec Sources
	if src.Spec.Source != nil && src.Spec.Source.Type == LocalType && src.Spec.Source.Local != nil {
		alphaBuildRun.Spec.Sources = append(alphaBuildRun.Spec.Sources, v1alpha1.BuildSource{
			Name:    src.Spec.Source.Local.Name,
			Type:    v1alpha1.LocalCopy,
			Timeout: src.Spec.Source.Local.Timeout,
		})
	}

	// BuildRunSpec ServiceAccount
	// With the deprecation of serviceAccount.Generate, serviceAccount is set to ".generate" to have the SA created on fly.
	if src.Spec.ServiceAccount != nil && *src.Spec.ServiceAccount == ".generate" {
		alphaBuildRun.Spec.ServiceAccount = &v1alpha1.ServiceAccount{
			Name:     &src.ObjectMeta.Name,
			Generate: ptr.To(true),
		}
	} else {
		alphaBuildRun.Spec.ServiceAccount = &v1alpha1.ServiceAccount{
			Name: src.Spec.ServiceAccount,
		}
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

	// BuildRun Status
	var sourceStatus []v1alpha1.SourceResult
	if src.Status.Source != nil && src.Status.Source.Git != nil {
		// Note: v1alpha contains a Name field under the SourceResult
		// object, which we dont set here.
		sourceStatus = append(sourceStatus, v1alpha1.SourceResult{
			Name:      "default",
			Git:       (*v1alpha1.GitSourceResult)(src.Status.Source.Git),
			Timestamp: src.Status.Source.Timestamp,
		})
	}

	if src.Status.Source != nil && src.Status.Source.OciArtifact != nil {
		// Note: v1alpha contains a Name field under the SourceResult
		// object, which we dont set here.
		sourceStatus = append(sourceStatus, v1alpha1.SourceResult{
			Name:      "default",
			Bundle:    (*v1alpha1.BundleSourceResult)(src.Status.Source.OciArtifact),
			Timestamp: src.Status.Source.Timestamp,
		})
	}

	var conditions []v1alpha1.Condition
	for _, c := range src.Status.Conditions {
		ct := v1alpha1.Condition{
			Type:               v1alpha1.Type(c.Type),
			Status:             c.Status,
			LastTransitionTime: c.LastTransitionTime,
			Reason:             c.Reason,
			Message:            c.Message,
		}
		conditions = append(conditions, ct)
	}

	var output *v1alpha1.Output
	if src.Status.Output != nil {
		output = &v1alpha1.Output{
			Digest: src.Status.Output.Digest,
			Size:   src.Status.Output.Size,
		}
	}

	alphaBuildRun.Status = v1alpha1.BuildRunStatus{
		Sources:          sourceStatus,
		Output:           output,
		Conditions:       conditions,
		LatestTaskRunRef: src.Status.TaskRunName,
		StartTime:        src.Status.StartTime,
		CompletionTime:   src.Status.CompletionTime,
	}

	if src.Status.FailureDetails != nil {
		alphaBuildRun.Status.FailureDetails = &v1alpha1.FailureDetails{
			Reason:  src.Status.FailureDetails.Reason,
			Message: src.Status.FailureDetails.Message,
		}
	}

	if src.Status.FailureDetails != nil && src.Status.FailureDetails.Location != nil {
		alphaBuildRun.Status.FailureDetails.Location = &v1alpha1.FailedAt{
			Pod:       src.Status.FailureDetails.Location.Pod,
			Container: src.Status.FailureDetails.Location.Container,
		}
		//nolint:staticcheck // SA1019 we want to give users some time to adopt to failureDetails
		alphaBuildRun.Status.FailedAt = alphaBuildRun.Status.FailureDetails.Location
	}

	aux := &v1alpha1.BuildSpec{}
	if src.Status.BuildSpec != nil {
		src.Status.BuildSpec.ConvertTo(aux)
		alphaBuildRun.Status.BuildSpec = aux
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

	ctxlog.Info(ctx, "converting BuildRun from alpha to beta", "namespace", alphaBuildRun.Namespace, "name", alphaBuildRun.Name)

	src.ObjectMeta = alphaBuildRun.ObjectMeta
	src.TypeMeta = alphaBuildRun.TypeMeta
	src.TypeMeta.APIVersion = betaGroupVersion

	src.Spec.ConvertFrom(&alphaBuildRun.Spec)

	var sourceStatus *SourceResult
	for _, s := range alphaBuildRun.Status.Sources {
		sourceStatus = &SourceResult{
			Git:         (*GitSourceResult)(s.Git),
			OciArtifact: (*OciArtifactSourceResult)(s.Bundle),
			Timestamp:   s.Timestamp,
		}
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

	if alphaBuildRun.Status.FailureDetails != nil {
		src.Status.FailureDetails = &FailureDetails{
			Reason:   alphaBuildRun.Status.FailureDetails.Reason,
			Message:  alphaBuildRun.Status.FailureDetails.Message,
			Location: (*Location)(alphaBuildRun.Status.FailureDetails.Location),
		}
	}

	var output *Output
	if alphaBuildRun.Status.Output != nil {
		output = &Output{
			Digest: alphaBuildRun.Status.Output.Digest,
			Size:   alphaBuildRun.Status.Output.Size,
		}
	}

	src.Status = BuildRunStatus{
		Source:         sourceStatus,
		Output:         output,
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
	if orig.BuildSpec != nil {
		dest.Build.Spec = &BuildSpec{}
		dest.Build.Spec.ConvertFrom(orig.BuildSpec)
	}
	if orig.BuildRef != nil {
		dest.Build.Name = &orig.BuildRef.Name
	}

	// only interested on spec.sources as long as an item of the list
	// is of the type LocalCopy. Otherwise, we move into bundle or git types.
	index, isLocal := v1alpha1.IsLocalCopyType(orig.Sources)
	if isLocal {
		dest.Source = &BuildRunSource{
			Type: LocalType,
			Local: &Local{
				Name:    orig.Sources[index].Name,
				Timeout: orig.Sources[index].Timeout,
			},
		}
	}

	if orig.ServiceAccount != nil {
		dest.ServiceAccount = orig.ServiceAccount.Name
		if orig.ServiceAccount.Generate != nil && *orig.ServiceAccount.Generate {
			dest.ServiceAccount = ptr.To(".generate")
		}
	}

	dest.Timeout = orig.Timeout

	// BuildRunSpec ParamValues
	dest.ParamValues = []ParamValue{}
	for _, p := range orig.ParamValues {
		param := convertBetaParamValue(p)
		dest.ParamValues = append(dest.ParamValues, param)
	}

	// Handle BuildRunSpec Output
	if orig.Output != nil {
		dest.Output = &Image{
			Image:       orig.Output.Image,
			Annotations: orig.Output.Annotations,
			Labels:      orig.Output.Labels,
		}

		if orig.Output.Credentials != nil {
			dest.Output.PushSecret = &orig.Output.Credentials.Name
		}
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
