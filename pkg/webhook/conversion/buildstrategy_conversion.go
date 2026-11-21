// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package conversion

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

	buildapialpha "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/ctxlog"
)

// convertBuildStrategyBetaToAlpha converts this object to its v1alpha1 equivalent
func convertBuildStrategyBetaToAlpha(ctx context.Context, src *buildapi.BuildStrategy, obj *unstructured.Unstructured) error {
	ctxlog.Info(ctx, "converting BuildStrategy from beta to alpha", "namespace", src.Namespace, "name", src.Name)

	var bs buildapialpha.BuildStrategy
	bs.TypeMeta = src.TypeMeta
	bs.APIVersion = alphaGroupVersion
	bs.ObjectMeta = src.ObjectMeta

	convertBuildStrategySpecBetaToAlpha(&src.Spec, &bs.Spec)

	mapito, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&bs)
	if err != nil {
		ctxlog.Error(ctx, err, "failed structuring the newObject")
		return err
	}
	obj.Object = mapito

	return nil
}

func convertBuildStrategySpecBetaToAlpha(src *buildapi.BuildStrategySpec, bs *buildapialpha.BuildStrategySpec) {
	usesMigratedDockerfileArg, usesMigratedBuilderArg := false, false

	bs.Parameters = []buildapialpha.Parameter{}
	for _, param := range src.Parameters {
		if param.Name == "dockerfile" && param.Type == buildapi.ParameterTypeString && param.Default != nil && *param.Default == "Dockerfile" {
			usesMigratedDockerfileArg = true
			continue
		}

		if param.Name == "builder-image" && param.Type == buildapi.ParameterTypeString && param.Default == nil {
			usesMigratedBuilderArg = true
			continue
		}

		bs.Parameters = append(bs.Parameters, buildapialpha.Parameter{
			Name:        param.Name,
			Description: param.Description,
			Type:        buildapialpha.ParameterType(param.Type),
			Default:     param.Default,
			Defaults:    param.Defaults,
		})
	}

	bs.BuildSteps = []buildapialpha.BuildStep{}
	for _, step := range src.Steps {
		buildStep := buildapialpha.BuildStep{
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
		bs.SecurityContext = (*buildapialpha.BuildStrategySecurityContext)(src.SecurityContext)
	}

	bs.Volumes = []buildapialpha.BuildStrategyVolume{}
	for _, vol := range src.Volumes {
		bs.Volumes = append(bs.Volumes, buildapialpha.BuildStrategyVolume{
			Overridable:  vol.Overridable,
			Name:         vol.Name,
			Description:  vol.Description,
			VolumeSource: vol.VolumeSource,
		})
	}
}

// convertBuildStrategyAlphaToBeta converts from buildapialpha.BuildStrategy into buildapi.BuildStrategy.
func convertBuildStrategyAlphaToBeta(ctx context.Context, src *buildapi.BuildStrategy, obj *unstructured.Unstructured) error {
	var bs buildapialpha.BuildStrategy

	unstructured := obj.UnstructuredContent()
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured, &bs)
	if err != nil {
		ctxlog.Error(ctx, err, "failed unstructuring the buildrun convertedObject")
		return err
	}

	ctxlog.Info(ctx, "converting BuildStrategy from alpha to beta", "namespace", bs.Namespace, "name", bs.Name)

	src.ObjectMeta = bs.ObjectMeta
	src.TypeMeta = bs.TypeMeta
	src.APIVersion = betaGroupVersion

	convertBuildStrategySpecAlphaToBeta(&src.Spec, bs.Spec)

	return nil
}

func convertBuildStrategySpecAlphaToBeta(src *buildapi.BuildStrategySpec, bs buildapialpha.BuildStrategySpec) {
	src.Steps = []buildapi.Step{}

	usesDockerfile, usesBuilderImage := false, false

	for _, brStep := range bs.BuildSteps {
		step := buildapi.Step{
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

	src.Parameters = []buildapi.Parameter{}
	for _, param := range bs.Parameters {
		src.Parameters = append(src.Parameters, buildapi.Parameter{
			Name:        param.Name,
			Description: param.Description,
			Type:        buildapi.ParameterType(param.Type),
			Default:     param.Default,
			Defaults:    param.Defaults,
		})
	}

	// Add replacement for legacy arguments
	if usesDockerfile {
		src.Parameters = append(src.Parameters, buildapi.Parameter{
			Name:        "dockerfile",
			Description: "The Dockerfile to be built.",
			Type:        buildapi.ParameterTypeString,
			Default:     ptr.To("Dockerfile"),
		})
	}

	if usesBuilderImage {
		src.Parameters = append(src.Parameters, buildapi.Parameter{
			Name:        "builder-image",
			Description: "The builder image.",
			Type:        buildapi.ParameterTypeString,
		})
	}

	src.SecurityContext = (*buildapi.BuildStrategySecurityContext)(bs.SecurityContext)

	src.Volumes = []buildapi.BuildStrategyVolume{}
	for _, vol := range bs.Volumes {
		src.Volumes = append(src.Volumes, buildapi.BuildStrategyVolume{
			Overridable:  vol.Overridable,
			Name:         vol.Name,
			Description:  vol.Description,
			VolumeSource: vol.VolumeSource,
		})
	}
}
