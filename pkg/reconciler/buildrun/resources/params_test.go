// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

var _ = Describe("Params overrides", func() {

	DescribeTable("original params can be overridden",
		func(b []buildv1alpha1.ParamValue, br []buildv1alpha1.ParamValue, expected types.GomegaMatcher) {
			Expect(overrideParams(b, br)).To(expected)
		},

		Entry("override a single parameter",
			[]buildv1alpha1.ParamValue{
				{Name: "a", Value: "2"},
			}, []buildv1alpha1.ParamValue{
				{Name: "a", Value: "3"},
			}, ContainElements([]buildv1alpha1.ParamValue{
				{Name: "a", Value: "3"},
			})),

		Entry("override two parameters",
			[]buildv1alpha1.ParamValue{
				{Name: "a", Value: "2"},
				{Name: "b", Value: "2"},
			}, []buildv1alpha1.ParamValue{
				{Name: "a", Value: "3"},
				{Name: "b", Value: "3"},
			}, ContainElements([]buildv1alpha1.ParamValue{
				{Name: "a", Value: "3"},
				{Name: "b", Value: "3"},
			})),

		Entry("override multiple parameters",
			[]buildv1alpha1.ParamValue{
				{Name: "a", Value: "2"},
				{Name: "b", Value: "2"},
				{Name: "c", Value: "2"},
			}, []buildv1alpha1.ParamValue{
				{Name: "a", Value: "6"},
				{Name: "c", Value: "6"},
			}, ContainElements([]buildv1alpha1.ParamValue{
				{Name: "a", Value: "6"},
				{Name: "b", Value: "2"},
				{Name: "c", Value: "6"},
			})),

		Entry("dont override when second list is empty",
			[]buildv1alpha1.ParamValue{
				{Name: "t", Value: "2"},
				{Name: "z", Value: "2"},
				{Name: "g", Value: "2"},
			},
			[]buildv1alpha1.ParamValue{
				// no overrides
			},
			ContainElements([]buildv1alpha1.ParamValue{
				{Name: "t", Value: "2"},
				{Name: "z", Value: "2"},
				{Name: "g", Value: "2"},
			})),

		Entry("override when first list is empty but not the second list",
			[]buildv1alpha1.ParamValue{
				// no original values
			}, []buildv1alpha1.ParamValue{
				{Name: "a", Value: "6"},
				{Name: "c", Value: "6"},
			}, ContainElements([]buildv1alpha1.ParamValue{
				{Name: "a", Value: "6"},
				{Name: "c", Value: "6"},
			})),

		Entry("override multiple parameters if the match and add them if not present in first list",
			[]buildv1alpha1.ParamValue{
				{Name: "a", Value: "2"},
			}, []buildv1alpha1.ParamValue{
				{Name: "a", Value: "22"},
				{Name: "b", Value: "20"},
				{Name: "c", Value: "10"},
				{Name: "d", Value: "8"},
				{Name: "e", Value: "4"},
			}, ContainElements([]buildv1alpha1.ParamValue{
				{Name: "a", Value: "22"},
				{Name: "b", Value: "20"},
				{Name: "c", Value: "10"},
				{Name: "d", Value: "8"},
				{Name: "e", Value: "4"},
			})),
	)
})
