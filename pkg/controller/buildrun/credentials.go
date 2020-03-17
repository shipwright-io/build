package buildrun

import (
	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"

	corev1 "k8s.io/api/core/v1"
)

func applyCredentials(build *buildv1alpha1.Build, buildRun *buildv1alpha1.BuildRun, serviceAccount *corev1.ServiceAccount) *corev1.ServiceAccount {

	// credentials of the source/git repo
	sourceSecret := build.Spec.Source.SecretRef
	if sourceSecret != nil {
		serviceAccount = updateServiceAccountIfSecretNotLinked(sourceSecret, serviceAccount)
	}

	// credentials of the 'Builder' image registry
	builderImage := build.Spec.BuilderImage
	if builderImage != nil && builderImage.SecretRef != nil {
		serviceAccount = updateServiceAccountIfSecretNotLinked(builderImage.SecretRef, serviceAccount)
	}

	// credentials of the 'output' image registry
	sourceSecret = build.Spec.Output.SecretRef
	if sourceSecret != nil {
		serviceAccount = updateServiceAccountIfSecretNotLinked(sourceSecret, serviceAccount)
	}

	return serviceAccount
}

func updateServiceAccountIfSecretNotLinked(sourceSecret *corev1.LocalObjectReference, serviceAccount *corev1.ServiceAccount) *corev1.ServiceAccount {
	isSecretPresent := false
	for _, credentialSecret := range serviceAccount.Secrets {
		if credentialSecret.Name == sourceSecret.Name {
			isSecretPresent = true
			break
		}
	}

	if !isSecretPresent {
		serviceAccount.Secrets = append(serviceAccount.Secrets, corev1.ObjectReference{
			Name: sourceSecret.Name,
		})
	}

	return serviceAccount
}
