// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"fmt"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/ctxlog"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// StrategyRef contains all required fields
// to validate a Build spec strategy definition
type StrategyRef struct {
	Build  *build.Build
	Client client.Client
}

// ValidatePath implements BuildPath interface and validates
// that the referenced strategy exists. This applies to both
// namespaced or cluster scoped strategies
func (s StrategyRef) ValidatePath(ctx context.Context) error {
	if s.Build.Spec.StrategyRef != nil {
		if s.Build.Spec.StrategyRef.Kind != nil {
			switch *s.Build.Spec.StrategyRef.Kind {
			case build.NamespacedBuildStrategyKind:
				if err := s.validateBuildStrategy(ctx, s.Build.Spec.StrategyRef.Name, s.Build); err != nil {
					return err
				}
			case build.ClusterBuildStrategyKind:
				if err := s.validateClusterBuildStrategy(ctx, s.Build.Spec.StrategyRef.Name, s.Build); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unknown strategy kind: %v", *s.Build.Spec.StrategyRef.Kind)
			}
		} else {
			ctxlog.Info(ctx, "buildStrategy kind is nil, use default NamespacedBuildStrategyKind")
			if err := s.validateBuildStrategy(ctx, s.Build.Spec.StrategyRef.Name, s.Build); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s StrategyRef) validateBuildStrategy(ctx context.Context, strategyName string, b *build.Build) error {
	buildStrategy := &build.BuildStrategy{}
	if err := s.Client.Get(ctx, types.NamespacedName{Name: strategyName, Namespace: b.Namespace}, buildStrategy); err != nil && !apierrors.IsNotFound(err) {
		return err
	} else if apierrors.IsNotFound(err) {
		b.Status.Reason = build.BuildStrategyNotFound
		b.Status.Message = fmt.Sprintf("buildStrategy %s does not exist in namespace %s", b.Spec.StrategyRef.Name, b.Namespace)
	}

	return nil
}

func (s StrategyRef) validateClusterBuildStrategy(ctx context.Context, strategyName string, b *build.Build) error {
	clusterBuildStrategy := &build.ClusterBuildStrategy{}
	if err := s.Client.Get(ctx, types.NamespacedName{Name: strategyName}, clusterBuildStrategy); err != nil && !apierrors.IsNotFound(err) {
		return err
	} else if apierrors.IsNotFound(err) {
		b.Status.Reason = build.ClusterBuildStrategyNotFound
		b.Status.Message = fmt.Sprintf("clusterBuildStrategy %s does not exist", b.Spec.StrategyRef.Name)
	}
	return nil
}
