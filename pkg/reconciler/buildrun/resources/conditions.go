// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/ctxlog"
)

// Common condition strings for reason, kind, etc.
const (
	ConditionUnknownStrategyKind                     string = "UnknownStrategyKind"
	ClusterBuildStrategyNotFound                     string = "ClusterBuildStrategyNotFound"
	BuildStrategyNotFound                            string = "BuildStrategyNotFound"
	ConditionSetOwnerReferenceFailed                 string = "SetOwnerReferenceFailed"
	ConditionFailed                                  string = "Failed"
	ConditionTaskRunIsMissing                        string = "TaskRunIsMissing"
	ConditionTaskRunGenerationFailed                 string = "TaskRunGenerationFailed"
	ConditionServiceAccountNotFound                  string = "ServiceAccountNotFound"
	ConditionBuildRegistrationFailed                 string = "BuildRegistrationFailed"
	ConditionBuildNotFound                           string = "BuildNotFound"
	ConditionMissingParameterValues                  string = "MissingParameterValues"
	ConditionRestrictedParametersInUse               string = "RestrictedParametersInUse"
	ConditionUndefinedParameter                      string = "UndefinedParameter"
	ConditionWrongParameterValueType                 string = "WrongParameterValueType"
	ConditionInconsistentParameterValues             string = "InconsistentParameterValues"
	ConditionEmptyArrayItemParameterValues           string = "EmptyArrayItemParameterValues"
	ConditionIncompleteConfigMapValueParameterValues string = "IncompleteConfigMapValueParameterValues"
	ConditionIncompleteSecretValueParameterValues    string = "IncompleteSecretValueParameterValues"
	BuildRunNameInvalid                              string = "BuildRunNameInvalid"
	BuildRunNoRefOrSpec                              string = "BuildRunNoRefOrSpec"
	BuildRunAmbiguousBuild                           string = "BuildRunAmbiguousBuild"
	BuildRunBuildFieldOverrideForbidden              string = "BuildRunBuildFieldOverrideForbidden"
)

// UpdateConditionWithFalseStatus sets the Succeeded condition fields and mark
// the condition as Status False. It also updates the object in the cluster by
// calling client Status Update
func UpdateConditionWithFalseStatus(ctx context.Context, client client.Client, buildRun *buildv1beta1.BuildRun, errorMessage string, reason string) error {
	now := metav1.Now()
	buildRun.Status.CompletionTime = &now
	buildRun.Status.SetCondition(&buildv1beta1.Condition{
		LastTransitionTime: now,
		Type:               buildv1beta1.Succeeded,
		Status:             corev1.ConditionFalse,
		Reason:             reason,
		Message:            errorMessage,
	})
	ctxlog.Debug(ctx, "updating buildRun status", namespace, buildRun.Namespace, name, buildRun.Name, "reason", reason)
	if err := client.Status().Update(ctx, buildRun); err != nil {
		return &ClientStatusUpdateError{err}
	}

	return nil
}
