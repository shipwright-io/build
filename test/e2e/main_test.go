package e2e

import (
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/redhat-developer/build/pkg/apis"
	operator "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	"github.com/stretchr/testify/require"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
)

func TestMain(m *testing.M) {
	framework.MainEntry(m)
}

var (
	retryInterval         = time.Second * 5
	timeout               = time.Second * 120
	cleanupRetryInterval  = time.Second * 1
	cleanupTimeout        = time.Second * 5
	EnvVarImageRepo       = "TEST_IMAGE_REPO"
	EnvVarImageRepoSecret = "TEST_IMAGE_REPO_SECRET"
)

func TestBuild(t *testing.T) {
	buildList := &operator.BuildList{}
	err := framework.AddToFrameworkScheme(apis.AddToScheme, buildList)
	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}

	err = framework.AddToFrameworkScheme(apis.AddToScheme, &operator.BuildStrategyList{})
	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}

	err = framework.AddToFrameworkScheme(apis.AddToScheme, &operator.ClusterBuildStrategyList{})
	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}

	err = framework.AddToFrameworkScheme(pipelinev1.AddToScheme, &pipelinev1.TaskList{})
	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}

	err = framework.AddToFrameworkScheme(pipelinev1.AddToScheme, &pipelinev1.TaskRunList{})
	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}

	// run subtests
	t.Run("build-group", func(t *testing.T) {
		t.Run("Build_e2e_tests", BuildCluster)
	})
}

func BuildCluster(t *testing.T) {
	t.Parallel()
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()
	err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatalf("failed to initialize cluster resources: %v", err)
	}
	t.Log("Initialized cluster resources")
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}
	// get global framework variables
	f := framework.Global
	// wait for build-operator to be ready
	err = e2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, "build-operator", 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	// Run e2e tests for kaniko
	oE := newOperatorEmulation(namespace,
		"example-build-kaniko",
		"samples/buildstrategy/kaniko/buildstrategy_kaniko_cr.yaml",
		"samples/build/build_kaniko_cr.yaml",
		"samples/buildrun/buildrun_kaniko_cr.yaml",
	)
	err = BuildTestData(oE)
	require.NoError(t, err)
	createClusterBuildStrategy(t, ctx, f, oE.clusterBuildStrategy)
	validateController(t, ctx, f, oE.build, oE.buildRun)

	// Run e2e tests for source2image
	oE = newOperatorEmulation(namespace,
		"example-build-s2i",
		"samples/buildstrategy/source-to-image/buildstrategy_source-to-image_cr.yaml",
		"samples/build/build_source-to-image_cr.yaml",
		"samples/buildrun/buildrun_source-to-image_cr.yaml",
	)
	err = BuildTestData(oE)
	require.NoError(t, err)
	createClusterBuildStrategy(t, ctx, f, oE.clusterBuildStrategy)
	validateController(t, ctx, f, oE.build, oE.buildRun)

	// Run e2e tests for buildah
	oE = newOperatorEmulation(namespace,
		"example-build-buildah",
		"samples/buildstrategy/buildah/buildstrategy_buildah_cr.yaml",
		"samples/build/build_buildah_cr.yaml",
		"samples/buildrun/buildrun_buildah_cr.yaml",
	)
	err = BuildTestData(oE)
	require.NoError(t, err)
	createClusterBuildStrategy(t, ctx, f, oE.clusterBuildStrategy)
	validateController(t, ctx, f, oE.build, oE.buildRun)

	// Run e2e tests for buidlpacks v3
	oE = newOperatorEmulation(namespace,
		"example-build-buildpacks-v3",
		"samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_cr.yaml",
		"samples/build/build_buildpacks-v3_cr.yaml",
		"samples/buildrun/buildrun_buildpacks-v3_cr.yaml",
	)
	err = BuildTestData(oE)
	require.NoError(t, err)
	createClusterBuildStrategy(t, ctx, f, oE.clusterBuildStrategy)
	validateController(t, ctx, f, oE.build, oE.buildRun)

	// Run e2e tests for buildpacks v3 with a namespaced scope
	oE = newOperatorEmulation(namespace,
		"example-build-buildpacks-v3-namespaced",
		"samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_namespaced_cr.yaml",
		"samples/build/build_buildpacks-v3_namespaced_cr.yaml",
		"samples/buildrun/buildrun_buildpacks-v3_namespaced_cr.yaml",
	)
	err = BuildTestData(oE)
	require.NoError(t, err)

	oE.buildStrategy.SetNamespace(namespace)
	createNamespacedBuildStrategy(t, ctx, f, oE.buildStrategy)
	validateController(t, ctx, f, oE.build, oE.buildRun)

}
