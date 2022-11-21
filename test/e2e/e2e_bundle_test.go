// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/docker/cli/cli/config"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

var _ = Describe("Test local source code (bundle) functionality", func() {

	insecure := false
	value, found := os.LookupEnv(EnvVarImageRepoInsecure)
	if found {
		var err error
		insecure, err = strconv.ParseBool(value)
		Expect(err).ToNot(HaveOccurred())
	}

	var (
		testID string
		err    error

		build    *buildv1alpha1.Build
		buildRun *buildv1alpha1.BuildRun
	)

	AfterEach(func() {
		if CurrentSpecReport().Failed() {
			printTestFailureDebugInfo(testBuild, testBuild.Namespace, testID)
		} else if buildRun != nil {
			validateServiceAccountDeletion(buildRun, testBuild.Namespace)
		}

		if buildRun != nil {
			testBuild.DeleteBR(buildRun.Name)
			buildRun = nil
		}

		if build != nil {
			testBuild.DeleteBuild(build.Name)
			build = nil
		}
	})

	Context("when using local source code bundle images as input", func() {
		var inputImage, outputImage string

		BeforeEach(func() {
			testID = generateTestID("bundle")

			inputImage = "ghcr.io/shipwright-io/sample-go/source-bundle:latest"
			outputImage = fmt.Sprintf("%s/%s:%s",
				os.Getenv(EnvVarImageRepo),
				testID,
				"latest",
			)
		})

		It("should work with Kaniko build strategy", func() {
			build, err = NewBuildPrototype().
				ClusterBuildStrategy("kaniko").
				Name(testID).
				Namespace(testBuild.Namespace).
				SourceBundle(inputImage).
				SourceContextDir("docker-build").
				Dockerfile("Dockerfile").
				OutputImage(outputImage).
				OutputImageCredentials(os.Getenv(EnvVarImageRepoSecret)).
				OutputImageInsecure(insecure).
				Create()
			Expect(err).ToNot(HaveOccurred())

			buildRun, err = NewBuildRunPrototype().
				Name(testID).
				ForBuild(build).
				GenerateServiceAccount().
				Create()
			Expect(err).ToNot(HaveOccurred())

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validateBuildRunResultsFromBundleSource(buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})

		It("should work with Buildpacks build strategy", func() {
			build, err = NewBuildPrototype().
				ClusterBuildStrategy("buildpacks-v3").
				Name(testID).
				Namespace(testBuild.Namespace).
				SourceBundle(inputImage).
				SourceContextDir("source-build").
				OutputImage(outputImage).
				OutputImageCredentials(os.Getenv(EnvVarImageRepoSecret)).
				OutputImageInsecure(insecure).
				Create()
			Expect(err).ToNot(HaveOccurred())

			buildRun, err = NewBuildRunPrototype().
				Name(testID).
				ForBuild(build).
				GenerateServiceAccount().
				Create()
			Expect(err).ToNot(HaveOccurred())

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validateBuildRunResultsFromBundleSource(buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})

		It("should work with Buildah build strategy", func() {
			buildPrototype := NewBuildPrototype().
				ClusterBuildStrategy("buildah-shipwright-managed-push").
				Name(testID).
				Namespace(testBuild.Namespace).
				SourceBundle(inputImage).
				SourceContextDir("docker-build").
				Dockerfile("Dockerfile").
				OutputImage(outputImage).
				OutputImageCredentials(os.Getenv(EnvVarImageRepoSecret)).
				OutputImageInsecure(insecure)

			if strings.Contains(outputImage, "cluster.local") {
				parts := strings.Split(outputImage, "/")
				host := parts[0]
				buildPrototype.ArrayParamValue("registries-insecure", host)
			}

			build, err = buildPrototype.Create()
			Expect(err).ToNot(HaveOccurred())

			buildRun, err = NewBuildRunPrototype().
				Name(testID).
				ForBuild(build).
				GenerateServiceAccount().
				Create()
			Expect(err).ToNot(HaveOccurred())

			buildRun = validateBuildRunToSucceed(testBuild, buildRun)
			validateBuildRunResultsFromBundleSource(buildRun)
			testBuild.ValidateImageDigest(buildRun)
		})

		It("should prune the source image after pulling it", func() {
			var secretName = os.Getenv(EnvVarImageRepoSecret)
			var registryName string
			var auth authn.Authenticator
			var tmpImage = fmt.Sprintf("%s/source-%s:%s",
				os.Getenv(EnvVarImageRepo),
				testID,
				"latest",
			)

			By("looking up the registry name", func() {
				ref, err := name.ParseReference(outputImage)
				Expect(err).ToNot(HaveOccurred())

				registryName = ref.Context().RegistryStr()
			})

			By("setting up the respective authenticator", func() {
				switch {
				case secretName != "":
					secret, err := testBuild.Clientset.
						CoreV1().
						Secrets(testBuild.Namespace).
						Get(testBuild.Context, secretName, v1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					dockerConfigJSON, ok := secret.Data[".dockerconfigjson"]
					Expect(ok).To(BeTrue())

					configFile, err := config.LoadFromReader(bytes.NewReader(dockerConfigJSON))
					Expect(err).ToNot(HaveOccurred())

					authConfig, err := configFile.GetAuthConfig(registryName)
					Expect(err).ToNot(HaveOccurred())

					auth = authn.FromConfig(authn.AuthConfig{
						Username:      authConfig.Username,
						Password:      authConfig.Password,
						Auth:          authConfig.Auth,
						IdentityToken: authConfig.IdentityToken,
						RegistryToken: authConfig.RegistryToken,
					})

				default:
					auth = authn.Anonymous
				}
			})

			By("creating a temporary new input image based on the default input image", func() {
				src, err := name.ParseReference(inputImage)
				Expect(err).ToNot(HaveOccurred())

				// Special case for a local registry in the cluster:
				// Since the test client is not running in the cluster, it relies on being able to
				// reach the same registry via a local port. Therefore, the image name needs to be
				// different for the image copy preparation step.
				var dstImage = tmpImage
				if strings.Contains(dstImage, "cluster.local") {
					dstImage = strings.ReplaceAll(
						dstImage,
						"registry.registry.svc.cluster.local",
						"localhost",
					)
				}

				dst, err := name.ParseReference(dstImage)
				Expect(err).ToNot(HaveOccurred())

				srcDesc, err := remote.Get(src)
				Expect(err).ToNot(HaveOccurred())

				image, err := srcDesc.Image()
				Expect(err).ToNot(HaveOccurred())

				Expect(remote.Write(
					dst,
					image,
					remote.WithContext(testBuild.Context),
					remote.WithAuth(auth),
				)).ToNot(HaveOccurred())
			})

			By("eventually running the actual build with prune option", func() {
				build, err = NewBuildPrototype().
					ClusterBuildStrategy("kaniko").
					Name(testID).
					Namespace(testBuild.Namespace).
					SourceBundle(tmpImage).
					SourceBundlePrune(buildv1alpha1.PruneAfterPull).
					SourceCredentials(secretName).
					SourceContextDir("docker-build").
					Dockerfile("Dockerfile").
					OutputImage(outputImage).
					OutputImageCredentials(secretName).
					OutputImageInsecure(insecure).
					Create()
				Expect(err).ToNot(HaveOccurred())

				buildRun, err = NewBuildRunPrototype().
					Name(testID).
					ForBuild(build).
					GenerateServiceAccount().
					Create()
				Expect(err).ToNot(HaveOccurred())
				validateBuildRunToSucceed(testBuild, buildRun)
			})

			By("checking the temporary input image was removed", func() {
				tmp, err := name.ParseReference(tmpImage)
				Expect(err).ToNot(HaveOccurred())

				_, err = remote.Head(tmp, remote.WithContext(testBuild.Context), remote.WithAuth(auth))
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
