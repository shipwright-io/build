// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
)

var _ = Describe("Multiple errors", func() {
	Context("Dealing with multiple errors", func() {
		It("should handle multiple errors", func() {
			error01 := errors.New("error01")
			error02 := errors.New("error02")
			msg := "handling multiple errors"
			customError := resources.HandleError(msg, error01, error02)
			Expect(customError).To(Equal(fmt.Errorf("errors: %s, %s, msg: %s", error01, error02, msg)))
		})
	})
})
