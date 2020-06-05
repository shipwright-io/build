package e2e

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("For a Kubernetes cluster with Tekton and build installed", func() {

	var (
		namespace string
		testId    string
	)

	BeforeEach(func() {
		ns, err := ctx.GetWatchNamespace()
		Expect(err).ToNot(HaveOccurred())
		namespace = ns
	})

	Context("when a Buildah build is defined", func() {

		BeforeEach(func() {
			testId = generateTestID("buildah")

			// create the build definition
			createBuild(namespace, testId, "samples/build/build_buildah_cr.yaml")
		})

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, testId)
			}
		})

		It("successfully runs a build", func() {
			br, err := buildRunTestData(namespace, testId, "samples/buildrun/buildrun_buildah_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(namespace, br)
		})
	})

	Context("when a Buildah build with a contextDir and a custom Dockerfile name is defined", func() {

		BeforeEach(func() {
			testId = generateTestID("buildah-custom-context-dockerfile")

			// create the build definition
			createBuild(namespace, testId, "test/data/build_buildah_cr_custom_context+dockerfile.yaml")
		})

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, testId)
			}
		})

		It("successfully runs a build", func() {
			br, err := buildRunTestData(namespace, testId, "test/data/buildrun_buildah_cr_custom_context+dockerfile.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(namespace, br)
			validateBuildDeletion(namespace, testId, br, false)
		})
	})

	Context("when a heroku Buildpacks build is defined using a cluster strategy", func() {

		BeforeEach(func() {
			testId = generateTestID("buildpacks-v3-heroku")

			// create the build definition
			createBuild(namespace, testId, "samples/build/build_buildpacks-v3-heroku_cr.yaml")
		})

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, testId)
			}
		})

		It("successfully runs a build", func() {
			br, err := buildRunTestData(namespace, testId, "samples/buildrun/buildrun_buildpacks-v3-heroku_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(namespace, br)
		})
	})

	Context("when a heroku Buildpacks build is defined using a namespaced strategy", func() {

		BeforeEach(func() {
			testId = generateTestID("buildpacks-v3-heroku-namespaced")

			// create the build definition
			createBuild(namespace, testId, "samples/build/build_buildpacks-v3-heroku_namespaced_cr.yaml")
		})

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, testId)
			}
		})

		It("successfully runs a build", func() {
			br, err := buildRunTestData(namespace, testId, "samples/buildrun/buildrun_buildpacks-v3-heroku_namespaced_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(namespace, br)
		})
	})

	Context("when a Buildpacks v3 build is defined using a cluster strategy", func() {

		BeforeEach(func() {
			testId = generateTestID("buildpacks-v3")

			// create the build definition
			createBuild(namespace, testId, "samples/build/build_buildpacks-v3_cr.yaml")
		})

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, testId)
			}
		})

		It("successfully runs a build", func() {
			br, err := buildRunTestData(namespace, testId, "samples/buildrun/buildrun_buildpacks-v3_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(namespace, br)
			validateBuildDeletion(namespace, testId, br, false)
		})
	})

	Context("when a Buildpacks v3 build is defined using a namespaced strategy", func() {

		BeforeEach(func() {
			testId = generateTestID("buildpacks-v3-namespaced")

			// create the build definition
			createBuild(namespace, testId, "samples/build/build_buildpacks-v3_namespaced_cr.yaml")
		})

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, testId)
			}
		})

		It("successfully runs a build", func() {
			br, err := buildRunTestData(namespace, testId, "samples/buildrun/buildrun_buildpacks-v3_namespaced_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(namespace, br)
		})
	})

	Context("when a Buildpacks v3 build is defined for a php runtime", func() {

		BeforeEach(func() {
			testId = generateTestID("buildpacks-v3-php")

			// create the build definition
			createBuild(namespace, testId, "test/data/build_buildpacks-v3_php_cr.yaml")
		})

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, testId)
			}
		})

		It("successfully runs a build", func() {
			br, err := buildRunTestData(namespace, testId, "test/data/buildrun_buildpacks-v3_php_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(namespace, br)
		})
	})

	Context("when a Buildpacks v3 build is defined for a ruby runtime", func() {

		BeforeEach(func() {
			// issue track in paketo side: https://github.com/paketo-community/ruby/issues/59
			Skip("Skipping test case because Ruby support in paketo is still under development.")

			testId = generateTestID("buildpacks-v3-ruby")

			// create the build definition
			createBuild(namespace, testId, "test/data/build_buildpacks-v3_ruby_cr.yaml")
		})

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, testId)
			}
		})

		It("successfully runs a build", func() {
			br, err := buildRunTestData(namespace, testId, "test/data/buildrun_buildpacks-v3_ruby_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(namespace, br)
		})
	})

	Context("when a Buildpacks v3 build is defined for a golang runtime", func() {

		BeforeEach(func() {
			testId = generateTestID("buildpacks-v3-golang")

			// create the build definition
			createBuild(namespace, testId, "test/data/build_buildpacks-v3_golang_cr.yaml")
		})

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, testId)
			}
		})

		It("successfully runs a build", func() {
			br, err := buildRunTestData(namespace, testId, "test/data/buildrun_buildpacks-v3_golang_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(namespace, br)
		})
	})

	Context("when a Buildpacks v3 build is defined for a java runtime", func() {

		BeforeEach(func() {
			testId = generateTestID("buildpacks-v3-java")

			// create the build definition
			createBuild(namespace, testId, "test/data/build_buildpacks-v3_java_cr.yaml")
		})

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, testId)
			}
		})

		It("successfully runs a build", func() {
			br, err := buildRunTestData(namespace, testId, "test/data/buildrun_buildpacks-v3_java_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(namespace, br)
		})
	})

	Context("when a Kaniko build is defined", func() {

		BeforeEach(func() {
			testId = generateTestID("kaniko")

			// create the build definition
			createBuild(namespace, testId, "samples/build/build_kaniko_cr.yaml")
		})

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, testId)
			}
		})

		It("successfully runs a build", func() {
			br, err := buildRunTestData(namespace, testId, "samples/buildrun/buildrun_kaniko_cr.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(namespace, br)
			validateBuildDeletion(namespace, testId, br, true)
		})
	})

	Context("when a Kaniko build with a Dockerfile that requires advanced permissions is defined", func() {

		BeforeEach(func() {
			testId = generateTestID("kaniko-advanced-dockerfile")

			// create the build definition
			createBuild(namespace, testId, "test/data/build_kaniko_cr_advanced_dockerfile.yaml")
		})

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, testId)
			}
		})

		It("successfully runs a build", func() {
			br, err := buildRunTestData(namespace, testId, "test/data/buildrun_kaniko_cr_advanced_dockerfile.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(namespace, br)
		})
	})

	Context("when a Kaniko build with a contextDir and a custom Dockerfile name is defined", func() {

		BeforeEach(func() {
			testId = generateTestID("kaniko-custom-context-dockerfile")

			// create the build definition
			createBuild(namespace, testId, "test/data/build_kaniko_cr_custom_context+dockerfile.yaml")
		})

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, testId)
			}
		})

		It("successfully runs a build", func() {
			br, err := buildRunTestData(namespace, testId, "test/data/buildrun_kaniko_cr_custom_context+dockerfile.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToSucceed(namespace, br)
		})
	})

	Context("when a Kaniko build with a short timeout is defined", func() {

		BeforeEach(func() {
			testId = generateTestID("kaniko-timeout")

			// create the build definition
			createBuild(namespace, testId, "test/data/build_timeout.yaml")
		})

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, testId)
			}
		})

		It("fails the build run", func() {
			br, err := buildRunTestData(namespace, testId, "test/data/buildrun_timeout.yaml")
			Expect(err).ToNot(HaveOccurred())

			validateBuildRunToFail(namespace, br, "kaniko-timeout.*failed to finish within \"15s\"")
		})
	})

	Context("when a s2i build is defined", func() {

		BeforeEach(func() {
			testId = generateTestID("s2i")

			// create the build definition
			createBuild(namespace, testId, "samples/build/build_source-to-image_cr.yaml")
		})

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				Logf("Print failed BuildRun's log")
				outputBuildAndBuildRunStatusAndPodLogs(namespace, testId)
			}
		})

		It("successfully runs a build", func() {
			br, err := buildRunTestData(namespace, testId, "samples/buildrun/buildrun_source-to-image_cr.yaml")
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
				testId = generateTestID("private-github-buildah")

				// create the build definition
				createBuild(namespace, testId, "test/data/build_buildah_cr_private_github.yaml")
			})

			AfterEach(func() {
				if CurrentGinkgoTestDescription().Failed {
					Logf("Print failed BuildRun's log")
					outputBuildAndBuildRunStatusAndPodLogs(namespace, testId)
				}
			})

			It("successfully runs a build", func() {
				br, err := buildRunTestData(namespace, testId, "samples/buildrun/buildrun_buildah_cr.yaml")
				Expect(err).ToNot(HaveOccurred())

				validateBuildRunToSucceed(namespace, br)
			})
		})

		Context("when a Buildah build is defined to use a private GitLab repository", func() {

			BeforeEach(func() {
				testId = generateTestID("private-gitlab-buildah")

				// create the build definition
				createBuild(namespace, testId, "test/data/build_buildah_cr_private_gitlab.yaml")
			})

			AfterEach(func() {
				if CurrentGinkgoTestDescription().Failed {
					Logf("Print failed BuildRun's log")
					outputBuildAndBuildRunStatusAndPodLogs(namespace, testId)
				}
			})

			It("successfully runs a build", func() {
				br, err := buildRunTestData(namespace, testId, "samples/buildrun/buildrun_buildah_cr.yaml")
				Expect(err).ToNot(HaveOccurred())

				validateBuildRunToSucceed(namespace, br)
			})
		})

		Context("when a Kaniko build is defined to use a private GitHub repository", func() {

			BeforeEach(func() {
				testId = generateTestID("private-github-kaniko")

				// create the build definition
				createBuild(namespace, testId, "test/data/build_kaniko_cr_private_github.yaml")
			})

			AfterEach(func() {
				if CurrentGinkgoTestDescription().Failed {
					Logf("Print failed BuildRun's log")
					outputBuildAndBuildRunStatusAndPodLogs(namespace, testId)
				}
			})

			It("successfully runs a build", func() {
				br, err := buildRunTestData(namespace, testId, "samples/buildrun/buildrun_kaniko_cr.yaml")
				Expect(err).ToNot(HaveOccurred())

				validateBuildRunToSucceed(namespace, br)
			})
		})

		Context("when a Kaniko build is defined to use a private GitLab repository", func() {

			BeforeEach(func() {
				testId = generateTestID("private-gitlab-kaniko")

				// create the build definition
				createBuild(namespace, testId, "test/data/build_kaniko_cr_private_gitlab.yaml")
			})

			AfterEach(func() {
				if CurrentGinkgoTestDescription().Failed {
					Logf("Print failed BuildRun's log")
					outputBuildAndBuildRunStatusAndPodLogs(namespace, testId)
				}
			})

			It("successfully runs a build", func() {
				br, err := buildRunTestData(namespace, testId, "samples/buildrun/buildrun_kaniko_cr.yaml")
				Expect(err).ToNot(HaveOccurred())

				validateBuildRunToSucceed(namespace, br)
			})
		})

		Context("when a s2i build is defined to use a private GitHub repository", func() {

			BeforeEach(func() {
				testId = generateTestID("private-github-s2i")

				// create the build definition
				createBuild(namespace, testId, "test/data/build_source-to-image_cr_private_github.yaml")
			})

			AfterEach(func() {
				if CurrentGinkgoTestDescription().Failed {
					Logf("Print failed BuildRun's log")
					outputBuildAndBuildRunStatusAndPodLogs(namespace, testId)
				}
			})

			It("successfully runs a build", func() {
				br, err := buildRunTestData(namespace, testId, "samples/buildrun/buildrun_source-to-image_cr.yaml")
				Expect(err).ToNot(HaveOccurred())

				validateBuildRunToSucceed(namespace, br)
			})
		})
	})
})
