// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"
	"fmt"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/ctxlog"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// DefaultServiceAccountName defines the default sa name
	// in vanilla Kubernetes clusters
	DefaultServiceAccountName = "default"
	// PipelineServiceAccountName defines the default sa name
	// in vanilla OpenShift clusters
	PipelineServiceAccountName        = "pipeline"
	namespace                  string = "namespace"
	name                       string = "name"
)

// GetGeneratedServiceAccountName returns the name of the generated service account for a build run
func GetGeneratedServiceAccountName(buildRun *buildv1alpha1.BuildRun) string {
	return buildRun.Name
}

// IsGeneratedServiceAccountUsed checks if a build run uses a generated service account
func IsGeneratedServiceAccountUsed(buildRun *buildv1alpha1.BuildRun) bool {
	return buildRun.Spec.ServiceAccount != nil && buildRun.Spec.ServiceAccount.Generate != nil && *buildRun.Spec.ServiceAccount.Generate
}

// GenerateSA generates a new service account on the fly
func GenerateSA(ctx context.Context, client client.Client, build *buildv1alpha1.Build, buildRun *buildv1alpha1.BuildRun) (serviceAccount *corev1.ServiceAccount, err error) {
	serviceAccount = &corev1.ServiceAccount{}
	err = client.Get(
		ctx,
		types.NamespacedName{
			Name:      GetGeneratedServiceAccountName(buildRun),
			Namespace: buildRun.Namespace},
		serviceAccount)

	switch {
	case err == nil: // if the service account already exists, do nothing and just return it
		ctxlog.Info(ctx, "serviceAccount for BuildRun already exists", namespace, buildRun.Namespace, name, serviceAccount.Name, "BuildRun", buildRun.Name)
		return serviceAccount, nil

	case apierrors.IsNotFound(err):
		// populate the object with well known required fields
		serviceAccount = &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      GetGeneratedServiceAccountName(buildRun),
				Namespace: buildRun.Namespace,
				Labels:    map[string]string{buildv1alpha1.LabelBuildRun: buildRun.Name},
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(buildRun, buildv1alpha1.SchemeGroupVersion.WithKind("BuildRun")),
				},
			},
			AutomountServiceAccountToken: pointer.Bool(false),
		}
		ctxlog.Debug(ctx, "automatic generation of service account", namespace, serviceAccount.Namespace, name, serviceAccount.Name)

		// add the secrets references into the new sa
		ApplyCredentials(ctx, build, buildRun, serviceAccount)

		// if we didnt have the sa, then create one and ensure the Build secrets are referenced
		if err := client.Create(ctx, serviceAccount); err != nil {
			return nil, err
		}

		ctxlog.Info(ctx, "created serviceAccount for BuildRun", namespace, buildRun.Namespace, name, serviceAccount.Name, "BuildRun", buildRun.Name)
		return serviceAccount, nil

	default:
		return nil, err
	}
}

// DeleteServiceAccount deletes the service account of a completed BuildRun if the service account
// was generated
func DeleteServiceAccount(ctx context.Context, client client.Client, completedBuildRun *buildv1alpha1.BuildRun) error {
	if !IsGeneratedServiceAccountUsed(completedBuildRun) {
		return nil
	}

	serviceAccount := &corev1.ServiceAccount{}
	serviceAccount.Name = GetGeneratedServiceAccountName(completedBuildRun)
	serviceAccount.Namespace = completedBuildRun.Namespace

	ctxlog.Info(ctx, "deleting service account", namespace, completedBuildRun.Namespace, name, completedBuildRun.Name, "serviceAccount", serviceAccount.Name)
	if err := client.Delete(ctx, serviceAccount); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}

// getDefaultNamespaceSA retrieves a pipeline or default sa per namespace. This is used when users do not specify a service account
// to use on BuildRuns
func getDefaultNamespaceSA(ctx context.Context, client client.Client, buildRun *buildv1alpha1.BuildRun) (*corev1.ServiceAccount, error) {
	// Note: If the default SA is not in the namespace, the controller will be always reconciling until if finds it or until the
	// BuildRun gets deleted
	serviceAccount := &corev1.ServiceAccount{}

	err := client.Get(ctx, types.NamespacedName{Name: PipelineServiceAccountName, Namespace: buildRun.Namespace}, serviceAccount)
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	} else if apierrors.IsNotFound(err) {
		ctxlog.Info(ctx, "falling back to default serviceAccount", namespace, buildRun.Namespace)
		err = client.Get(ctx, types.NamespacedName{Name: DefaultServiceAccountName, Namespace: buildRun.Namespace}, serviceAccount)
		if err != nil {
			return nil, err
		}
	}
	return serviceAccount, nil
}

// RetrieveServiceAccount provides either a default sa with a referenced secret or it will generate a new sa on the fly.
// When not using the generate feature, it will modify and return the default sa from a k8s namespace, which is "default"
// or the default sa inside an openshift namespace, which is "pipeline".
func RetrieveServiceAccount(ctx context.Context, client client.Client, build *buildv1alpha1.Build, buildRun *buildv1alpha1.BuildRun) (serviceAccount *corev1.ServiceAccount, err error) {
	// generate or retrieve an existing autogenerated sa
	if IsGeneratedServiceAccountUsed(buildRun) {
		return GenerateSA(ctx, client, build, buildRun)
	}

	if buildRun.Spec.ServiceAccount != nil && buildRun.Spec.ServiceAccount.Name != nil {
		serviceAccountName := *buildRun.Spec.ServiceAccount.Name

		// here we might need to update Status Conditions and Fail the BR
		serviceAccount = &corev1.ServiceAccount{}
		if err = client.Get(ctx, types.NamespacedName{Name: serviceAccountName, Namespace: buildRun.Namespace}, serviceAccount); err != nil {
			if apierrors.IsNotFound(err) {
				if updateErr := UpdateConditionWithFalseStatus(ctx, client, buildRun, fmt.Sprintf("service account %s not found", serviceAccountName), ConditionServiceAccountNotFound); updateErr != nil {
					return nil, HandleError("failed to retrieve service account", err, updateErr)
				}
			}

			return nil, err
		}

	} else {
		// we default to pipeline/default sa
		serviceAccount, err = getDefaultNamespaceSA(ctx, client, buildRun)
		if err != nil {
			return nil, err
		}
	}

	// Add credentials and update the service account
	if modified := ApplyCredentials(ctx, build, buildRun, serviceAccount); modified {
		ctxlog.Info(ctx, "updating ServiceAccount with secrets from build", namespace, serviceAccount.Namespace, name, serviceAccount.Name)
		if err := client.Update(ctx, serviceAccount); err != nil {
			return nil, err
		}
	}

	return serviceAccount, nil
}
