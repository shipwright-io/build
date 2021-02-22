// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package ready_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/shipwright-io/build/pkg/controller/ready"
)

var _ = Describe("controller ready helper", func() {
	var fileExists = func(filename string) bool {
		_, err := os.Stat(filename)
		return err == nil || os.IsExist(err)
	}

	Context("setting and unsetting ready file", func() {
		const filename = "/tmp/foobar"

		BeforeEach(func(){
			Expect(fileExists(filename)).To(BeFalse())
		})

		AfterEach(func(){
			os.Remove(filename)
		})

		It("creates a ready file with the given name when using Set", func() {
			r := NewFileReady(filename)
			Expect(r.Set()).ToNot(HaveOccurred())
			Expect(fileExists(filename)).To(BeTrue())
		})

		It("should not break if Set is used multiple times", func() {
			r := NewFileReady(filename)
			Expect(r.Set()).ToNot(HaveOccurred())
			Expect(r.Set()).ToNot(HaveOccurred())
			Expect(fileExists(filename)).To(BeTrue())
		})

		It("removes the created file when using Unset", func() {
			r := NewFileReady(filename)
			Expect(r.Set()).ToNot(HaveOccurred())
			Expect(fileExists(filename)).To(BeTrue())

			Expect(r.Unset()).ToNot(HaveOccurred())
			Expect(fileExists(filename)).To(BeFalse())
		})

		It("should not break if Unset ist used multiple times", func() {
			r := NewFileReady(filename)
			Expect(r.Set()).ToNot(HaveOccurred())
			Expect(fileExists(filename)).To(BeTrue())

			Expect(r.Unset()).ToNot(HaveOccurred())
			Expect(r.Unset()).ToNot(HaveOccurred())
			Expect(fileExists(filename)).To(BeFalse())
		})

		It("should not fail if the given file is already present", func() {
			_, err :=os.Create(filename)
			Expect(err).ToNot(HaveOccurred())

			r := NewFileReady(filename)
			Expect(r.Set()).ToNot(HaveOccurred())
			Expect(fileExists(filename)).To(BeTrue())
		})
	})
})
