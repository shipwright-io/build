// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package conversion

import (
	"context"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

	buildapialpha "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/ctxlog"
)

// convertBuildBetaToAlpha converts this Build object to v1alpha1 format.
func convertBuildBetaToAlpha(ctx context.Context, src *buildapi.Build, obj *unstructured.Unstructured) error {
	ctxlog.Info(ctx, "converting Build from beta to alpha", "namespace", src.Namespace, "name", src.Name)

	var alphaBuild buildapialpha.Build

	alphaBuild.TypeMeta = src.TypeMeta
	alphaBuild.APIVersion = alphaGroupVersion

	alphaBuild.ObjectMeta = src.ObjectMeta

	if err := convertBuildSpecBetaToAlpha(&src.Spec, &alphaBuild.Spec); err != nil {
		ctxlog.Error(ctx, err, "failed to convert object")
		return err
	}

	alphaBuild.Status = buildapialpha.BuildStatus{
		Registered: src.Status.Registered,
		Reason:     (*buildapialpha.BuildReason)(src.Status.Reason),
		Message:    src.Status.Message,
	}

	// convert annotation-controlled features
	if src.Spec.Retention != nil && src.Spec.Retention.AtBuildDeletion != nil {
		// We must create a new Map as otherwise the addition is not kept
		alphaBuild.Annotations = map[string]string{}
		for k, v := range src.Annotations {
			alphaBuild.Annotations[k] = v
		}
		alphaBuild.Annotations[buildapialpha.AnnotationBuildRunDeletion] = strconv.FormatBool(*src.Spec.Retention.AtBuildDeletion)
	}

	mapito, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&alphaBuild)
	if err != nil {
		ctxlog.Error(ctx, err, "failed structuring the newObject")
		return err
	}
	obj.Object = mapito

	return nil
}

// convertBuildAlphaToBeta converts a provided buildapialpha.Build object into this v1beta1.Build object.
func convertBuildAlphaToBeta(ctx context.Context, src *buildapi.Build, obj *unstructured.Unstructured) error {
	var alphaBuild buildapialpha.Build

	unstructured := obj.UnstructuredContent()
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured, &alphaBuild)
	if err != nil {
		ctxlog.Error(ctx, err, "failed unstructuring the convertedObject")
		return err
	}

	ctxlog.Info(ctx, "converting Build from alpha to beta", "namespace", alphaBuild.Namespace, "name", alphaBuild.Name)

	src.ObjectMeta = alphaBuild.ObjectMeta
	src.TypeMeta = alphaBuild.TypeMeta
	src.APIVersion = betaGroupVersion

	if err := convertBuildSpecAlphaToBeta(&src.Spec, &alphaBuild.Spec); err != nil {
		ctxlog.Error(ctx, err, "failed to convert object")
		return err
	}

	// convert annotation-controlled features
	if value, set := alphaBuild.Annotations[buildapialpha.AnnotationBuildRunDeletion]; set {
		if src.Spec.Retention == nil {
			src.Spec.Retention = &buildapi.BuildRetention{}
		}
		src.Spec.Retention.AtBuildDeletion = ptr.To(value == "true")
		delete(src.Annotations, buildapialpha.AnnotationBuildRunDeletion)
	}

	src.Status = buildapi.BuildStatus{
		Registered: alphaBuild.Status.Registered,
		Reason:     (*buildapi.BuildReason)(alphaBuild.Status.Reason),
		Message:    alphaBuild.Status.Message,
	}

	return nil
}

func convertBuildSpecAlphaToBeta(dest *buildapi.BuildSpec, orig *buildapialpha.BuildSpec) error {
	// Handle BuildSpec Source

	// only interested on spec.sources as long as an item of the list
	// is of the type LocalCopy. Otherwise, we move into bundle or git types.
	index, isLocal := buildapialpha.IsLocalCopyType(orig.Sources)
	if isLocal {
		dest.Source = &buildapi.Source{
			Type: buildapi.LocalType,
			Local: &buildapi.Local{
				Name:    orig.Sources[index].Name,
				Timeout: orig.Sources[index].Timeout,
			},
			ContextDir: orig.Source.ContextDir,
		}
	} else if orig.Source.BundleContainer != nil {
		dest.Source = &buildapi.Source{
			Type: buildapi.OCIArtifactType,
			OCIArtifact: &buildapi.OCIArtifact{
				Image: orig.Source.BundleContainer.Image,
				Prune: (*buildapi.PruneOption)(orig.Source.BundleContainer.Prune),
			},
			ContextDir: orig.Source.ContextDir,
		}
		if orig.Source.Credentials != nil {
			dest.Source.OCIArtifact.PullSecret = &orig.Source.Credentials.Name
		}
	} else if orig.Source.URL != nil {
		dest.Source = &buildapi.Source{
			Type: buildapi.GitType,
			Git: &buildapi.Git{
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
		dest.Trigger = &buildapi.Trigger{}
		for i := range orig.Trigger.When {
			dest.Trigger.When = append(dest.Trigger.When, convertToBetaTriggers(&orig.Trigger.When[i]))
		}
		if orig.Trigger.SecretRef != nil {
			dest.Trigger.TriggerSecret = &orig.Trigger.SecretRef.Name
		}
	}

	// Handle BuildSpec Strategy
	dest.Strategy = buildapi.Strategy{
		Name: orig.StrategyName(),
		Kind: (*buildapi.BuildStrategyKind)(orig.Strategy.Kind),
	}

	// Handle BuildSpec ParamValues
	for _, p := range orig.ParamValues {
		param := convertBetaParamValue(p)
		dest.ParamValues = append(dest.ParamValues, param)
	}

	//handle spec.Dockerfile migration
	if orig.Dockerfile != nil && *orig.Dockerfile != "" {
		dockerfileParam := buildapi.ParamValue{
			Name: "dockerfile",
			SingleValue: &buildapi.SingleValue{
				Value: orig.Dockerfile,
			},
		}
		dest.ParamValues = append(dest.ParamValues, dockerfileParam)
	}

	// handle spec.Builder migration
	if orig.Builder != nil {
		builderParam := buildapi.ParamValue{
			Name: "builder-image",
			SingleValue: &buildapi.SingleValue{
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
		dest.Retention = &buildapi.BuildRetention{
			FailedLimit:       orig.Retention.FailedLimit,
			SucceededLimit:    orig.Retention.SucceededLimit,
			TTLAfterFailed:    orig.Retention.TTLAfterFailed,
			TTLAfterSucceeded: orig.Retention.TTLAfterSucceeded,
		}
	}

	// Handle BuildSpec Volumes
	dest.Volumes = []buildapi.BuildVolume{}
	for _, vol := range orig.Volumes {
		aux := buildapi.BuildVolume{
			Name:         vol.Name,
			VolumeSource: vol.VolumeSource,
		}
		dest.Volumes = append(dest.Volumes, aux)
	}

	return nil
}

func convertBuildSpecBetaToAlpha(dest *buildapi.BuildSpec, bs *buildapialpha.BuildSpec) error {
	// Handle BuildSpec Sources or Source
	if dest.Source != nil && dest.Source.Type == buildapi.LocalType && dest.Source.Local != nil {
		bs.Sources = append(bs.Sources, buildapialpha.BuildSource{
			Name:    dest.Source.Local.Name,
			Type:    buildapialpha.LocalCopy,
			Timeout: dest.Source.Local.Timeout,
		})
	} else {
		bs.Source = getAlphaBuildSource(*dest)
	}

	// Handle BuildSpec Trigger
	if dest.Trigger != nil {
		bs.Trigger = &buildapialpha.Trigger{}
		for _, t := range dest.Trigger.When {
			tw := buildapialpha.TriggerWhen{}
			convertTriggerWhenBetaToAlpha(t, &tw)
			bs.Trigger.When = append(bs.Trigger.When, tw)
		}
		if dest.Trigger.TriggerSecret != nil {
			bs.Trigger.SecretRef = &corev1.LocalObjectReference{Name: *dest.Trigger.TriggerSecret}
		}
	}

	// Handle BuildSpec Strategy
	bs.Strategy = buildapialpha.Strategy{
		Name: dest.StrategyName(),
		Kind: (*buildapialpha.BuildStrategyKind)(dest.Strategy.Kind),
	}

	// Handle BuildSpec Builder, TODO
	bs.Builder = nil

	// Handle BuildSpec ParamValues
	bs.ParamValues = nil
	for _, p := range dest.ParamValues {
		if p.Name == "dockerfile" && p.SingleValue != nil {
			bs.Dockerfile = p.Value
			continue
		}

		if p.Name == "builder-image" && p.SingleValue != nil {
			bs.Builder = &buildapialpha.Image{
				Image: *p.Value,
			}
			continue
		}
		param := buildapialpha.ParamValue{}
		convertParamValueBetaToAlpha(p, &param)
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
		bs.Retention = &buildapialpha.BuildRetention{
			FailedLimit:       dest.Retention.FailedLimit,
			SucceededLimit:    dest.Retention.SucceededLimit,
			TTLAfterFailed:    dest.Retention.TTLAfterFailed,
			TTLAfterSucceeded: dest.Retention.TTLAfterSucceeded,
		}
	}

	// Handle BuildSpec Volumes
	bs.Volumes = []buildapialpha.BuildVolume{}
	for _, vol := range dest.Volumes {
		aux := buildapialpha.BuildVolume{
			Name:         vol.Name,
			VolumeSource: vol.VolumeSource,
		}
		bs.Volumes = append(bs.Volumes, aux)
	}

	return nil
}

func convertParamValueBetaToAlpha(p buildapi.ParamValue, dest *buildapialpha.ParamValue) {
	if p.SingleValue != nil {
		if p.Value != nil {
			dest.SingleValue = &buildapialpha.SingleValue{}
			dest.Value = p.Value
		}

		if p.ConfigMapValue != nil {
			dest.SingleValue = &buildapialpha.SingleValue{
				ConfigMapValue: (*buildapialpha.ObjectKeyRef)(p.ConfigMapValue),
			}
		}

		if p.SecretValue != nil {
			dest.SingleValue = &buildapialpha.SingleValue{
				SecretValue: (*buildapialpha.ObjectKeyRef)(p.SecretValue),
			}
		}
	}

	dest.Name = p.Name

	for _, singleValue := range p.Values {
		dest.Values = append(dest.Values, buildapialpha.SingleValue{
			Value:          singleValue.Value,
			ConfigMapValue: (*buildapialpha.ObjectKeyRef)(singleValue.ConfigMapValue),
			SecretValue:    (*buildapialpha.ObjectKeyRef)(singleValue.SecretValue),
		})
	}
}

func convertTriggerWhenBetaToAlpha(p buildapi.TriggerWhen, dest *buildapialpha.TriggerWhen) {
	dest.Name = p.Name
	dest.Type = buildapialpha.TriggerType(p.Type)

	if p.GitHub != nil {
		dest.GitHub = &buildapialpha.WhenGitHub{}
		for _, e := range p.GitHub.Events {
			dest.GitHub.Events = append(dest.GitHub.Events, buildapialpha.GitHubEventName(e))
		}
		dest.GitHub.Branches = p.GetBranches(buildapi.GitHubWebHookTrigger)
	}

	dest.Image = (*buildapialpha.WhenImage)(p.Image)
	dest.ObjectRef = (*buildapialpha.WhenObjectRef)(p.ObjectRef)
}

func convertBetaParamValue(orig buildapialpha.ParamValue) buildapi.ParamValue {
	p := buildapi.ParamValue{}
	if orig.SingleValue != nil {
		if orig.Value != nil {
			p.SingleValue = &buildapi.SingleValue{}
			p.Value = orig.Value
		}

		if orig.ConfigMapValue != nil {
			p.SingleValue = &buildapi.SingleValue{}
			p.ConfigMapValue = (*buildapi.ObjectKeyRef)(orig.ConfigMapValue)
		}
		if orig.SecretValue != nil {
			p.SingleValue = &buildapi.SingleValue{}
			p.SecretValue = (*buildapi.ObjectKeyRef)(orig.SecretValue)
		}
	}

	p.Name = orig.Name

	for _, singleValue := range orig.Values {
		p.Values = append(p.Values, buildapi.SingleValue{
			Value:          singleValue.Value,
			ConfigMapValue: (*buildapi.ObjectKeyRef)(singleValue.ConfigMapValue),
			SecretValue:    (*buildapi.ObjectKeyRef)(singleValue.SecretValue),
		})
	}
	return p
}

func convertToBetaTriggers(orig *buildapialpha.TriggerWhen) buildapi.TriggerWhen {
	dest := buildapi.TriggerWhen{
		Name: orig.Name,
		Type: buildapi.TriggerType(orig.Type),
	}

	if orig.GitHub != nil {
		dest.GitHub = &buildapi.WhenGitHub{}
		for _, e := range orig.GitHub.Events {
			dest.GitHub.Events = append(dest.GitHub.Events, buildapi.GitHubEventName(e))
		}

		dest.GitHub.Branches = orig.GetBranches(buildapialpha.GitHubWebHookTrigger)
	}
	dest.Image = (*buildapi.WhenImage)(orig.Image)
	dest.ObjectRef = (*buildapi.WhenObjectRef)(orig.ObjectRef)

	return dest
}

func getAlphaBuildSource(src buildapi.BuildSpec) buildapialpha.Source {
	source := buildapialpha.Source{}

	if src.Source == nil {
		return source
	}

	var credentials corev1.LocalObjectReference
	var revision *string

	switch src.Source.Type {
	case buildapi.OCIArtifactType:
		if src.Source.OCIArtifact != nil {
			if src.Source.OCIArtifact.PullSecret != nil {
				credentials = corev1.LocalObjectReference{
					Name: *src.Source.OCIArtifact.PullSecret,
				}
			}
			source.BundleContainer = &buildapialpha.BundleContainer{
				Image: src.Source.OCIArtifact.Image,
				Prune: (*buildapialpha.PruneOption)(src.Source.OCIArtifact.Prune),
			}
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
