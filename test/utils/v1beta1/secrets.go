// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// This class is intended to host all CRUD calls for testing secrets primitive resources

// CreateSecret generates a Secret on the current test namespace
func (t *TestBuild) CreateSecret(secret *corev1.Secret) error {
	client := t.Clientset.CoreV1().Secrets(t.Namespace)
	_, err := client.Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// DeleteSecret removes the desired secret
func (t *TestBuild) DeleteSecret(name string) error {
	client := t.Clientset.CoreV1().Secrets(t.Namespace)
	if err := client.Delete(context.TODO(), name, metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

// PatchSecret patches a secret based on name and with the provided data.
// It used the merge type strategy
func (t *TestBuild) PatchSecret(name string, data []byte) (*corev1.Secret, error) {
	return t.PatchSecretWithPatchType(name, data, types.MergePatchType)
}

// PatchSecretWithPatchType patches a secret with a desire data and patch strategy
func (t *TestBuild) PatchSecretWithPatchType(name string, data []byte, pt types.PatchType) (*corev1.Secret, error) {
	secInterface := t.Clientset.CoreV1().Secrets(t.Namespace)
	b, err := secInterface.Patch(context.TODO(), name, pt, data, metav1.PatchOptions{})
	if err != nil {
		return nil, err
	}
	return b, nil
}
