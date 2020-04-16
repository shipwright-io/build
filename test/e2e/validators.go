package e2e

import (
	goctx "context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	operatorapis "github.com/redhat-developer/build/pkg/apis"
	operator "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	"github.com/stretchr/testify/require"

	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	taskv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/scheme"
)

var (
	EnvVarImageRepo          = "TEST_IMAGE_REPO"
	EnvVarEnablePrivateRepos = "TEST_PRIVATE_REPO"
	EnvVarImageRepoSecret    = "TEST_IMAGE_REPO_SECRET"
	EnvVarSourceURLGithub    = "TEST_PRIVATE_GITHUB"
	EnvVarSourceURLGitlab    = "TEST_PRIVATE_GITLAB"
	EnvVarSourceURLSecret    = "TEST_SOURCE_SECRET"
)

// cleanupOptions return a CleanupOptions instance.
func cleanupOptions(ctx *framework.TestCtx) *framework.CleanupOptions {
	return &framework.CleanupOptions{
		TestContext:   ctx,
		Timeout:       cleanupTimeout,
		RetryInterval: cleanupRetryInterval,
	}
}

// createContainerRegistrySecret create a secret type DockerConfigJSON to store registry credentials.
func createContainerRegistrySecret(t *testing.T, ctx *framework.TestCtx, f *framework.Framework) {
	if os.Getenv(EnvVarImageRepoSecret) == "" {
		t.Logf("Environment variable with container registry secret is not present.")
		return
	}

	ns, err := ctx.GetNamespace()
	require.NoError(t, err, "unable to obtain test namespace")

	payload := []byte(os.Getenv(EnvVarImageRepoSecret))
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      SecretName,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			".dockerconfigjson": payload,
		},
	}

	t.Logf("Creating container-registry secret '%s/%s'", ns, SecretName)
	err = f.Client.Create(goctx.TODO(), secret, cleanupOptions(ctx))
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		t.Fatal(err)
	}
}

// createNamespacedBuildStrategy create a namespaced BuildStrategy.
func createNamespacedBuildStrategy(
	t *testing.T,
	ctx *framework.TestCtx,
	f *framework.Framework,
	testBuildStrategy *operator.BuildStrategy,
) {
	err := f.Client.Create(goctx.TODO(), testBuildStrategy, cleanupOptions(ctx))
	if err != nil {
		t.Fatal(err)
	}
}

// createClusterBuildStrategy create ClusterBuildStrategy resource.
func createClusterBuildStrategy(
	t *testing.T,
	ctx *framework.TestCtx,
	f *framework.Framework,
	testBuildStrategy *operator.ClusterBuildStrategy,
) {
	err := f.Client.Create(goctx.TODO(), testBuildStrategy, cleanupOptions(ctx))
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		t.Fatal(err)
	}
}

// validateController create and watch the build flow happening, probing each step for a image
// successfully created.
func validateController(
	t *testing.T,
	ctx *framework.TestCtx,
	f *framework.Framework,
	testBuild *operator.Build,
	testBuildRun *operator.BuildRun,
) {
	ns, _ := ctx.GetNamespace()
	pendingStatus := "Pending"
	runningStatus := "Running"
	trueCondition := v1.ConditionTrue
	pendingAndRunningStatues := []string{pendingStatus, runningStatus}

	// Ensure the Build has been created
	err := f.Client.Create(goctx.TODO(), testBuild, cleanupOptions(ctx))
	require.NoError(t, err)

	// Ensure the BuildRun has been created
	err = f.Client.Create(goctx.TODO(), testBuildRun, cleanupOptions(ctx))
	require.NoError(t, err)

	time.Sleep(15 * time.Second)

	// Ensure that a TaskRun has been created and is in pending or running state
	generatedTaskRun, err := getTaskRun(f, testBuild, testBuildRun)
	require.NoError(t, err)
	conditionReason := generatedTaskRun.Status.Conditions[0].Reason
	require.Contains(t, pendingAndRunningStatues, conditionReason, "TaskRun not pending or running")

	// Ensure BuildRun is in pending or running state
	buildRunNsName := types.NamespacedName{Name: testBuildRun.Name, Namespace: ns}
	err = f.Client.Get(goctx.TODO(), buildRunNsName, testBuildRun)
	require.NoError(t, err)
	reason := testBuildRun.Status.Reason
	require.Contains(t, pendingAndRunningStatues, reason, "BuildRun not pending or running")

	// Ensure that Build moves to Running State
	require.Eventually(t, func() bool {
		err = f.Client.Get(goctx.TODO(), buildRunNsName, testBuildRun)
		require.NoError(t, err)

		return testBuildRun.Status.Reason == runningStatus
	}, 180*time.Second, 3*time.Second, "BuildRun not running")

	// Ensure that eventually the Build moves to Succeeded.
	require.Eventually(t, func() bool {
		err = f.Client.Get(goctx.TODO(), buildRunNsName, testBuildRun)
		require.NoError(t, err)

		return testBuildRun.Status.Succeeded == trueCondition
	}, 600*time.Second, 5*time.Second, "BuildRun not succeeded")
}

// readAndDecode read file path and decode.
func readAndDecode(filePath string) (runtime.Object, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	operatorapis.AddToScheme(scheme.Scheme)

	payload, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	obj, _, err := decode([]byte(payload), nil, nil)
	return obj, err
}

// buildStrategyTestData gets the us the BuildStrategy test data set up
func buildStrategyTestData(ns string, buildStrategyCRPath string) (*operator.BuildStrategy, error) {
	obj, err := readAndDecode(buildStrategyCRPath)
	if err != nil {
		return nil, err
	}

	buildStrategy := obj.(*operator.BuildStrategy)
	buildStrategy.SetNamespace(ns)

	return buildStrategy, err
}

// clusterBuildStrategyTestData gets the us the ClusterBuildStrategy test data set up
func clusterBuildStrategyTestData(buildStrategyCRPath string) (*operator.ClusterBuildStrategy, error) {
	obj, err := readAndDecode(buildStrategyCRPath)
	if err != nil {
		return nil, err
	}

	clusterBuildStrategy := obj.(*operator.ClusterBuildStrategy)
	return clusterBuildStrategy, err
}

// buildTestData gets the us the Build test data set up
func buildTestData(ns string, identifier string, buildCRPath string) (*operator.Build, error) {
	obj, err := readAndDecode(buildCRPath)
	if err != nil {
		return nil, err
	}

	build := obj.(*operator.Build)
	build.SetNamespace(ns)
	build.SetName(identifier)
	return build, err
}

// buildTestData gets the us the Build test data set up
func buildRunTestData(ns string, identifier string, buildRunCRPath string) (*operator.BuildRun, error) {
	obj, err := readAndDecode(buildRunCRPath)
	if err != nil {
		return nil, err
	}

	buildRun := obj.(*operator.BuildRun)
	buildRun.SetNamespace(ns)
	buildRun.SetName(identifier)
	buildRun.Spec.BuildRef.Name = identifier
	return buildRun, err
}

// getTaskRun retrieve Tekton's Task based on BuildRun instance.
func getTaskRun(
	f *framework.Framework,
	build *buildv1alpha1.Build,
	buildRun *buildv1alpha1.BuildRun,
) (*taskv1.TaskRun, error) {
	taskRunList := &taskv1.TaskRunList{}

	lbls := map[string]string{
		buildv1alpha1.LabelBuild:    build.Name,
		buildv1alpha1.LabelBuildRun: buildRun.Name,
	}
	opts := client.ListOptions{
		Namespace:     buildRun.Namespace,
		LabelSelector: labels.SelectorFromSet(lbls),
	}
	err := f.Client.List(goctx.TODO(), taskRunList, &opts)

	if err != nil {
		return nil, err
	}

	if len(taskRunList.Items) > 0 {
		return &taskRunList.Items[len(taskRunList.Items)-1], nil
	}
	return nil, nil
}
