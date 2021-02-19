// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"os"
	"os/exec"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
)

var (
	operatorCmd *exec.Cmd

	ctx       *framework.Context
	globalCtx *framework.Context
	testingT  *testing.T

	clusterBuildStrategies = []string{
		"samples/buildstrategy/buildah/buildstrategy_buildah_cr.yaml",
		"samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3-heroku_cr.yaml",
		"samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_cr.yaml",
		"samples/buildstrategy/kaniko/buildstrategy_kaniko_cr.yaml",
		"samples/buildstrategy/source-to-image/buildstrategy_source-to-image_cr.yaml",
		"samples/buildstrategy/source-to-image/buildstrategy_source-to-image-redhat_cr.yaml",
	}

	namespaceBuildStrategies = []string{
		"samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3-heroku_namespaced_cr.yaml",
		"samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_namespaced_cr.yaml",
	}

	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 5

	operatorDeploymentRetryInterval = time.Second * 5
	operatorDeploymentTimeout       = time.Second * 120
)

func TestMain(m *testing.M) {
	err := configureOperatorSDKTestFramework()
	if err != nil {
		Logf("Failed to configure operator-sdk test framework: %s", err.Error())
		os.Exit(1)
	}

	framework.MainEntry(m)
}

func TestBuildRun(t *testing.T) {
	testingT = t
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	var err error

	operator := os.Getenv(EnvVarOperator)
	switch operator {
	case "start_local":
		Logf("Starting local operator")
		operatorCmd, err = startLocalOperator()
		Expect(err).ToNot(HaveOccurred(), "Failed to start local operator")
	case "managed_outside":
		Logf("Using operator that is started outside of the e2e test suite.")
	default:
		Fail("Unexpected value for " + EnvVarOperator + ": '" + operator + "'")
	}

	// add schemes to operator-sdk test framework
	Logf("Adding schemes to operator-sdk test framework")
	err = populateOperatorSDKTestFrameworkScheme()
	Expect(err).ToNot(HaveOccurred(), "Failed to add schemes to operator-sdk test framework")

	// setup the Operator SDK testing global context
	globalCtx = framework.NewContext(testingT)

	// initialize cluster resources
	Logf("Initializing cluster resources")
	err = globalCtx.InitializeClusterResources(cleanupOptions(globalCtx, cleanupTimeout, cleanupRetryInterval))
	Expect(err).ToNot(HaveOccurred(), "unable to initialize cluster resources")

	// get the namespace
	Logf("Getting the namespace")
	namespace, err := globalCtx.GetWatchNamespace()
	Expect(err).ToNot(HaveOccurred(), "unable to obtain namespace")
	Logf("Namespace is %s", namespace)

	f := framework.Global

	if operator != "managed_outside" {
		// wait for operator deployment
		Logf("Waiting for operator deployment")
		err = e2eutil.WaitForOperatorDeployment(
			testingT,
			f.KubeClient,
			// TODO we currently have no codepath where this is relevant, but this namespace is the wrong one
			// it is the watch namespace, but needs to be the operator namespace
			namespace,
			"shipwright-build-controller",
			1,
			operatorDeploymentRetryInterval,
			operatorDeploymentTimeout,
		)
		Expect(err).ToNot(HaveOccurred(), "error on waiting for operator deployment")
	}

	// create the pipeline service account
	Logf("Creating the pipeline service account")
	createPipelineServiceAccount(globalCtx, f, namespace, cleanupTimeout, cleanupRetryInterval)

	// create the container registry secret
	Logf("Creating the container registry secret")
	createContainerRegistrySecret(globalCtx, f, namespace, cleanupTimeout, cleanupRetryInterval)

	if os.Getenv(EnvVarCreateGlobalObjects) == "true" {
		// create cluster build strategies
		Logf("Creating cluster build strategies")
		for _, clusterBuildStrategy := range clusterBuildStrategies {
			Logf("Creating cluster build strategy %s", clusterBuildStrategy)
			cbs, err := clusterBuildStrategyTestData(clusterBuildStrategy)
			Expect(err).ToNot(HaveOccurred(), "Error retrieving cluster buildstrategy test data")
			cbs.SetNamespace(namespace)

			createClusterBuildStrategy(globalCtx, f, cbs, cleanupTimeout, cleanupRetryInterval)
		}

		// create namespace build strategies
		Logf("Creating namespace build strategies")
		for _, namespaceBuildStrategy := range namespaceBuildStrategies {
			Logf("Creating namespace build strategy %s", namespaceBuildStrategy)
			nbs, err := buildStrategyTestData(namespace, namespaceBuildStrategy)
			Expect(err).ToNot(HaveOccurred(), "Error retrieving build strategy test data")

			createNamespacedBuildStrategy(globalCtx, f, nbs, cleanupTimeout, cleanupRetryInterval)
		}
	} else {
		Logf("Build strategy creation skipped.")
	}

	return nil
}, func(data []byte) {
	// add schemes to operator-sdk test framework
	Logf("Adding schemes to operator-sdk test framework")
	err := populateOperatorSDKTestFrameworkScheme()
	Expect(err).ToNot(HaveOccurred(), "Failed to add schemes to operator-sdk test framework")

	// setup the operator-sdk test framework node context
	Logf("Creating node test context")
	ctx = framework.NewContext(testingT)
})

var _ = SynchronizedAfterSuite(func() {
	if ctx != nil {
		Logf("Cleaning up node context")
		ctx.Cleanup()
	}
}, func() {
	if globalCtx != nil {
		Logf("Cleaning up global context")
		globalCtx.Cleanup()
	}
	if operatorCmd != nil && operatorCmd.Process != nil {
		Logf("Log output from the local operator:")
		Logf("%v", operatorCmd.Stdout)

		Logf("Killing the local operator")
		operatorCmd.Process.Kill()
	}

	err := cleanupOperatorSDKTestFramework()
	Expect(err).ToNot(HaveOccurred(), "Failed to cleanup operator-sdk test framework")
})
