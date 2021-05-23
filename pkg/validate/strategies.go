// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"fmt"
	"strings"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/ctxlog"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Strategy contains all required fields
// to validate a Build spec strategy definition
type Strategy struct {
	Build  *build.Build
	Client client.Client
}

// ValidatePath implements BuildPath interface and validates
// that the referenced strategy exists. This applies to both
// namespaced or cluster scoped strategies
func (s Strategy) ValidatePath(ctx context.Context) error {
	if s.Build.Spec.Strategy != nil {
		buildStrategy := &build.BuildStrategy{}
		if s.Build.Spec.Strategy.Kind != nil {
			switch *s.Build.Spec.Strategy.Kind {
			case build.NamespacedBuildStrategyKind:
				if err := s.validateBuildStrategy(ctx, s.Build.Spec.Strategy.Name, buildStrategy); err != nil {
					return err
				}
				if err := s.validateBuildParams(buildStrategy.Spec.Parameters); err != nil {
					return err
				}
			case build.ClusterBuildStrategyKind:
				clusterBuildStrategy := &build.ClusterBuildStrategy{}
				if err := s.validateClusterBuildStrategy(ctx, s.Build.Spec.Strategy.Name, clusterBuildStrategy); err != nil {
					return err
				}
				if err := s.validateBuildParams(clusterBuildStrategy.Spec.Parameters); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unknown strategy kind: %v", *s.Build.Spec.Strategy.Kind)
			}
		} else {
			ctxlog.Info(ctx, "buildStrategy kind is nil, use default NamespacedBuildStrategyKind", namespace, s.Build.Namespace, name, s.Build.Name)
			if err := s.validateBuildStrategy(ctx, s.Build.Spec.Strategy.Name, buildStrategy); err != nil {
				return err
			}
			if err := s.validateBuildParams(buildStrategy.Spec.Parameters); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s Strategy) validateBuildStrategy(ctx context.Context, strategyName string, buildStrategy *build.BuildStrategy) error {
	if err := s.Client.Get(ctx, types.NamespacedName{Name: strategyName, Namespace: s.Build.Namespace}, buildStrategy); err != nil && !apierrors.IsNotFound(err) {
		return err
	} else if apierrors.IsNotFound(err) {
		s.Build.Status.Reason = build.BuildStrategyNotFound
		s.Build.Status.Message = fmt.Sprintf("buildStrategy %s does not exist in namespace %s", s.Build.Spec.Strategy.Name, s.Build.Namespace)
	}

	return nil
}

func (s Strategy) validateClusterBuildStrategy(ctx context.Context, strategyName string, clusterBuildStrategy *build.ClusterBuildStrategy) error {
	if err := s.Client.Get(ctx, types.NamespacedName{Name: strategyName}, clusterBuildStrategy); err != nil && !apierrors.IsNotFound(err) {
		return err
	} else if apierrors.IsNotFound(err) {
		s.Build.Status.Reason = build.ClusterBuildStrategyNotFound
		s.Build.Status.Message = fmt.Sprintf("clusterBuildStrategy %s does not exist", s.Build.Spec.Strategy.Name)
	}
	return nil
}

func (s Strategy) validateBuildParams(parameters []build.Parameter) error {

	if len(s.Build.Spec.ParamValues) == 0 {
		return nil
	}

	// Check that the Build param is defined in the strategies parameters
	s.validateParamsInStrategies(s.Build.Spec.ParamValues, parameters)

	// Check that the Build param is not a restricted shipwright one
	s.validateParamsNamesDefinition()

	return nil
}

func (s Strategy) validateParamsNamesDefinition() {

	undesiredParams := []string{}

	for _, p := range s.Build.Spec.ParamValues {
		if isReserved := resources.IsSystemReservedParameter(p.Name); isReserved {
			undesiredParams = append(undesiredParams, p.Name)
		}
	}

	if len(undesiredParams) > 0 {
		s.Build.Status.Reason = build.RestrictedParametersInUse
		s.Build.Status.Message = fmt.Sprintf("restricted parameters in use: %s", strings.Join(undesiredParams, ","))
	}
}

func (s Strategy) validateParamsInStrategies(params []build.ParamValue, parameters []build.Parameter) {

	undefinedParams := []string{}
	definedParameter := false
	for _, bp := range params {
		for _, sp := range parameters {
			if bp.Name == sp.Name {
				definedParameter = true
			}
		}
		if !definedParameter {
			undefinedParams = append(undefinedParams, bp.Name)
		}
		definedParameter = false
	}

	if len(undefinedParams) > 0 {
		s.Build.Status.Reason = build.UndefinedParameter
		s.Build.Status.Message = fmt.Sprintf("parameter not defined in the strategies: %s", strings.Join(undefinedParams, ","))
	}
}
