// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/ctxlog"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	return buildRun.Name + "-sa"
}

// IsGeneratedServiceAccountUsed checks if a build run uses a generated service account
func IsGeneratedServiceAccountUsed(buildRun *buildv1alpha1.BuildRun) bool {
	return buildRun.Spec.ServiceAccount != nil && buildRun.Spec.ServiceAccount.Generate
}

// RetrieveServiceAccount provides either a default sa with a referenced secret or it will generate a new sa on the fly.
// When not using the generate feature, it will modify and return the default sa from a k8s namespace, which is "default"
// or the default sa inside an openshift namespace, which is "pipeline".
func RetrieveServiceAccount(ctx context.Context, client client.Client, build *buildv1alpha1.Build, buildRun *buildv1alpha1.BuildRun) (*corev1.ServiceAccount, error) {
	serviceAccount := &corev1.ServiceAccount{}

	if IsGeneratedServiceAccountUsed(buildRun) {
		serviceAccountName := GetGeneratedServiceAccountName(buildRun)

		serviceAccount.Name = serviceAccountName
		serviceAccount.Namespace = buildRun.Namespace

		// Create the service account, use CreateOrUpdate as it might exist already from a previous reconciliation that
		// succeeded to create the service account but failed to update the build run that references it
		ctxlog.Info(ctx, "create or update serviceAccount for BuildRun", namespace, buildRun.Namespace, name, serviceAccountName, "BuildRun", buildRun.Name)
		op, err := controllerutil.CreateOrUpdate(ctx, client, serviceAccount, func() error {
			serviceAccount.SetLabels(map[string]string{buildv1alpha1.LabelBuildRun: buildRun.Name})

			ownerReference := metav1.NewControllerRef(buildRun, buildv1alpha1.SchemeGroupVersion.WithKind("BuildRun"))
			serviceAccount.SetOwnerReferences([]metav1.OwnerReference{*ownerReference})

			ApplyCredentials(ctx, build, serviceAccount)

			return nil
		})
		if err != nil {
			return nil, err
		}
		ctxlog.Debug(ctx, "automatic generation of service account", namespace, serviceAccount.Namespace, name, serviceAccount.Name, "Operation", op)
	} else {
		// If ServiceAccount or the name of ServiceAccount in buildRun is nil, use pipeline serviceaccount
		if buildRun.Spec.ServiceAccount == nil || buildRun.Spec.ServiceAccount.Name == nil {
			serviceAccountName := PipelineServiceAccountName
			err := client.Get(ctx, types.NamespacedName{Name: serviceAccountName, Namespace: buildRun.Namespace}, serviceAccount)
			if err != nil && !apierrors.IsNotFound(err) {
				return nil, err
			} else if apierrors.IsNotFound(err) {
				serviceAccountName = DefaultServiceAccountName
				ctxlog.Info(ctx, "falling back to default serviceAccount", namespace, buildRun.Namespace)
				err = client.Get(ctx, types.NamespacedName{Name: serviceAccountName, Namespace: buildRun.Namespace}, serviceAccount)
				if err != nil {
					return nil, err
				}
			}
		} else {
			serviceAccountName := *buildRun.Spec.ServiceAccount.Name
			err := client.Get(ctx, types.NamespacedName{Name: serviceAccountName, Namespace: buildRun.Namespace}, serviceAccount)
			if err != nil {
				return nil, err
			}
		}

		// Add credentials and update the service account
		if modified := ApplyCredentials(ctx, build, serviceAccount); modified {
			ctxlog.Info(ctx, "updating ServiceAccount with secrets from build", namespace, serviceAccount.Namespace, name, serviceAccount.Name)
			if err := client.Update(ctx, serviceAccount); err != nil {
				return nil, err
			}
		}
	}
	return serviceAccount, nil
}
