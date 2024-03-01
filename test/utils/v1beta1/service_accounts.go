// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// This class is intended to host all CRUD calls for testing secrets primitive resources

// CreateSAFromName creates a simple ServiceAccount with the provided name if it does not exist.
func (t *TestBuild) CreateSAFromName(saName string) error {
	client := t.Clientset.CoreV1().ServiceAccounts(t.Namespace)
	_, err := client.Get(t.Context, saName, metav1.GetOptions{})
	// If the service account already exists, no error is returned
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}
	_, err = client.Create(t.Context, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: saName,
		}}, metav1.CreateOptions{})
	return err
}

// GetSA retrieves an existing service-account by name
// Deprecated: Use LookupServiceAccount instead.
func (t *TestBuild) GetSA(saName string) (*corev1.ServiceAccount, error) {
	client := t.Clientset.CoreV1().ServiceAccounts(t.Namespace)
	sa, err := client.Get(t.Context, saName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return sa, nil
}
