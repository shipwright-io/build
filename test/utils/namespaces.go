// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// This class is intended to host all CRUD calls for Namespace primitive resources

// CreateNamespace generates a Namespace with the current test name
func (t *TestBuild) CreateNamespace() error {
	client := t.Clientset.CoreV1().Namespaces()
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: t.Namespace,
		},
	}
	_, err := client.Create(context.TODO(), ns, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// wait for the default service account to exist and contain the token secret
	var (
		pollServiceAccount = func() (bool, error) {

			serviceAccountInterface := t.Clientset.CoreV1().ServiceAccounts(t.Namespace)

			serviceAccount, err := serviceAccountInterface.Get(context.TODO(), "default", metav1.GetOptions{})
			if err != nil {
				if !apierrors.IsNotFound(err) {
					return false, err
				}
				return false, nil
			}

			if len(serviceAccount.Secrets) > 0 {
				return true, nil
			}

			return false, nil
		}
	)

	return wait.PollImmediate(t.Interval, t.TimeOut, pollServiceAccount)
}

// DeleteNamespaces remove existing namespaces that match the provided list name items
func (t *TestBuild) DeleteNamespaces(nsList []string) error {
	client := t.Clientset.CoreV1().Namespaces()

	for _, ns := range nsList {
		err := client.Delete(context.TODO(), ns, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}
