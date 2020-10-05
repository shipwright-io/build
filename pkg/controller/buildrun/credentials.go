// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package buildrun

import (
	"context"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/ctxlog"

	corev1 "k8s.io/api/core/v1"
)

// ApplyCredentials adds all credentials that are referenced by the build and adds them to the service account.
// The function returns true if the service account was modified.
func ApplyCredentials(ctx context.Context, build *buildv1alpha1.Build, serviceAccount *corev1.ServiceAccount) bool {

	modified := false

	// credentials of the source/git repo
	sourceSecret := build.Spec.Source.SecretRef
	if sourceSecret != nil {
		modified = updateServiceAccountIfSecretNotLinked(ctx, sourceSecret, serviceAccount) || modified
	}

	// credentials of the 'Builder' image registry
	builderImage := build.Spec.BuilderImage
	if builderImage != nil && builderImage.SecretRef != nil {
		modified = updateServiceAccountIfSecretNotLinked(ctx, builderImage.SecretRef, serviceAccount) || modified
	}

	// credentials of the 'output' image registry
	sourceSecret = build.Spec.Output.SecretRef
	if sourceSecret != nil {
		modified = updateServiceAccountIfSecretNotLinked(ctx, sourceSecret, serviceAccount) || modified
	}

	return modified
}

func updateServiceAccountIfSecretNotLinked(ctx context.Context, sourceSecret *corev1.LocalObjectReference, serviceAccount *corev1.ServiceAccount) bool {
	isSecretPresent := false
	for _, credentialSecret := range serviceAccount.Secrets {
		if credentialSecret.Name == sourceSecret.Name {
			isSecretPresent = true
			break
		}
	}

	if !isSecretPresent {
		ctxlog.Debug(ctx, "adding secret to serviceAccount", "secret", sourceSecret.Name, "serviceAccount", serviceAccount.Name)
		serviceAccount.Secrets = append(serviceAccount.Secrets, corev1.ObjectReference{
			Name: sourceSecret.Name,
		})
		return true
	}

	return false
}
