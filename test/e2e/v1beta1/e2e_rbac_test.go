// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("User RBAC for Shipwright", func() {

	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	It("should install an aggregated edit role for developers", func() {
		editRole, err := testBuild.Clientset.RbacV1().ClusterRoles().Get(ctx, "shipwright-build-aggregate-edit", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		expectedAggregates := []string{
			"rbac.authorization.k8s.io/aggregate-to-edit",
			"rbac.authorization.k8s.io/aggregate-to-admin",
		}
		for _, aggregate := range expectedAggregates {
			aggregateValue, exists := editRole.Labels[aggregate]
			Expect(exists).To(BeTrue())
			Expect(aggregateValue).To(Equal("true"))
		}
		// We should have at least two rules - one for ClusterBuildStrategy, another for all else
		// More than two rules is acceptable.
		Expect(len(editRole.Rules)).To(BeNumerically(">=", 2))
		for _, rule := range editRole.Rules {
			Expect(rule.APIGroups).To(ContainElement("shipwright.io"))
			for _, resource := range rule.Resources {
				if resource == "clusterbuildstrategies" {
					Expect(rule.Verbs).To(ContainElements("get", "list", "watch"))
					Expect(rule.Verbs).NotTo(ContainElement("create"))
					Expect(rule.Verbs).NotTo(ContainElement("update"))
					Expect(rule.Verbs).NotTo(ContainElement("patch"))
					Expect(rule.Verbs).NotTo(ContainElement("delete"))
				} else {
					Expect(rule.Verbs).To(ContainElements("get", "list", "watch", "create", "update", "patch", "delete"))
				}
			}
		}
	})

	It("should install an aggregated view role for all users", func() {
		viewRole, err := testBuild.Clientset.RbacV1().ClusterRoles().Get(ctx, "shipwright-build-aggregate-view", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		aggregateValue, exists := viewRole.Labels["rbac.authorization.k8s.io/aggregate-to-view"]
		Expect(exists).To(BeTrue())
		Expect(aggregateValue).To(Equal("true"))
		// We should have at least one rule, as this applies "view" permissions to all Shipwright Build objects
		// More rules are acceptable for future fine-grained controls.
		Expect(len(viewRole.Rules)).To(BeNumerically(">=", 1))
		for _, rule := range viewRole.Rules {
			Expect(rule.APIGroups).To(ContainElement("shipwright.io"))
			Expect(rule.Verbs).To(ContainElements("get", "list", "watch"))
			Expect(rule.Verbs).NotTo(ContainElement("create"))
			Expect(rule.Verbs).NotTo(ContainElement("update"))
			Expect(rule.Verbs).NotTo(ContainElement("patch"))
			Expect(rule.Verbs).NotTo(ContainElement("delete"))
		}
	})

})
