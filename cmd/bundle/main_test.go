// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package main_test

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/shipwright-io/build/cmd/bundle"
)

var _ = Describe("Bundle Loader", func() {
	var run = func(args ...string) error {
		log.SetOutput(ioutil.Discard)
		os.Args = append([]string{"tool"}, args...)
		return Do(context.Background())
	}

	var withTempDir = func(f func(target string)) {
		path, err := ioutil.TempDir(os.TempDir(), "bundle")
		Expect(err).ToNot(HaveOccurred())
		defer os.RemoveAll(path)

		f(path)
	}

	Context("Error cases", func() {
		It("should fail in case the image is not specified", func() {
			Expect(run(
				"--image", "",
			)).To(HaveOccurred())
		})
	})

	Context("Pulling image anonymously", func() {
		const exampleImage = "quay.io/shipwright/source-bundle:latest"

		It("should pull and unbundle an image from a public registry", func() {
			withTempDir(func(target string) {
				Expect(run(
					"--image", exampleImage,
					"--target", target,
				)).ToNot(HaveOccurred())

				Expect(filepath.Join(target, "LICENSE")).To(BeAnExistingFile())
			})
		})
	})
})
