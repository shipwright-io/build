// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

var _ = Describe("SingleValue", func() {

	value := "BUILD_VERSION=1.0.0"

	// verifies that the mutually-exclusive optional fields of SingleValue are
	// omitted from the serialized output when unset, instead of being rendered
	// as explicit nulls (e.g. "configMapValue": null)
	DescribeTable("the serialization of a ParamValue",
		func(param buildapi.ParamValue, wantPresent []string, wantAbsent []string) {
			out, err := json.Marshal(param)
			Expect(err).ToNot(HaveOccurred())

			for _, want := range wantPresent {
				Expect(string(out)).To(ContainSubstring(want))
			}
			for _, absent := range wantAbsent {
				Expect(string(out)).ToNot(ContainSubstring(absent))
			}
		},
		Entry("literal value set",
			buildapi.ParamValue{Name: "build-args", SingleValue: &buildapi.SingleValue{Value: &value}},
			[]string{`"value":"BUILD_VERSION=1.0.0"`},
			[]string{`"configMapValue"`, `"secretValue"`},
		),
		Entry("configMap value set",
			buildapi.ParamValue{Name: "p", SingleValue: &buildapi.SingleValue{ConfigMapValue: &buildapi.ObjectKeyRef{Name: "cm", Key: "k"}}},
			[]string{`"configMapValue"`},
			[]string{`"value"`, `"secretValue"`},
		),
		Entry("secret value set",
			buildapi.ParamValue{Name: "p", SingleValue: &buildapi.SingleValue{SecretValue: &buildapi.ObjectKeyRef{Name: "s", Key: "k"}}},
			[]string{`"secretValue"`},
			[]string{`"value"`, `"configMapValue"`},
		),
	)
})
