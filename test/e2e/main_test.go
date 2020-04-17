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
	retryInterval        = time.Second * 5
	timeout              = time.Second * 120
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 5
)

func TestBuild(t *testing.T) {
	err := framework.AddToFrameworkScheme(apis.AddToScheme, &operator.BuildList{})
	require.NoError(t, err, "unable to add BuildList to scheme")

	err = framework.AddToFrameworkScheme(apis.AddToScheme, &operator.BuildStrategyList{})
	require.NoError(t, err, "unable to add BuildStrategyList to scheme")

	err = framework.AddToFrameworkScheme(apis.AddToScheme, &operator.ClusterBuildStrategyList{})
	require.NoError(t, err, "unable to add ClusterBuildStrategyList to scheme")

	err = framework.AddToFrameworkScheme(pipelinev1.AddToScheme, &pipelinev1.TaskList{})
	require.NoError(t, err, "unable to add TaskList to scheme")

	err = framework.AddToFrameworkScheme(pipelinev1.AddToScheme, &pipelinev1.TaskRunList{})
	require.NoError(t, err, "unable to add TaskRunList to scheme")

	prepareClusterAndTest(t)
}

func prepareClusterAndTest(t *testing.T) {
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()

	t.Log("Initializing cluster resources...")
	err := ctx.InitializeClusterResources(cleanupOptions(ctx))
	require.NoError(t, err, "unable to initialize cluster resources")

	ns, err := ctx.GetNamespace()
	require.NoError(t, err, "unable to obtain namespace")

	f := framework.Global
	err = e2eutil.WaitForOperatorDeployment(
		t,
		f.KubeClient,
		ns,
		"build-operator",
		1,
		retryInterval,
		timeout,
	)
	require.NoError(t, err, "error on waiting for operator deployment")

	OperatorTests(t, ctx, f)
}
