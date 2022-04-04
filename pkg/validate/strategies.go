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

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
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
	var (
		builderStrategy build.BuilderStrategy
		strategyExists  bool
		err             error
	)

	if s.Build.Spec.Strategy.Kind != nil {
		switch *s.Build.Spec.Strategy.Kind {
		case build.NamespacedBuildStrategyKind:
			buildStrategy := &build.BuildStrategy{}
			strategyExists, err = s.validateBuildStrategy(ctx, s.Build.Spec.Strategy.Name, buildStrategy)
			builderStrategy = buildStrategy
		case build.ClusterBuildStrategyKind:
			clusterBuildStrategy := &build.ClusterBuildStrategy{}
			strategyExists, err = s.validateClusterBuildStrategy(ctx, s.Build.Spec.Strategy.Name, clusterBuildStrategy)
			builderStrategy = clusterBuildStrategy
		default:
			return fmt.Errorf("unknown strategy kind: %v", *s.Build.Spec.Strategy.Kind)
		}
	} else {
		ctxlog.Info(ctx, "buildStrategy kind is nil, use default NamespacedBuildStrategyKind", namespace, s.Build.Namespace, name, s.Build.Name)
		buildStrategy := &build.BuildStrategy{}
		strategyExists, err = s.validateBuildStrategy(ctx, s.Build.Spec.Strategy.Name, buildStrategy)
		builderStrategy = buildStrategy
	}

	if err != nil {
		return err
	}

	if strategyExists {
		s.validateBuildParams(builderStrategy.GetParameters())
	}

	return nil
}

func (s Strategy) validateBuildStrategy(ctx context.Context, strategyName string, buildStrategy *build.BuildStrategy) (bool, error) {
	if err := s.Client.Get(ctx, types.NamespacedName{Name: strategyName, Namespace: s.Build.Namespace}, buildStrategy); err != nil && !apierrors.IsNotFound(err) {
		return false, err
	} else if apierrors.IsNotFound(err) {
		s.Build.Status.Reason = build.BuildReasonPtr(build.BuildStrategyNotFound)
		s.Build.Status.Message = pointer.String(fmt.Sprintf("buildStrategy %s does not exist in namespace %s", s.Build.Spec.Strategy.Name, s.Build.Namespace))
		return false, nil
	}
	return true, nil
}

func (s Strategy) validateClusterBuildStrategy(ctx context.Context, strategyName string, clusterBuildStrategy *build.ClusterBuildStrategy) (bool, error) {
	if err := s.Client.Get(ctx, types.NamespacedName{Name: strategyName}, clusterBuildStrategy); err != nil && !apierrors.IsNotFound(err) {
		return false, err
	} else if apierrors.IsNotFound(err) {
		s.Build.Status.Reason = build.BuildReasonPtr(build.ClusterBuildStrategyNotFound)
		s.Build.Status.Message = pointer.String(fmt.Sprintf("clusterBuildStrategy %s does not exist", s.Build.Spec.Strategy.Name))
		return false, nil
	}
	return true, nil
}

func (s Strategy) validateBuildParams(parameterDefinitions []build.Parameter) {
	valid, reason, message := BuildParameters(parameterDefinitions, s.Build.Spec.ParamValues)

	if !valid {
		s.Build.Status.Reason = build.BuildReasonPtr(reason)
		s.Build.Status.Message = pointer.String(message)
	}
}
