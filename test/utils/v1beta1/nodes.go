// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apply "k8s.io/client-go/applyconfigurations/core/v1"
)

// GetNodes returns all Nodes for the TestBuild object
func (t *TestBuild) GetNodes() (*corev1.NodeList, error) {
	client := t.Clientset.CoreV1().Nodes()
	nodes, err := client.List(t.Context, metav1.ListOptions{})
	return nodes, err
}

// AddNodeTaint sets a taint on the given Node name
func (t *TestBuild) AddNodeTaint(name string, taint *corev1.Taint) error {
	client := t.Clientset.CoreV1().Nodes()
	taintApplyCfg := apply.Taint()
	taintApplyCfg.WithKey(taint.Key)
	taintApplyCfg.WithValue(taint.Value)
	taintApplyCfg.WithEffect(taint.Effect)
	nodeSpecApplyCfg := apply.NodeSpec()
	nodeSpecApplyCfg.WithTaints(taintApplyCfg)
	applyCfg := apply.Node(name)
	applyCfg.WithSpec(nodeSpecApplyCfg)
	_, err := client.Apply(t.Context, applyCfg, metav1.ApplyOptions{})
	if err != nil {
		return err
	}
	return nil
}

// RemoveNodeTaints removes all taints on the given Node name
func (t *TestBuild) RemoveNodeTaints(name string) error {
	client := t.Clientset.CoreV1().Nodes()
	applyCfg := apply.Node(name)
	taintApplyCfg := apply.Taint()
	nodeSpecApplyCfg := apply.NodeSpec()
	nodeSpecApplyCfg.WithTaints(taintApplyCfg)
	applyCfg.WithSpec(nodeSpecApplyCfg)
	_, err := client.Apply(t.Context, applyCfg, metav1.ApplyOptions{})
	if err != nil {
		return err
	}
	return nil
}
