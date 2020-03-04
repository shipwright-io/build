package e2e

import (
	"fmt"
	"os"
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/redhat-developer/build/pkg/apis"
	operator "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	"github.com/stretchr/testify/require"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

func TestMain(m *testing.M) {
	framework.MainEntry(m)
}

var (
	retryInterval         = time.Second * 5
	timeout               = time.Second * 60
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
		t.Run("Buildah and Buildpack", BuildCluster)
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

	outputImage := fmt.Sprintf("image-registry.openshift-image-registry.svc:5000/%s/foo", namespace)

	buildIdentifier := "example-build-kaniko"
	testBuild, testBuildStrategy, err := kanikoBuildTestData(namespace, buildIdentifier)
	require.NoError(t, err)
	testBuild.Spec.Output.ImageURL = outputImage

	validateController(t, ctx, f, testBuild, testBuildStrategy)

	buildIdentifier = "example-build-s2i"
	testBuild, testBuildStrategy, err = s2iBuildTestData(namespace, buildIdentifier)
	require.NoError(t, err)
	testBuild.Spec.Output.ImageURL = outputImage

	validateController(t, ctx, f, testBuild, testBuildStrategy)

	buildIdentifier = "example-build-buildah"
	testBuild, testBuildStrategy, err = buildahBuildTestData(namespace, buildIdentifier)
	require.NoError(t, err)
	testBuild.Spec.Output.ImageURL = outputImage

	validateControllerReconcileWithModifiedSpec(t, ctx, f, testBuild, testBuildStrategy)

	buildIdentifier = "example-build-buildpacks-v3"
	testBuild, testBuildStrategy, err = buildpackBuildTestData(namespace, buildIdentifier)
	require.NoError(t, err)

	if os.Getenv(EnvVarImageRepo) != "" && os.Getenv(EnvVarImageRepoSecret) != "" {
		testBuild.Spec.Output = operator.Image{
			ImageURL: os.Getenv(EnvVarImageRepo),
			SecretRef: &v1.LocalObjectReference{
				Name: os.Getenv(EnvVarImageRepoSecret),
			},
		}
		validateController(t, ctx, f, testBuild, testBuildStrategy)
	}

}
