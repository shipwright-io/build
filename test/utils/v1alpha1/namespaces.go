// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"time"

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
	_, err := client.Create(t.Context, ns, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// wait for the default service account to exist
	pollServiceAccount := func(ctx context.Context) (bool, error) {

		serviceAccountInterface := t.Clientset.CoreV1().ServiceAccounts(t.Namespace)

		_, err := serviceAccountInterface.Get(ctx, "default", metav1.GetOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return false, err
			}
			return false, nil
		}

		return true, nil
	}

	return wait.PollUntilContextTimeout(t.Context, t.Interval, t.TimeOut, true, pollServiceAccount)
}

// DeleteNamespace deletes the namespace with the current test name
func (t *TestBuild) DeleteNamespace() error {
	client := t.Clientset.CoreV1().Namespaces()

	if err := client.Delete(t.Context, t.Namespace, metav1.DeleteOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	// wait for the namespace to be deleted
	pollNamespace := func(ctx context.Context) (bool, error) {

		if _, err := client.Get(ctx, t.Namespace, metav1.GetOptions{}); err != nil && apierrors.IsNotFound(err) {
			return true, nil
		}

		return false, nil
	}

	return wait.PollUntilContextTimeout(t.Context, t.Interval, 5*time.Second, true, pollNamespace)
}
