// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	build "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/ctxlog"
)

// Strategy contains all required fields
// to validate a Build spec strategy definition
type Strategy struct {
	Build  *build.Build
	Client client.Client
}

func NewStrategies(client client.Client, build *build.Build) *Strategy {
	return &Strategy{build, client}
}

// ValidatePath implements BuildPath interface and validates
// that the referenced strategy exists. This applies to both
// namespaced or cluster scoped strategies
func (s Strategy) ValidatePath(ctx context.Context) error {
	switch s.kind(ctx) {
	case build.NamespacedBuildStrategyKind:
		return s.validateBuildStrategy(ctx, s.Build.Spec.Strategy.Name)

	case build.ClusterBuildStrategyKind:
		return s.validateClusterBuildStrategy(ctx, s.Build.Spec.Strategy.Name)

	default:
		s.Build.Status.Reason = ptr.To[build.BuildReason](build.UnknownBuildStrategyKind)
		s.Build.Status.Message = ptr.To(fmt.Sprintf("unknown strategy kind %s used, must be one of %s, or %s",
			*s.Build.Spec.Strategy.Kind,
			build.NamespacedBuildStrategyKind,
			build.ClusterBuildStrategyKind))
		return nil
	}
}

func (s Strategy) kind(ctx context.Context) build.BuildStrategyKind {
	if s.Build.Spec.Strategy.Kind == nil {
		ctxlog.Info(ctx, "buildStrategy kind is nil, use default NamespacedBuildStrategyKind", namespace, s.Build.Namespace, name, s.Build.Name)
		return build.NamespacedBuildStrategyKind
	}

	return *s.Build.Spec.Strategy.Kind
}

func (s Strategy) validateBuildStrategy(ctx context.Context, strategyName string) error {
	buildStrategy := &build.BuildStrategy{}
	err := s.Client.Get(ctx, types.NamespacedName{Name: strategyName, Namespace: s.Build.Namespace}, buildStrategy)
	if err == nil {
		s.validateBuildParams(buildStrategy.GetParameters())
		s.validateBuildVolumes(buildStrategy.GetVolumes())
		return nil
	}

	if apierrors.IsNotFound(err) {
		s.Build.Status.Reason = ptr.To[build.BuildReason](build.BuildStrategyNotFound)
		s.Build.Status.Message = ptr.To(fmt.Sprintf("buildStrategy %s does not exist in namespace %s", strategyName, s.Build.Namespace))
		return nil
	}

	return err
}

func (s Strategy) validateClusterBuildStrategy(ctx context.Context, strategyName string) error {
	clusterBuildStrategy := &build.ClusterBuildStrategy{}
	err := s.Client.Get(ctx, types.NamespacedName{Name: strategyName}, clusterBuildStrategy)
	if err == nil {
		s.validateBuildParams(clusterBuildStrategy.GetParameters())
		s.validateBuildVolumes(clusterBuildStrategy.GetVolumes())
		return nil
	}

	if apierrors.IsNotFound(err) {
		s.Build.Status.Reason = ptr.To[build.BuildReason](build.ClusterBuildStrategyNotFound)
		s.Build.Status.Message = ptr.To(fmt.Sprintf("clusterBuildStrategy %s does not exist", strategyName))
		return nil
	}

	return err
}

func (s Strategy) validateBuildParams(parameterDefinitions []build.Parameter) {
	valid, reason, message := BuildParameters(parameterDefinitions, s.Build.Spec.ParamValues)
	if !valid {
		s.Build.Status.Reason = ptr.To[build.BuildReason](reason)
		s.Build.Status.Message = ptr.To(message)
	}
}

func (s Strategy) validateBuildVolumes(strategyVolumes []build.BuildStrategyVolume) {
	valid, reason, message := BuildVolumes(strategyVolumes, s.Build.Spec.Volumes)
	if !valid {
		s.Build.Status.Reason = ptr.To[build.BuildReason](reason)
		s.Build.Status.Message = ptr.To(message)
	}
}
