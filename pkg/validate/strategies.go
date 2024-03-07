// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/ctxlog"
)

// Strategy contains all required fields
// to validate a Build spec strategy definition
type Strategy struct {
	Build  *buildapi.Build
	Client client.Client
}

func NewStrategies(client client.Client, build *buildapi.Build) *Strategy {
	return &Strategy{build, client}
}

// ValidatePath implements BuildPath interface and validates
// that the referenced strategy exists. This applies to both
// namespaced or cluster scoped strategies
func (s Strategy) ValidatePath(ctx context.Context) error {
	switch s.kind(ctx) {
	case buildapi.NamespacedBuildStrategyKind:
		return s.validateBuildStrategy(ctx, s.Build.Spec.Strategy.Name)

	case buildapi.ClusterBuildStrategyKind:
		return s.validateClusterBuildStrategy(ctx, s.Build.Spec.Strategy.Name)

	default:
		s.Build.Status.Reason = buildapi.BuildReasonPtr(buildapi.UnknownBuildStrategyKind)
		s.Build.Status.Message = pointer.String(fmt.Sprintf("unknown strategy kind %s used, must be one of %s, or %s",
			*s.Build.Spec.Strategy.Kind,
			buildapi.NamespacedBuildStrategyKind,
			buildapi.ClusterBuildStrategyKind))
		return nil
	}
}

func (s Strategy) kind(ctx context.Context) buildapi.BuildStrategyKind {
	if s.Build.Spec.Strategy.Kind == nil {
		ctxlog.Info(ctx, "buildStrategy kind is nil, use default NamespacedBuildStrategyKind", namespace, s.Build.Namespace, name, s.Build.Name)
		return buildapi.NamespacedBuildStrategyKind
	}

	return *s.Build.Spec.Strategy.Kind
}

func (s Strategy) validateBuildStrategy(ctx context.Context, strategyName string) error {
	buildStrategy := &buildapi.BuildStrategy{}
	err := s.Client.Get(ctx, types.NamespacedName{Name: strategyName, Namespace: s.Build.Namespace}, buildStrategy)
	if err == nil {
		s.validateBuildParams(buildStrategy.GetParameters())
		s.validateBuildVolumes(buildStrategy.GetVolumes())
		return nil
	}

	if apierrors.IsNotFound(err) {
		s.Build.Status.Reason = buildapi.BuildReasonPtr(buildapi.BuildStrategyNotFound)
		s.Build.Status.Message = pointer.String(fmt.Sprintf("buildStrategy %s does not exist in namespace %s", strategyName, s.Build.Namespace))
		return nil
	}

	return err
}

func (s Strategy) validateClusterBuildStrategy(ctx context.Context, strategyName string) error {
	clusterBuildStrategy := &buildapi.ClusterBuildStrategy{}
	err := s.Client.Get(ctx, types.NamespacedName{Name: strategyName}, clusterBuildStrategy)
	if err == nil {
		s.validateBuildParams(clusterBuildStrategy.GetParameters())
		s.validateBuildVolumes(clusterBuildStrategy.GetVolumes())
		return nil
	}

	if apierrors.IsNotFound(err) {
		s.Build.Status.Reason = buildapi.BuildReasonPtr(buildapi.ClusterBuildStrategyNotFound)
		s.Build.Status.Message = pointer.String(fmt.Sprintf("clusterBuildStrategy %s does not exist", strategyName))
		return nil
	}

	return err
}

func (s Strategy) validateBuildParams(parameterDefinitions []buildapi.Parameter) {
	valid, reason, message := BuildParameters(parameterDefinitions, s.Build.Spec.ParamValues)
	if !valid {
		s.Build.Status.Reason = buildapi.BuildReasonPtr(reason)
		s.Build.Status.Message = pointer.String(message)
	}
}

func (s Strategy) validateBuildVolumes(strategyVolumes []buildapi.BuildStrategyVolume) {
	valid, reason, message := BuildVolumes(strategyVolumes, s.Build.Spec.Volumes)
	if !valid {
		s.Build.Status.Reason = buildapi.BuildReasonPtr(reason)
		s.Build.Status.Message = pointer.String(message)
	}
}
