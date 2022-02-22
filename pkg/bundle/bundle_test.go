// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package bundle_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/shipwright-io/build/pkg/bundle"
)

var _ = Describe("Bundle", func() {
	Context("packing and unpacking", func() {
		It("should pack and unpack a directory", func() {
			r, err := Pack("../../test/bundle")
			Expect(err).ToNot(HaveOccurred())
			Expect(r).ToNot(BeNil())

			tempDir, err := ioutil.TempDir("", "bundle")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(tempDir)

			err = Unpack(r, tempDir)
			Expect(err).ToNot((HaveOccurred()))

			Expect(filepath.Join(tempDir, "README.md")).To(BeAnExistingFile())

			Expect(filepath.Join(tempDir, ".someToolDir", "config.json")).ToNot(BeAnExistingFile())

			Expect(filepath.Join(tempDir, "somefile")).To(BeAnExistingFile())
			Expect(filepath.Join(tempDir, "linktofile")).To(BeAnExistingFile())
		})
	})
})
