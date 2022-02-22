// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/test"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
)

var _ = Describe("For a Kubernetes cluster with Tekton and build installed", func() {
	var (
		testID string
		err    error

		build         *buildv1alpha1.Build
		buildRun      *buildv1alpha1.BuildRun
		buildStrategy *buildv1alpha1.BuildStrategy
		configMap     *corev1.ConfigMap
		secret        *corev1.Secret
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

		if buildStrategy != nil {
			testBuild.DeleteBuildStrategy(buildStrategy.Name)
			buildStrategy = nil
		}

		if configMap != nil {
			testBuild.DeleteConfigMap(configMap.Name)
			configMap = nil
		}

		if secret != nil {
			testBuild.DeleteSecret(secret.Name)
			secret = nil
		}
	})

	Context("when using a cluster build strategy is used that uses a lot parameters", func() {
		BeforeEach(func() {
			buildStrategy, err = testBuild.Catalog.LoadBuildStrategyFromBytes([]byte(test.BuildStrategyWithParameterVerification))
			Expect(err).ToNot(HaveOccurred())
			err = testBuild.CreateBuildStrategy(buildStrategy)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when a secret and a configmap are in place with suitable values", func() {
			BeforeEach(func() {
				// prepare a ConfigMap
				configMap = testBuild.Catalog.ConfigMapWithData("a-configmap", testBuild.Namespace, map[string]string{
					"number1": "1",
					"shell":   "/bin/bash",
				})
				err = testBuild.CreateConfigMap(configMap)
				Expect(err).ToNot(HaveOccurred())

				// prepare a secret
				secret = testBuild.Catalog.SecretWithStringData("a-secret", testBuild.Namespace, map[string]string{
					"number2": "2",
					"number3": "3",
				})
				err = testBuild.CreateSecret(secret)
				Expect(err).ToNot(HaveOccurred())
			})

			Context("when a Build is in place that sets some of the parameters", func() {
				BeforeEach(func() {
					testID = generateTestID("params")

					build, err = NewBuildPrototype().
						BuildStrategy(buildStrategy.Name).
						Name(testID).
						Namespace(testBuild.Namespace).
						// The source is not actually used by the build, so just take a small one
						SourceGit("https://github.com/shipwright-io/sample-go.git").
						// There is not actually an image pushed
						OutputImage("dummy").
						// The parameters
						StringParamValue("env1", "13").
						StringParamValueFromConfigMap("env2", "a-configmap", "number1", pointer.String("2${CONFIGMAP_VALUE}")).
						ArrayParamValueFromConfigMap("commands", "a-configmap", "shell", nil).
						ArrayParamValue("commands", "-c").
						Create()
					Expect(err).ToNot(HaveOccurred())
				})

				It("correctly runs a BuildRun that passes the remaining parameters", func() {
					buildRun, err = NewBuildRunPrototype().
						ForBuild(build).
						Name(testID).
						GenerateServiceAccount().
						StringParamValue("image", "registry.access.redhat.com/ubi8/ubi-minimal").
						StringParamValueFromSecret("env3", "a-secret", "number2", nil).
						ArrayParamValueFromSecret("args", "a-secret", "number3", pointer.String("${SECRET_VALUE}9")).
						ArrayParamValue("args", "47").
						Create()
					Expect(err).ToNot(HaveOccurred())

					validateBuildRunToSucceed(testBuild, buildRun)

					// we verify the image digest here which is mis-used by the strategy to store a calculated sum
					// 13 (env1) + 21 (env2 = 2${a-configmap:number1}) + 2 (env3 = ${a-secret:number2}) + 39 (args[0] = ${a-secret:number3}9) + 47 (args[1]) = 122
					buildRun, err = testBuild.LookupBuildRun(types.NamespacedName{
						Namespace: buildRun.Namespace,
						Name:      buildRun.Name,
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(buildRun.Status.Output).NotTo(BeNil())
					Expect(buildRun.Status.Output.Size).To(BeEquivalentTo(122))
				})
			})
		})
	})
})
