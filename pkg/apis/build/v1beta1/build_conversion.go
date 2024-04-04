// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"context"
	"strconv"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/ctxlog"
	"github.com/shipwright-io/build/pkg/webhook"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
)

const (
	betaGroupVersion  = "shipwright.io/v1beta1"
	alphaGroupVersion = "shipwright.io/v1alpha1"
)

// ensure v1beta1 implements the Conversion interface
var _ webhook.Conversion = (*Build)(nil)

// ConvertTo converts this Build object to v1alpha1 format.
func (src *Build) ConvertTo(ctx context.Context, obj *unstructured.Unstructured) error {
	ctxlog.Info(ctx, "converting Build from beta to alpha", "namespace", src.Namespace, "name", src.Name)

	var alphaBuild v1alpha1.Build

	alphaBuild.TypeMeta = src.TypeMeta
	alphaBuild.TypeMeta.APIVersion = alphaGroupVersion

	alphaBuild.ObjectMeta = src.ObjectMeta

	src.Spec.ConvertTo(&alphaBuild.Spec)

	alphaBuild.Status = v1alpha1.BuildStatus{
		Registered: src.Status.Registered,
		Reason:     (*v1alpha1.BuildReason)(src.Status.Reason),
		Message:    src.Status.Message,
	}

	// convert annotation-controlled features
	if src.Spec.Retention != nil && src.Spec.Retention.AtBuildDeletion != nil {
		// We must create a new Map as otherwise the addition is not kept
		alphaBuild.ObjectMeta.Annotations = map[string]string{}
		for k, v := range src.Annotations {
			alphaBuild.ObjectMeta.Annotations[k] = v
		}
		alphaBuild.ObjectMeta.Annotations[v1alpha1.AnnotationBuildRunDeletion] = strconv.FormatBool(*src.Spec.Retention.AtBuildDeletion)
	}

	mapito, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&alphaBuild)
	if err != nil {
		ctxlog.Error(ctx, err, "failed structuring the newObject")
	}
	obj.Object = mapito

	return nil

}

// ConvertFrom converts a provided v1alpha1.Build object into this v1beta1.Build object.
func (src *Build) ConvertFrom(ctx context.Context, obj *unstructured.Unstructured) error {

	var alphaBuild v1alpha1.Build

	unstructured := obj.UnstructuredContent()
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured, &alphaBuild)
	if err != nil {
		ctxlog.Error(ctx, err, "failed unstructuring the convertedObject")
	}

	ctxlog.Info(ctx, "converting Build from alpha to beta", "namespace", alphaBuild.Namespace, "name", alphaBuild.Name)

	src.ObjectMeta = alphaBuild.ObjectMeta
	src.TypeMeta = alphaBuild.TypeMeta
	src.TypeMeta.APIVersion = betaGroupVersion

	src.Spec.ConvertFrom(&alphaBuild.Spec)

	// convert annotation-controlled features
	if value, set := alphaBuild.Annotations[v1alpha1.AnnotationBuildRunDeletion]; set {
		if src.Spec.Retention == nil {
			src.Spec.Retention = &BuildRetention{}
		}
		src.Spec.Retention.AtBuildDeletion = pointer.Bool(value == "true")
		delete(src.ObjectMeta.Annotations, v1alpha1.AnnotationBuildRunDeletion)
	}

	src.Status = BuildStatus{
		Registered: alphaBuild.Status.Registered,
		Reason:     (*BuildReason)(alphaBuild.Status.Reason),
		Message:    alphaBuild.Status.Message,
	}

	return nil
}

func (dest *BuildSpec) ConvertFrom(orig *v1alpha1.BuildSpec) error {
	// Handle BuildSpec Source

	// only interested on spec.sources as long as an item of the list
	// is of the type LocalCopy. Otherwise, we move into bundle or git types.
	index, isLocal := v1alpha1.IsLocalCopyType(orig.Sources)
	if isLocal {
		dest.Source = &Source{
			Type: LocalType,
			Local: &Local{
				Name:    orig.Sources[index].Name,
				Timeout: orig.Sources[index].Timeout,
			},
			ContextDir: orig.Source.ContextDir,
		}
	} else if orig.Source.BundleContainer != nil {
		dest.Source = &Source{
			Type: OCIArtifactType,
			OCIArtifact: &OCIArtifact{
				Image: orig.Source.BundleContainer.Image,
				Prune: (*PruneOption)(orig.Source.BundleContainer.Prune),
			},
			ContextDir: orig.Source.ContextDir,
		}
		if orig.Source.Credentials != nil {
			dest.Source.OCIArtifact.PullSecret = &orig.Source.Credentials.Name
		}
	} else if orig.Source.URL != nil {
		dest.Source = &Source{
			Type: GitType,
			Git: &Git{
				URL:      *orig.Source.URL,
				Revision: orig.Source.Revision,
			},
			ContextDir: orig.Source.ContextDir,
		}
		if orig.Source.Credentials != nil {
			dest.Source.Git.CloneSecret = &orig.Source.Credentials.Name
		}
	}

	// Handle BuildSpec Triggers
	if orig.Trigger != nil {
		dest.Trigger = &Trigger{}
		for i := range orig.Trigger.When {
			dest.Trigger.When = append(dest.Trigger.When, convertToBetaTriggers(&orig.Trigger.When[i]))
		}
		if orig.Trigger.SecretRef != nil {
			dest.Trigger.TriggerSecret = &orig.Trigger.SecretRef.Name
		}
	}

	// Handle BuildSpec Strategy
	dest.Strategy = Strategy{
		Name: orig.StrategyName(),
		Kind: (*BuildStrategyKind)(orig.Strategy.Kind),
	}

	// Handle BuildSpec ParamValues
	for _, p := range orig.ParamValues {
		param := convertBetaParamValue(p)
		dest.ParamValues = append(dest.ParamValues, param)
	}

	//handle spec.Dockerfile migration
	if orig.Dockerfile != nil && *orig.Dockerfile != "" {
		dockerfileParam := ParamValue{
			Name: "dockerfile",
			SingleValue: &SingleValue{
				Value: orig.Dockerfile,
			},
		}
		dest.ParamValues = append(dest.ParamValues, dockerfileParam)
	}

	// handle spec.Builder migration
	if orig.Builder != nil {
		builderParam := ParamValue{
			Name: "builder-image",
			SingleValue: &SingleValue{
				Value: &orig.Builder.Image,
			},
		}
		dest.ParamValues = append(dest.ParamValues, builderParam)
	}

	// Handle BuildSpec Output
	dest.Output.Image = orig.Output.Image
	dest.Output.Insecure = orig.Output.Insecure
	if orig.Output.Credentials != nil {
		dest.Output.PushSecret = &orig.Output.Credentials.Name
	}

	dest.Output.Annotations = orig.Output.Annotations
	dest.Output.Labels = orig.Output.Labels
	dest.Output.Timestamp = orig.Output.Timestamp

	// Handle BuildSpec Timeout
	dest.Timeout = orig.Timeout

	// Handle BuildSpec Env
	dest.Env = orig.Env

	// Handle BuildSpec Retention
	if orig.Retention != nil {
		dest.Retention = &BuildRetention{
			FailedLimit:       orig.Retention.FailedLimit,
			SucceededLimit:    orig.Retention.SucceededLimit,
			TTLAfterFailed:    orig.Retention.TTLAfterFailed,
			TTLAfterSucceeded: orig.Retention.TTLAfterSucceeded,
		}
	}

	// Handle BuildSpec Volumes
	dest.Volumes = []BuildVolume{}
	for _, vol := range orig.Volumes {
		aux := BuildVolume{
			Name:         vol.Name,
			VolumeSource: vol.VolumeSource,
		}
		dest.Volumes = append(dest.Volumes, aux)
	}

	return nil
}

func (dest *BuildSpec) ConvertTo(bs *v1alpha1.BuildSpec) error {
	// Handle BuildSpec Sources or Source
	if dest.Source != nil && dest.Source.Type == LocalType && dest.Source.Local != nil {
		bs.Sources = append(bs.Sources, v1alpha1.BuildSource{
			Name:    dest.Source.Local.Name,
			Type:    v1alpha1.LocalCopy,
			Timeout: dest.Source.Local.Timeout,
		})
	} else {
		bs.Source = getAlphaBuildSource(*dest)
	}

	// Handle BuildSpec Trigger
	if dest.Trigger != nil {
		bs.Trigger = &v1alpha1.Trigger{}
		for _, t := range dest.Trigger.When {
			tw := v1alpha1.TriggerWhen{}
			t.convertToAlpha(&tw)
			bs.Trigger.When = append(bs.Trigger.When, tw)
		}
		if dest.Trigger.TriggerSecret != nil {
			bs.Trigger.SecretRef = &corev1.LocalObjectReference{Name: *dest.Trigger.TriggerSecret}
		}
	}

	// Handle BuildSpec Strategy
	bs.Strategy = v1alpha1.Strategy{
		Name: dest.StrategyName(),
		Kind: (*v1alpha1.BuildStrategyKind)(dest.Strategy.Kind),
	}

	// Handle BuildSpec Builder, TODO
	bs.Builder = nil

	// Handle BuildSpec ParamValues
	bs.ParamValues = nil
	for _, p := range dest.ParamValues {
		if p.Name == "dockerfile" && p.SingleValue != nil {
			bs.Dockerfile = p.SingleValue.Value
			continue
		}

		if p.Name == "builder-image" && p.SingleValue != nil {
			bs.Builder = &v1alpha1.Image{
				Image: *p.SingleValue.Value,
			}
			continue
		}
		param := v1alpha1.ParamValue{}
		p.convertToAlpha(&param)
		bs.ParamValues = append(bs.ParamValues, param)
	}

	// Handle BuildSpec Output
	bs.Output.Image = dest.Output.Image
	bs.Output.Insecure = dest.Output.Insecure
	if dest.Output.PushSecret != nil {
		bs.Output.Credentials = &corev1.LocalObjectReference{}
		bs.Output.Credentials.Name = *dest.Output.PushSecret
	}
	bs.Output.Annotations = dest.Output.Annotations
	bs.Output.Labels = dest.Output.Labels
	bs.Output.Timestamp = dest.Output.Timestamp

	// Handle BuildSpec Timeout
	bs.Timeout = dest.Timeout

	// Handle BuildSpec Env
	bs.Env = dest.Env

	// Handle BuildSpec Retention
	if dest.Retention != nil &&
		(dest.Retention.FailedLimit != nil ||
			dest.Retention.SucceededLimit != nil ||
			dest.Retention.TTLAfterFailed != nil ||
			dest.Retention.TTLAfterSucceeded != nil) {
		bs.Retention = &v1alpha1.BuildRetention{
			FailedLimit:       dest.Retention.FailedLimit,
			SucceededLimit:    dest.Retention.SucceededLimit,
			TTLAfterFailed:    dest.Retention.TTLAfterFailed,
			TTLAfterSucceeded: dest.Retention.TTLAfterSucceeded,
		}
	}

	// Handle BuildSpec Volumes
	bs.Volumes = []v1alpha1.BuildVolume{}
	for _, vol := range dest.Volumes {
		aux := v1alpha1.BuildVolume{
			Name:         vol.Name,
			VolumeSource: vol.VolumeSource,
		}
		bs.Volumes = append(bs.Volumes, aux)
	}

	return nil
}

func (p ParamValue) convertToAlpha(dest *v1alpha1.ParamValue) {

	if p.SingleValue != nil && p.SingleValue.Value != nil {
		dest.SingleValue = &v1alpha1.SingleValue{}
		dest.Value = p.Value
	}

	if p.ConfigMapValue != nil {
		dest.SingleValue = &v1alpha1.SingleValue{
			ConfigMapValue: (*v1alpha1.ObjectKeyRef)(p.ConfigMapValue),
		}
	}

	if p.SecretValue != nil {
		dest.SingleValue = &v1alpha1.SingleValue{
			SecretValue: (*v1alpha1.ObjectKeyRef)(p.SecretValue),
		}
	}

	dest.Name = p.Name

	for _, singleValue := range p.Values {
		dest.Values = append(dest.Values, v1alpha1.SingleValue{
			Value:          singleValue.Value,
			ConfigMapValue: (*v1alpha1.ObjectKeyRef)(singleValue.ConfigMapValue),
			SecretValue:    (*v1alpha1.ObjectKeyRef)(singleValue.SecretValue),
		})
	}
}

func (p TriggerWhen) convertToAlpha(dest *v1alpha1.TriggerWhen) {
	dest.Name = p.Name
	dest.Type = v1alpha1.TriggerType(p.Type)

	dest.GitHub = &v1alpha1.WhenGitHub{}
	for _, e := range p.GitHub.Events {
		dest.GitHub.Events = append(dest.GitHub.Events, v1alpha1.GitHubEventName(e))
	}
	dest.GitHub.Branches = p.GetBranches(GitHubWebHookTrigger)

	dest.Image = (*v1alpha1.WhenImage)(p.Image)
	dest.ObjectRef = (*v1alpha1.WhenObjectRef)(p.ObjectRef)

}

func convertBetaParamValue(orig v1alpha1.ParamValue) ParamValue {
	p := ParamValue{}
	if orig.SingleValue != nil && orig.SingleValue.Value != nil {
		p.SingleValue = &SingleValue{}
		p.Value = orig.Value
	}

	if orig.ConfigMapValue != nil {
		p.SingleValue = &SingleValue{}
		p.ConfigMapValue = (*ObjectKeyRef)(orig.ConfigMapValue)
	}
	if orig.SecretValue != nil {
		p.SingleValue = &SingleValue{}
		p.SecretValue = (*ObjectKeyRef)(orig.SecretValue)
	}

	p.Name = orig.Name

	for _, singleValue := range orig.Values {
		p.Values = append(p.Values, SingleValue{
			Value:          singleValue.Value,
			ConfigMapValue: (*ObjectKeyRef)(singleValue.ConfigMapValue),
			SecretValue:    (*ObjectKeyRef)(singleValue.SecretValue),
		})
	}
	return p
}

func convertToBetaTriggers(orig *v1alpha1.TriggerWhen) TriggerWhen {
	dest := TriggerWhen{
		Name: orig.Name,
		Type: TriggerType(orig.Type),
	}

	dest.GitHub = &WhenGitHub{}
	for _, e := range orig.GitHub.Events {
		dest.GitHub.Events = append(dest.GitHub.Events, GitHubEventName(e))
	}

	dest.GitHub.Branches = orig.GetBranches(v1alpha1.GitHubWebHookTrigger)
	dest.Image = (*WhenImage)(orig.Image)
	dest.ObjectRef = (*WhenObjectRef)(orig.ObjectRef)

	return dest
}

func getAlphaBuildSource(src BuildSpec) v1alpha1.Source {
	source := v1alpha1.Source{}

	if src.Source == nil {
		return source
	}

	var credentials corev1.LocalObjectReference
	var revision *string

	switch src.Source.Type {
	case OCIArtifactType:
		if src.Source.OCIArtifact != nil && src.Source.OCIArtifact.PullSecret != nil {
			credentials = corev1.LocalObjectReference{
				Name: *src.Source.OCIArtifact.PullSecret,
			}
		}
		source.BundleContainer = &v1alpha1.BundleContainer{
			Image: src.Source.OCIArtifact.Image,
			Prune: (*v1alpha1.PruneOption)(src.Source.OCIArtifact.Prune),
		}
	default:
		if src.Source.Git != nil && src.Source.Git.CloneSecret != nil {
			credentials = corev1.LocalObjectReference{
				Name: *src.Source.Git.CloneSecret,
			}
		}
		if src.Source.Git != nil {
			source.URL = &src.Source.Git.URL
			revision = src.Source.Git.Revision
		}

	}

	if credentials.Name != "" {
		source.Credentials = &credentials
	}

	source.Revision = revision
	source.ContextDir = src.Source.ContextDir

	return source
}
