package build

import (
	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"

	corev1 "k8s.io/api/core/v1"
)

func applyCredentials(buildInstance *buildv1alpha1.Build, serviceAccount *corev1.ServiceAccount) *corev1.ServiceAccount {

	// credentials of the source/git repo
	sourceSecret := buildInstance.Spec.Source.SecretRef
	if sourceSecret != nil {
		serviceAccount = updateServiceAccountIfSecretNotLinked(sourceSecret, serviceAccount)
	}

	// credentials of the 'output' image registry
	sourceSecret = buildInstance.Spec.Output.SecretRef
	if sourceSecret != nil {
		serviceAccount = updateServiceAccountIfSecretNotLinked(sourceSecret, serviceAccount)
	}

	// credentials of the 'Builder' image registry
	builderImage := buildInstance.Spec.BuilderImage
	if builderImage != nil && builderImage.SecretRef != nil {
		serviceAccount = updateServiceAccountIfSecretNotLinked(builderImage.SecretRef, serviceAccount)
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
