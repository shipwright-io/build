// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package conditions_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/shipwright-io/build/pkg/conditions"
	"github.com/shipwright-io/build/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Conditions", func() {

	var (
		ctl test.Catalog
	)

	Context("Operating on Conditions", func() {

		It("should be able to get a manager based on a buildrun status", func() {
			// BuildRun sample with an embedded condition of the type Succeeded
			br := ctl.BuildRunWithSucceededCondition()
			m := conditions.Manage(&br.Status)
			Expect(m).ToNot(BeNil())

			c := m.GetCondition(conditions.Succeeded)
			Expect(c).ToNot(BeNil())
		})

		It("should be able to retrieve an existing condition message", func() {
			// BuildRun sample with an embedded condition of the type Succeeded
			br := ctl.BuildRunWithSucceededCondition()

			// BuildRun implements StatusConditions, therefore it can operate on
			// an existing Condition
			msg := br.Status.GetCondition(conditions.Succeeded).GetMessage()
			Expect(msg).To(Equal("foo is not bar"))
		})

		It("should be able to retrieve an existing condition reason", func() {
			// BuildRun sample with an embedded condition of the type Succeeded
			br := ctl.BuildRunWithSucceededCondition()

			reason := br.Status.GetCondition(conditions.Succeeded).GetReason()
			Expect(reason).To(Equal("foobar"))
		})

		It("should be able to retrieve an existing condition status", func() {
			// BuildRun sample with an embedded condition of the type Succeeded
			br := ctl.BuildRunWithSucceededCondition()

			status := br.Status.GetCondition(conditions.Succeeded).GetStatus()
			Expect(status).To(Equal(corev1.ConditionUnknown))
		})

		It("should return nil if a condition is not available when operating on it", func() {
			br := ctl.DefaultBuildRun("foo", "bar")

			// when getting a condition that does not exists on the BuildRun, do not
			// panic but rather return a nil
			cond := br.Status.GetCondition(conditions.Succeeded)
			Expect(cond).To(BeNil())
		})

		It("should be able to set a condition based on a type", func() {
			br := ctl.DefaultBuildRun("foo", "bar")

			// generate a condition of the type Succeeded
			tmpCond := &conditions.Condition{
				Type:               conditions.Succeeded,
				Status:             corev1.ConditionUnknown,
				Message:            "foobar",
				Reason:             "foo is bar",
				LastTransitionTime: metav1.Now(),
			}

			// set the condition on the BuildRun resource
			br.Status.SetCondition(tmpCond)

			condType := br.Status.GetCondition(conditions.Succeeded).Type
			Expect(condType).To(Equal(conditions.Succeeded))

			condMsg := br.Status.GetCondition(conditions.Succeeded).GetMessage()
			Expect(condMsg).To(Equal("foobar"))
		})

		It("should be able to update an existing condition based on a type", func() {
			// BuildRun sample with an embedded condition of the type Succeeded
			br := ctl.BuildRunWithSucceededCondition()

			reason := br.Status.GetCondition(conditions.Succeeded).GetReason()
			Expect(reason).To(Equal("foobar"))

			// generate a condition in order to update the existing one
			tmpCond := &conditions.Condition{
				Type:               conditions.Succeeded,
				Status:             corev1.ConditionUnknown,
				Message:            "foobar was updated",
				Reason:             "foo is bar",
				LastTransitionTime: metav1.Now(),
			}

			// update the condition on the BuildRun resource
			br.Status.SetCondition(tmpCond)

			condMsg := br.Status.GetCondition(conditions.Succeeded).GetMessage()
			Expect(condMsg).To(Equal("foobar was updated"))
		})
	})
})
