// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"context"
	"strings"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/ctxlog"
	"github.com/shipwright-io/build/pkg/webhook"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
)

// ensure v1beta1 implements the Conversion interface
var _ webhook.Conversion = (*BuildStrategy)(nil)

// ConvertTo converts this object to its v1alpha1 equivalent
func (src *BuildStrategy) ConvertTo(ctx context.Context, obj *unstructured.Unstructured) error {
	ctxlog.Info(ctx, "converting BuildStrategy from beta to alpha", "namespace", src.Namespace, "name", src.Name)

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
	usesMigratedDockerfileArg, usesMigratedBuilderArg := false, false

	bs.Parameters = []v1alpha1.Parameter{}
	for _, param := range src.Parameters {
		if param.Name == "dockerfile" && param.Type == ParameterTypeString && param.Default != nil && *param.Default == "Dockerfile" {
			usesMigratedDockerfileArg = true
			continue
		}

		if param.Name == "builder-image" && param.Type == ParameterTypeString && param.Default == nil {
			usesMigratedBuilderArg = true
			continue
		}

		bs.Parameters = append(bs.Parameters, v1alpha1.Parameter{
			Name:        param.Name,
			Description: param.Description,
			Type:        v1alpha1.ParameterType(param.Type),
			Default:     param.Default,
			Defaults:    param.Defaults,
		})
	}

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
				SecurityContext: step.SecurityContext,
			},
		}

		if usesMigratedDockerfileArg {
			// Migrate to legacy argument

			for commandIndex, command := range buildStep.Command {
				if strings.Contains(command, "$(params.dockerfile)") {
					buildStep.Command[commandIndex] = strings.ReplaceAll(command, "$(params.dockerfile)", "$(params.DOCKERFILE)")
				}
			}

			for argIndex, arg := range buildStep.Args {
				if strings.Contains(arg, "$(params.dockerfile)") {
					buildStep.Args[argIndex] = strings.ReplaceAll(arg, "$(params.dockerfile)", "$(params.DOCKERFILE)")
				}
			}

			for envIndex, env := range buildStep.Env {
				if strings.Contains(env.Value, "$(params.dockerfile)") {
					buildStep.Env[envIndex].Value = strings.ReplaceAll(env.Value, "$(params.dockerfile)", "$(params.DOCKERFILE)")
				}
			}
		}

		if usesMigratedBuilderArg {
			// Migrate to legacy argument

			for commandIndex, command := range buildStep.Command {
				if strings.Contains(command, "$(params.builder-image)") {
					buildStep.Command[commandIndex] = strings.ReplaceAll(command, "$(params.builder-image)", "$(build.builder.image)")
				}
			}

			for argIndex, arg := range buildStep.Args {
				if strings.Contains(arg, "$(params.builder-image)") {
					buildStep.Args[argIndex] = strings.ReplaceAll(arg, "$(params.builder-image)", "$(build.builder.image)")
				}
			}

			for envIndex, env := range buildStep.Env {
				if strings.Contains(env.Value, "$(params.builder-image)") {
					buildStep.Env[envIndex].Value = strings.ReplaceAll(env.Value, "$(params.builder-image)", "$(build.builder.image)")
				}
			}
		}

		bs.BuildSteps = append(bs.BuildSteps, buildStep)
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
	var bs v1alpha1.BuildStrategy

	unstructured := obj.UnstructuredContent()
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured, &bs)
	if err != nil {
		ctxlog.Error(ctx, err, "failed unstructuring the buildrun convertedObject")
	}

	ctxlog.Info(ctx, "converting BuildStrategy from alpha to beta", "namespace", bs.Namespace, "name", bs.Name)

	src.ObjectMeta = bs.ObjectMeta
	src.TypeMeta = bs.TypeMeta
	src.TypeMeta.APIVersion = betaGroupVersion

	src.Spec.ConvertFrom(bs.Spec)

	return nil
}

func (src *BuildStrategySpec) ConvertFrom(bs v1alpha1.BuildStrategySpec) {
	src.Steps = []Step{}

	usesDockerfile, usesBuilderImage := false, false

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

		// Migrate legacy argument usage

		for commandIndex, command := range step.Command {
			if strings.Contains(command, "$(params.DOCKERFILE)") {
				usesDockerfile = true
				step.Command[commandIndex] = strings.ReplaceAll(command, "$(params.DOCKERFILE)", "$(params.dockerfile)")
			}
			if strings.Contains(command, "$(build.dockerfile)") {
				usesDockerfile = true
				step.Command[commandIndex] = strings.ReplaceAll(command, "$(build.dockerfile)", "$(params.dockerfile)")
			}
			if strings.Contains(command, "$(build.builder.image)") {
				usesBuilderImage = true
				step.Command[commandIndex] = strings.ReplaceAll(command, "$(build.builder.image)", "$(params.builder-image)")
			}
		}

		for argIndex, arg := range step.Args {
			if strings.Contains(arg, "$(params.DOCKERFILE)") {
				usesDockerfile = true
				step.Args[argIndex] = strings.ReplaceAll(arg, "$(params.DOCKERFILE)", "$(params.dockerfile)")
			}
			if strings.Contains(arg, "$(build.dockerfile)") {
				usesDockerfile = true
				step.Args[argIndex] = strings.ReplaceAll(arg, "$(build.dockerfile)", "$(params.dockerfile)")
			}
			if strings.Contains(arg, "$(build.builder.image)") {
				usesBuilderImage = true
				step.Args[argIndex] = strings.ReplaceAll(arg, "$(build.builder.image)", "$(params.builder-image)")
			}
		}

		for envIndex, env := range step.Env {
			if strings.Contains(env.Value, "$(params.DOCKERFILE)") {
				usesDockerfile = true
				step.Env[envIndex].Value = strings.ReplaceAll(env.Value, "$(params.DOCKERFILE)", "$(params.dockerfile)")
			}
			if strings.Contains(env.Value, "$(build.dockerfile)") {
				usesDockerfile = true
				step.Env[envIndex].Value = strings.ReplaceAll(env.Value, "$(build.dockerfile)", "$(params.dockerfile)")
			}
			if strings.Contains(env.Value, "$(build.builder.image)") {
				usesBuilderImage = true
				step.Env[envIndex].Value = strings.ReplaceAll(env.Value, "$(build.builder.image)", "$(params.builder-image)")
			}
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

	// Add replacement for legacy arguments

	if usesDockerfile {
		src.Parameters = append(src.Parameters, Parameter{
			Name:        "dockerfile",
			Description: "The Dockerfile to be built.",
			Type:        ParameterTypeString,
			Default:     pointer.String("Dockerfile"),
		})
	}

	if usesBuilderImage {
		src.Parameters = append(src.Parameters, Parameter{
			Name:        "builder-image",
			Description: "The builder image.",
			Type:        ParameterTypeString,
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
