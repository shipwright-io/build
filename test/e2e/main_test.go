package e2e

import (
	goctx "context"
	"fmt"
	"os"
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/redhat-developer/build/pkg/apis"
	operator "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	"github.com/stretchr/testify/assert"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
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
		t.Run("Buildah test with modified Build", BuildCluster)
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

	buildIdentifier := "example-build-buildah"
	testBuild, testBuildStrategy := buildahBuildTestData(namespace, buildIdentifier) //buildahBuildTestData(namespace, buildIdentifier)
	validateControllerReconcileWithModifiedSpec(t, ctx, f, testBuild, testBuildStrategy)

	buildIdentifier = "example-build-buildpacks-v3"
	testBuild, testBuildStrategy = buildpackBuildTestData(namespace, buildIdentifier)

	if os.Getenv(EnvVarImageRepo) != "" && os.Getenv(EnvVarImageRepoSecret) != "" {
		// need to parameterize for test environments
		testBuild.Spec.Output = operator.Output{
			ImageURL: os.Getenv(EnvVarImageRepo),
			SecretRef: &v1.LocalObjectReference{
				Name: os.Getenv(EnvVarImageRepoSecret),
			},
		}
		validateController(t, ctx, f, testBuild, testBuildStrategy)
	}

}

func validateControllerReconcileWithModifiedSpec(t *testing.T,
	ctx *framework.TestCtx,
	f *framework.Framework,
	testBuild *operator.Build,
	testBuildStrategy *operator.BuildStrategy,
) {

	buildIdentifier := testBuild.GetName()
	namespace, _ := ctx.GetNamespace()

	err := f.Client.Create(goctx.TODO(), testBuildStrategy, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	err = f.Client.Create(goctx.TODO(), testBuild, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(5 * time.Second)

	//  Ensure that a Task has been created

	generatedTask := &pipelinev1.Task{}
	err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: buildIdentifier, Namespace: namespace}, generatedTask)
	assert.NoError(t, err)
	assert.NotNil(t, generatedTask)

	// Ensure that a TaskRun has been created and is in pending state

	generatedTaskRun := &pipelinev1.TaskRun{}
	err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: buildIdentifier, Namespace: namespace}, generatedTaskRun)
	assert.NoError(t, err)
	assert.NotNil(t, generatedTaskRun)
	//assert.Equal(t, "Pending", generatedTaskRun.Status.Conditions[0].Reason)

	// Ensure Build is in Pending state
	err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: buildIdentifier, Namespace: namespace}, testBuild)
	assert.NoError(t, err)
	assert.Equal(t, "Pending", testBuild.Status.Status)

	// Ensure that Build moves to Running State
	foundRunning := false
	for i := 1; i <= 10; i++ {
		time.Sleep(3 * time.Second)
		err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: buildIdentifier, Namespace: namespace}, testBuild)
		assert.NoError(t, err)

		if testBuild.Status.Status == "Running" {
			foundRunning = true
			break
		}
	}
	assert.True(t, foundRunning)

	// Instead of letting it go to Succeeded, let's update the spec a bit.
	// These trigger deletion of existing Task[Run]

	testBuild.Spec.Output.ImageURL = fmt.Sprintf("image-registry.openshift-image-registry.svc:5000/%s/foo", namespace)
	err = f.Client.Update(goctx.TODO(), testBuild)
	if err != nil {
		t.Fatal(err)
	}

	// Ensure Build is BACK TO Pending state

	time.Sleep(5 * time.Second)
	err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: buildIdentifier, Namespace: namespace}, testBuild)
	assert.NoError(t, err)
	assert.Equal(t, "Pending", testBuild.Status.Status)

	// Ensure that eventually the Build moves to Succeeded.
	foundSuccessful := false
	for i := 1; i <= 5; i++ {
		time.Sleep(20 * time.Second)
		err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: buildIdentifier, Namespace: namespace}, testBuild)
		assert.NoError(t, err)

		if testBuild.Status.Status == "Succeeded" {
			foundSuccessful = true
			break
		}
	}
	assert.True(t, foundSuccessful)
}

func validateController(t *testing.T,
	ctx *framework.TestCtx,
	f *framework.Framework,
	testBuild *operator.Build,
	testBuildStrategy *operator.BuildStrategy,
) {

	buildIdentifier := testBuild.GetName()
	namespace, _ := ctx.GetNamespace()

	err := f.Client.Create(goctx.TODO(), testBuildStrategy, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	err = f.Client.Create(goctx.TODO(), testBuild, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(5 * time.Second)

	//  Ensure that a Task has been created

	generatedTask := &pipelinev1.Task{}
	err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: buildIdentifier, Namespace: namespace}, generatedTask)
	assert.NoError(t, err)
	assert.NotNil(t, generatedTask)

	// Ensure that a TaskRun has been created and is in pending state

	generatedTaskRun := &pipelinev1.TaskRun{}
	err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: buildIdentifier, Namespace: namespace}, generatedTaskRun)
	assert.NoError(t, err)
	assert.NotNil(t, generatedTaskRun)
	//assert.Equal(t, "Pending", generatedTaskRun.Status.Conditions[0].Reason)

	// Ensure Build is in Pending state
	err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: buildIdentifier, Namespace: namespace}, testBuild)
	assert.NoError(t, err)
	assert.Equal(t, "Pending", testBuild.Status.Status)

	// Ensure that Build moves to Running State
	foundRunning := false
	for i := 1; i <= 10; i++ {
		time.Sleep(3 * time.Second)
		err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: buildIdentifier, Namespace: namespace}, testBuild)
		assert.NoError(t, err)

		if testBuild.Status.Status == "Running" {
			foundRunning = true
			break
		}
	}
	assert.True(t, foundRunning)

	// Ensure that eventually the Build moves to Succeeded.
	foundSuccessful := false
	for i := 1; i <= 5; i++ {
		time.Sleep(20 * time.Second)
		err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: buildIdentifier, Namespace: namespace}, testBuild)
		assert.NoError(t, err)

		if testBuild.Status.Status == "Succeeded" {
			foundSuccessful = true
			break
		}
	}
	assert.True(t, foundSuccessful)
}
