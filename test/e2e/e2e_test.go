package e2e

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("For a Kubernetes cluster with Tekton and build installed", func() {

	var (
		namespace string
	)

	BeforeEach(func() {
		ns, err := ctx.GetWatchNamespace()
		Expect(err).ToNot(HaveOccurred())
		namespace = ns
	})

	Context("when a Buildah build is defined", func() {

		BeforeEach(func() {
			// create the build definition
			createBuild(namespace, "buildah", "samples/build/build_buildah_cr.yaml")
		})

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, "buildah")
			}
		})

		It("successfully runs a build", func() {
			br, err := buildRunTestData(namespace, "buildah", "samples/buildrun/buildrun_buildah_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(namespace, br)
		})
	})

	Context("when a Buildah build with a contextDir and a custom Dockerfile name is defined", func() {

		BeforeEach(func() {
			// create the build definition
			createBuild(namespace, "buildah-custom-context-dockerfile", "test/data/build_buildah_cr_custom_context+dockerfile.yaml")
		})

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, "buildah-custom-context-dockerfile")
			}
		})

		It("successfully runs a build", func() {
			br, err := buildRunTestData(namespace, "buildah-custom-context-dockerfile", "test/data/buildrun_buildah_cr_custom_context+dockerfile.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(namespace, br)
			validateBuildDeletion(namespace, "buildah-custom-context-dockerfile", br, false)
		})
	})

	Context("when a heroku Buildpacks build is defined using a cluster strategy", func() {

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, "buildpacks-v3-heroku")
			}
		})

		It("successfully runs a build", func() {
			createBuild(namespace, "buildpacks-v3-heroku", "samples/build/build_buildpacks-v3-heroku_cr.yaml")
			br, err := buildRunTestData(namespace, "buildpacks-v3-heroku", "samples/buildrun/buildrun_buildpacks-v3-heroku_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(namespace, br)
		})
	})

	Context("when a heroku Buildpacks build is defined using a namespaced strategy", func() {

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, "buildpacks-v3-heroku-namespaced")
			}
		})

		It("successfully runs a build", func() {
			createBuild(namespace, "buildpacks-v3-heroku-namespaced", "samples/build/build_buildpacks-v3-heroku_namespaced_cr.yaml")
			br, err := buildRunTestData(namespace, "buildpacks-v3-heroku-namespaced", "samples/buildrun/buildrun_buildpacks-v3-heroku_namespaced_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(namespace, br)
		})
	})

	Context("when a Buildpacks v3 build is defined using a cluster strategy", func() {

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, "buildpacks-v3")
			}
		})

		It("successfully runs with a cluster scope strategy", func() {
			createBuild(namespace, "buildpacks-v3", "samples/build/build_buildpacks-v3_cr.yaml")
			br, err := buildRunTestData(namespace, "buildpacks-v3", "samples/buildrun/buildrun_buildpacks-v3_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(namespace, br)
			validateBuildDeletion(namespace, "buildpacks-v3", br, false)
		})
	})

	Context("when a Buildpacks v3 build is defined using a namespaced strategy", func() {

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, "buildpacks-v3-namespaced")
			}
		})

		It("successfully runs a build", func() {
			createBuild(namespace, "buildpacks-v3-namespaced", "samples/build/build_buildpacks-v3_namespaced_cr.yaml")
			br, err := buildRunTestData(namespace, "buildpacks-v3-namespaced", "samples/buildrun/buildrun_buildpacks-v3_namespaced_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(namespace, br)
		})
	})

	Context("when a Buildpacks v3 build is defined for a php runtime", func() {

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, "buildpacks-v3-php")
			}
		})

		It("successfully runs a build", func() {
			createBuild(namespace, "buildpacks-v3-php", "test/data/build_buildpacks-v3_php_cr.yaml")
			br, err := buildRunTestData(namespace, "buildpacks-v3-php", "test/data/buildrun_buildpacks-v3_php_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(namespace, br)
		})
	})

	Context("when a Buildpacks v3 build is defined for a ruby runtime", func() {

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, "buildpacks-v3-ruby")
			}
		})

		It("successfully runs a build", func() {
			createBuild(namespace, "buildpacks-v3-ruby", "test/data/build_buildpacks-v3_ruby_cr.yaml")
			br, err := buildRunTestData(namespace, "buildpacks-v3-ruby", "test/data/buildrun_buildpacks-v3_ruby_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(namespace, br)
		})
	})

	Context("when a Buildpacks v3 build is defined for a golang runtime", func() {

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, "buildpacks-v3-golang")
			}
		})

		It("successfully runs a build", func() {
			createBuild(namespace, "buildpacks-v3-golang", "test/data/build_buildpacks-v3_golang_cr.yaml")
			br, err := buildRunTestData(namespace, "buildpacks-v3-golang", "test/data/buildrun_buildpacks-v3_golang_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(namespace, br)
		})
	})

	Context("when a Buildpacks v3 build is defined for a java runtime", func() {

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, "buildpacks-v3-java")
			}
		})

		It("successfully runs a build", func() {
			createBuild(namespace, "buildpacks-v3-java", "test/data/build_buildpacks-v3_java_cr.yaml")
			br, err := buildRunTestData(namespace, "buildpacks-v3-java", "test/data/buildrun_buildpacks-v3_java_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(namespace, br)
		})
	})

	Context("when a Kaniko build is defined", func() {

		BeforeEach(func() {
			// create the build definition
			createBuild(namespace, "kaniko", "samples/build/build_kaniko_cr.yaml")
		})

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, "kaniko")
			}
		})

		It("successfully runs a build", func() {
			br, err := buildRunTestData(namespace, "kaniko", "samples/buildrun/buildrun_kaniko_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(namespace, br)
			validateBuildDeletion(namespace, "kaniko", br, true)
		})
	})

	Context("when a Kaniko build with a Dockerfile that requires advanced permissions is defined", func() {

		BeforeEach(func() {
			// create the build definition
			createBuild(namespace, "kaniko-advanced-dockerfile", "test/data/build_kaniko_cr_advanced_dockerfile.yaml")
		})

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, "kaniko-advanced-dockerfile")
			}
		})

		It("successfully runs a build", func() {
			br, err := buildRunTestData(namespace, "kaniko-advanced-dockerfile", "test/data/buildrun_kaniko_cr_advanced_dockerfile.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(namespace, br)
		})
	})

	Context("when a Kaniko build with a contextDir and a custom Dockerfile name is defined", func() {

		BeforeEach(func() {
			// create the build definition
			createBuild(namespace, "kaniko-custom-context-dockerfile", "test/data/build_kaniko_cr_custom_context+dockerfile.yaml")
		})

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, "kaniko-custom-context-dockerfile")
			}
		})

		It("successfully runs a build", func() {
			br, err := buildRunTestData(namespace, "kaniko-custom-context-dockerfile", "test/data/buildrun_kaniko_cr_custom_context+dockerfile.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(namespace, br)
		})
	})

	Context("when a Kaniko build with a short timeout is defined", func() {

		BeforeEach(func() {
			// create the build definition
			createBuild(namespace, "kaniko-timeout", "test/data/build_timeout.yaml")
		})

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, "kaniko-timeout")
			}
		})

		It("fails the build run", func() {
			br, err := buildRunTestData(namespace, "kaniko-timeout", "test/data/buildrun_timeout.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToFail(namespace, br, "kaniko-timeout.*failed to finish within \"15s\"")
		})
	})

	Context("when a s2i build is defined", func() {

		BeforeEach(func() {
			// create the build definition
			createBuild(namespace, "s2i", "samples/build/build_source-to-image_cr.yaml")
		})

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, "s2i")
			}
		})

		It("successfully runs a build", func() {
			br, err := buildRunTestData(namespace, "s2i", "samples/buildrun/buildrun_source-to-image_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(namespace, br)
		})
	})

	Context("when a private source repository is used", func() {

		BeforeEach(func() {
			if os.Getenv(EnvVarEnablePrivateRepos) != "true" {
				Skip("Skipping test cases that use a private source repository")
			}
		})

		Context("when a Buildah build is defined to use a private GitHub repository", func() {

			BeforeEach(func() {
				// create the build definition
				createBuild(namespace, "private-github-buildah", "test/data/build_buildah_cr_private_github.yaml")
			})

			AfterEach(func() {
				if CurrentGinkgoTestDescription().Failed {
					Logf("Print failed BuildRun's log")
					outputBuildAndBuildRunStatusAndPodLogs(namespace, "private-github-buildah")
				}
			})

			It("successfully runs a build", func() {
				br, err := buildRunTestData(namespace, "private-github-buildah", "samples/buildrun/buildrun_buildah_cr.yaml")
				Expect(err).ToNot(HaveOccurred())

				validateBuildRunToSucceed(namespace, br)
			})
		})

		Context("when a Buildah build is defined to use a private GitLab repository", func() {

			BeforeEach(func() {
				// create the build definition
				createBuild(namespace, "private-gitlab-buildah", "test/data/build_buildah_cr_private_gitlab.yaml")
			})

			AfterEach(func() {
				if CurrentGinkgoTestDescription().Failed {
					Logf("Print failed BuildRun's log")
					outputBuildAndBuildRunStatusAndPodLogs(namespace, "private-gitlab-buildah")
				}
			})

			It("successfully runs a build", func() {
				br, err := buildRunTestData(namespace, "private-gitlab-buildah", "samples/buildrun/buildrun_buildah_cr.yaml")
				Expect(err).ToNot(HaveOccurred())

				validateBuildRunToSucceed(namespace, br)
			})
		})

		Context("when a Kaniko build is defined to use a private GitHub repository", func() {

			BeforeEach(func() {
				// create the build definition
				createBuild(namespace, "private-github-kaniko", "test/data/build_kaniko_cr_private_github.yaml")
			})

			AfterEach(func() {
				if CurrentGinkgoTestDescription().Failed {
					Logf("Print failed BuildRun's log")
					outputBuildAndBuildRunStatusAndPodLogs(namespace, "private-github-kaniko")
				}
			})

			It("successfully runs a build", func() {
				br, err := buildRunTestData(namespace, "private-github-kaniko", "samples/buildrun/buildrun_kaniko_cr.yaml")
				Expect(err).ToNot(HaveOccurred())

				validateBuildRunToSucceed(namespace, br)
			})
		})

		Context("when a Kaniko build is defined to use a private GitLab repository", func() {

			BeforeEach(func() {
				// create the build definition
				createBuild(namespace, "private-gitlab-kaniko", "test/data/build_kaniko_cr_private_gitlab.yaml")
			})

			AfterEach(func() {
				if CurrentGinkgoTestDescription().Failed {
					Logf("Print failed BuildRun's log")
					outputBuildAndBuildRunStatusAndPodLogs(namespace, "private-gitlab-kaniko")
				}
			})

			It("successfully runs a build", func() {
				br, err := buildRunTestData(namespace, "private-gitlab-kaniko", "samples/buildrun/buildrun_kaniko_cr.yaml")
				Expect(err).ToNot(HaveOccurred())

				validateBuildRunToSucceed(namespace, br)
			})
		})

		Context("when a s2i build is defined to use a private GitHub repository", func() {

			BeforeEach(func() {
				// create the build definition
				createBuild(namespace, "private-github-s2i", "test/data/build_source-to-image_cr_private_github.yaml")
			})

			AfterEach(func() {
				if CurrentGinkgoTestDescription().Failed {
					Logf("Print failed BuildRun's log")
					outputBuildAndBuildRunStatusAndPodLogs(namespace, "private-github-s2i")
				}
			})

			It("successfully runs a build", func() {
				br, err := buildRunTestData(namespace, "private-github-s2i", "samples/buildrun/buildrun_source-to-image_cr.yaml")
				Expect(err).ToNot(HaveOccurred())

				validateBuildRunToSucceed(namespace, br)
			})
		})
	})
})
