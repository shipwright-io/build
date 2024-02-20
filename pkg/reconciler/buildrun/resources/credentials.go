// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/ctxlog"
)

// ApplyCredentials adds all credentials that are referenced by the build and adds them to the service account.
// The function returns true if the service account was modified.
func ApplyCredentials(ctx context.Context, build *buildv1beta1.Build, buildRun *buildv1beta1.BuildRun, serviceAccount *corev1.ServiceAccount) bool {

	modified := false

	// if output is overridden by buildrun, and if this override has credentials,
	// it should be added to the sa
	if buildRun.Spec.Output != nil && buildRun.Spec.Output.PushSecret != nil {
		modified = updateServiceAccountIfSecretNotLinked(ctx, *buildRun.Spec.Output.PushSecret, serviceAccount) || modified
	} else {
		// otherwise, if buildrun does not override the output credentials,
		// we should use the ones provided by the build
		if build.Spec.Output.PushSecret != nil {
			modified = updateServiceAccountIfSecretNotLinked(ctx, *build.Spec.Output.PushSecret, serviceAccount) || modified
		}
	}

	return modified
}

func updateServiceAccountIfSecretNotLinked(ctx context.Context, sourceSecret string, serviceAccount *corev1.ServiceAccount) bool {
	isSecretPresent := false
	for _, credentialSecret := range serviceAccount.Secrets {
		if credentialSecret.Name == sourceSecret {
			isSecretPresent = true
			break
		}
	}

	if !isSecretPresent {
		ctxlog.Debug(ctx, "adding secret to serviceAccount", "secret", sourceSecret, "serviceAccount", serviceAccount.Name)
		serviceAccount.Secrets = append(serviceAccount.Secrets, corev1.ObjectReference{
			Name: sourceSecret,
		})
		return true
	}

	return false
}
