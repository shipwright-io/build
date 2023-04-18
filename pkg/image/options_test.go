// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package image_test

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/shipwright-io/build/pkg/image"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetOptions", func() {

	withTempFile := func(pattern string, f func(filename string)) {
		file, err := os.CreateTemp(os.TempDir(), pattern)
		Expect(err).ToNot(HaveOccurred())
		defer os.Remove(file.Name())

		f(file.Name())
	}

	withDockerConfigJSON := func(hostname string, username string, password string, f func(dockerConfigJSONPath string)) {
		withTempFile("docker.config", func(tempFile string) {
			err := os.WriteFile(tempFile, ([]byte(fmt.Sprintf("{\"auths\":{%q:{\"username\":%q,\"password\":%q}}}", hostname, username, password))), 0644)
			Expect(err).ToNot(HaveOccurred())

			f(tempFile)
		})
	}

	imageName, err := name.ParseReference("somehost/image:tag")
	Expect(err).ToNot(HaveOccurred())

	Context("without a dockerconfigjson", func() {

		It("constructs options and empty auth", func() {
			options, auth, err := image.GetOptions(context.TODO(), imageName, true, "", "test-agent")
			Expect(err).ToNot(HaveOccurred())

			// there is no way to further check what is in because the options are functions
			Expect(options).To(HaveLen(3))

			// auth is empty in all cases
			Expect(auth).ToNot(BeNil())
			Expect(auth.Username).To(Equal(""))
			Expect(auth.Password).To(Equal(""))
		})
	})

	Context("with a dockerconfigjson that matches the image name", func() {

		It("constructs options and auth with the matching user", func() {
			withDockerConfigJSON(authn.DefaultAuthKey, "aUser", "aPassword", func(dockerConfigJSONPath string) {

				options, auth, err := image.GetOptions(context.TODO(), imageName, true, dockerConfigJSONPath, "test-agent")
				Expect(err).ToNot(HaveOccurred())

				// there is no way to further check what is in because the options are functions
				Expect(options).To(HaveLen(3))

				// auth is empty in all cases
				Expect(auth).ToNot(BeNil())
				Expect(auth.Username).To(Equal("aUser"))
				Expect(auth.Password).To(Equal("aPassword"))
			})
		})
	})

	Context("with a dockerconfigjson that does not match the image name", func() {

		It("fails with an error", func() {
			withDockerConfigJSON("ghcr.io", "aUser", "aPassword", func(dockerConfigJSONPath string) {
				_, _, err := image.GetOptions(context.TODO(), imageName, true, dockerConfigJSONPath, "test-agent")
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
