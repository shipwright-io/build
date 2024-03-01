// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// This class is intended to host all CRUD calls for testing configmap primitive resources

// CreateConfigMap generates a ConfigMap on the current test namespace
func (t *TestBuild) CreateConfigMap(configMap *corev1.ConfigMap) error {
	client := t.Clientset.CoreV1().ConfigMaps(t.Namespace)
	_, err := client.Create(t.Context, configMap, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// DeleteConfigMap removes the desired configMap
func (t *TestBuild) DeleteConfigMap(name string) error {
	client := t.Clientset.CoreV1().ConfigMaps(t.Namespace)
	if err := client.Delete(t.Context, name, metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}
