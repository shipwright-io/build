package git

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Parsing Git Error Messages", func() {
	Context("parse raw to PrefixToken", func() {
		It("should recognize and parse fatal", func() {
			parsed := parsePrefix("fatal")

			Expect(parsed.scope).To(Equal(Fatal))
			Expect(parsed.raw).To(Equal("fatal"))
		})
		It("should recognize and parse remote", func() {
			parsed := parsePrefix("remote")

			Expect(parsed.scope).To(Equal(Remote))
			Expect(parsed.raw).To(Equal("remote"))
		})
		It("should recognize and parse warning", func() {
			parsed := parsePrefix("warning")

			Expect(parsed.scope).To(Equal(Warning))
			Expect(parsed.raw).To(Equal("warning"))
		})
		It("should not parse unknown input as general", func() {
			parsed := parsePrefix("random")

			Expect(parsed.scope).To(Equal(General))
			Expect(parsed.raw).To(Equal("random"))
		})
	})

	Context("Parse raw to ErrorToken", func() {
		It("should recognize and parse unknown branch", func() {
			parsed := parseErrorMessage("Remote branch not found")
			Expect(parsed.class).To(Equal(BranchNotFound))
		})
		It("should recognize and parse invalid auth key", func() {
			parsed := parseErrorMessage("could not read from remote.")
			Expect(parsed.class).To(Equal(AuthInvalidKey))
		})
		It("should recognize and parse invalid basic auth", func() {
			parsed := parseErrorMessage("Invalid username or password.")
			Expect(parsed.class).To(Equal(AuthInvalidUserOrPass))
		})
		It("should recognize denied terminal prompt e.g. for private repo with no auth", func() {
			parsed := parseErrorMessage("could not read Username for 'https://github.com': terminal prompts disabled.")
			Expect(parsed.class).To(Equal(AuthPrompted))
		})
		It("should recognize non-existing repo", func() {
			parsed := parseErrorMessage("Repository not found.")
			Expect(parsed.class).To(Equal(RepositoryNotFound))
		})
		It("should not be able to specify exact error class for unknown message type", func() {
			parsed := parseErrorMessage("Something went wrong")
			Expect(parsed.class).To(Equal(Unknown))
		})
	})
	Context("If remote exists then prioritize it", func() {
		It("case with repo not found", func() {
			tokens := parse("remote:\nremote: ========================================================================\nremote:\nremote: The project you were looking for could not be found or you don't have permission to view it.\nremote:\nremote: ========================================================================\nremote:\nfatal: Could not read from remote repository.\n\nPlease make sure you have the correct access rights\nand the repository exists.")
			errorResult := extractResultsFromTokens(tokens)
			Expect(errorResult.Reason.String()).To(Equal(RepositoryNotFound.String()))
		})
	})
})
