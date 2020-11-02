// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// This class is intended to host all CRUD calls for testing secrets primitive resources

// CreateSecret generates a Secret on the current test namespace
func (t *TestBuild) CreateSecret(ns string, secret *corev1.Secret) error {
	client := t.Clientset.CoreV1().Secrets(ns)
	_, err := client.Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}
