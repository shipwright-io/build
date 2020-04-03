package e2e

import (
	goctx "context"
	"errors"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	operatorapis "github.com/redhat-developer/build/pkg/apis"
	operator "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	"github.com/stretchr/testify/require"

	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	taskv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/scheme"
)

var (
	EnvVarImageRepo       = "TEST_IMAGE_REPO"
	EnvVarImageRepoSecret = "TEST_IMAGE_REPO_SECRET"
	EnvVarSourceURLGithub = "TEST_PRIVATE_GITHUB"
	EnvVarSourceURLGitlab = "TEST_PRIVATE_GITLAB"
	EnvVarSourURLSecret   = "TEST_SOURCE_SECRET"
)

// OperatorEmulation is used as an struct
// to hold required data
type OperatorEmulation struct {
	buildRun                *operator.BuildRun
	clusterBuildStrategy    *operator.ClusterBuildStrategy
	buildStrategy           *operator.BuildStrategy
	build                   *operator.Build
	namespace               string
	identifier              string
	buildStrategySamplePath string
	buildSamplePath         string
	buildRunSamplePath      string
}

func newOperatorEmulation(n string, id string, bSPath string, bPath string, bRPath string) *OperatorEmulation {
	return &OperatorEmulation{
		buildRun:                &operator.BuildRun{},
		clusterBuildStrategy:    &operator.ClusterBuildStrategy{},
		buildStrategy:           &operator.BuildStrategy{},
		build:                   &operator.Build{},
		namespace:               n,
		identifier:              id,
		buildStrategySamplePath: bSPath,
		buildSamplePath:         bPath,
		buildRunSamplePath:      bRPath,
	}

}

func createNamespacedBuildStrategy(
	t *testing.T,
	ctx *framework.TestCtx,
	f *framework.Framework,
	testBuildStrategy *operator.BuildStrategy) {
	err := f.Client.Create(goctx.TODO(), testBuildStrategy, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}
}

func createClusterBuildStrategy(
	t *testing.T,
	ctx *framework.TestCtx,
	f *framework.Framework,
	testBuildStrategy *operator.ClusterBuildStrategy) {
	err := f.Client.Create(goctx.TODO(), testBuildStrategy, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}
}

func deleteClusterBuildStrategy(
	t *testing.T,
	f *framework.Framework,
	testBuildStrategy *operator.ClusterBuildStrategy) {
	err := f.Client.Delete(goctx.TODO(), testBuildStrategy)
	if err != nil {
		t.Fatal(err)
	}
}

func deleteBuildStrategy(
	t *testing.T,
	f *framework.Framework,
	testBuildStrategy *operator.BuildStrategy) {
	err := f.Client.Delete(goctx.TODO(), testBuildStrategy)
	if err != nil {
		t.Fatal(err)
	}
}

func validateController(
	t *testing.T,
	ctx *framework.TestCtx,
	f *framework.Framework,
	testBuild *operator.Build,
	testBuildRun *operator.BuildRun) {
	namespace, _ := ctx.GetNamespace()
	pendingStatus := "Pending"
	runningStatus := "Running"
	trueCondition := v1.ConditionTrue

	// Ensure the Build has been created
	err := f.Client.Create(goctx.TODO(), testBuild, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	// Ensure the BuildRun has been created
	err = f.Client.Create(goctx.TODO(), testBuildRun, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(15 * time.Second)

	// Ensure that a TaskRun has been created and is in pending or running state
	pendingOrRunning := false
	generatedTaskRun, err := getTaskRun(testBuild, testBuildRun, f)
	require.NoError(t, err)
	pendingOrRunning = false
	if generatedTaskRun.Status.Conditions[0].Reason == pendingStatus || generatedTaskRun.Status.Conditions[0].Reason == runningStatus {
		pendingOrRunning = true
	}
	require.True(t, pendingOrRunning)

	// Ensure BuildRun is in pending or running state
	err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: testBuildRun.Name, Namespace: namespace}, testBuildRun)
	require.NoError(t, err)
	pendingOrRunning = false
	if testBuildRun.Status.Reason == pendingStatus || testBuildRun.Status.Reason == runningStatus {
		pendingOrRunning = true
	}
	require.True(t, pendingOrRunning)

	// Ensure that Build moves to Running State
	foundRunning := false
	for i := 1; i <= 10; i++ {
		time.Sleep(3 * time.Second)
		err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: testBuildRun.Name, Namespace: namespace}, testBuildRun)
		require.NoError(t, err)

		if testBuildRun.Status.Reason == runningStatus {
			foundRunning = true
			break
		}
	}
	require.True(t, foundRunning)

	// Ensure that eventually the Build moves to Succeeded.
	foundSuccessful := false
	for i := 1; i <= 30; i++ {
		time.Sleep(10 * time.Second)
		err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: testBuildRun.Name, Namespace: namespace}, testBuildRun)
		require.NoError(t, err)

		if testBuildRun.Status.Succeeded == trueCondition {
			foundSuccessful = true
			break
		}
	}
	require.True(t, foundSuccessful)
}

// BuildTestData loads all different Build objects
// into the OperatorEmulation structure
func BuildTestData(oE *OperatorEmulation) error {

	// Load the ClusterBuildStrategy sample into the OperatorEmulation.clusterBuildS field
	if err := oE.LoadBuildSamples(oE.buildStrategySamplePath); err != nil {
		return err
	}

	// Load the Build sample into the OperatorEmulation.build field
	if err := oE.LoadBuildSamples(oE.buildSamplePath); err != nil {
		return err
	}

	// Load the BuildRun sample into the OperatorEmulation.buildRun field
	if err := oE.LoadBuildSamples(oE.buildRunSamplePath); err != nil {
		return err
	}
	return nil
}

// LoadBuildSamples populates Build objects depending on the
// object type
func (os *OperatorEmulation) LoadBuildSamples(buildStrategySample string) error {

	decode := scheme.Codecs.UniversalDeserializer().Decode
	operatorapis.AddToScheme(scheme.Scheme)

	y, err := ioutil.ReadFile(buildStrategySample)
	if err != nil {
		return err
	}

	obj, _, err := decode([]byte(y), nil, nil)
	if err != nil {
		return err
	}

	switch object := obj.(type) {
	case *operator.Build:
		os.build = object
		os.build.SetNamespace(os.namespace)
		os.build.SetName(os.identifier)
		return nil
	case *operator.BuildRun:
		os.buildRun = object
		os.buildRun.SetNamespace(os.namespace)
		os.buildRun.SetName(os.identifier)
		os.buildRun.Spec.BuildRef.Name = os.identifier
		return nil
	case *operator.ClusterBuildStrategy:
		os.clusterBuildStrategy = object
		return nil
	case *operator.BuildStrategy:
		os.buildStrategy = object
		return nil
	default:
		return errors.New("none build strategy identified")
	}
}

// validateOutputEnvVars looks for known environment variables
// in order to modify on the fly specific Build object specs:
// - Spec.Output.ImageURL
// - Spec.Output.SecretRef
func validateOutputEnvVars(o *operator.Build) {

	// Get TEST_IMAGE_REPO env variable
	if val, bool := os.LookupEnv(EnvVarImageRepo); bool {
		o.Spec.Output.ImageURL = val
	}

	// Get TEST_IMAGE_REPO_SECRET env variable
	if s, bool := os.LookupEnv(EnvVarImageRepoSecret); bool {
		o.Spec.Output.SecretRef = &v1.LocalObjectReference{
			Name: s,
		}
	}
}

func validateGithubURL(o *operator.Build) {
	if val, bool := os.LookupEnv(EnvVarSourceURLGithub); bool {
		o.Spec.Source.URL = val
	}
}

func validateGitlabURL(o *operator.Build) {
	if val, bool := os.LookupEnv(EnvVarSourceURLGitlab); bool {
		o.Spec.Source.URL = val
	}
}

func validateSourceSecretRef(o *operator.Build) {
	// Get TEST_SOURCE_SECRET env variable
	if s, bool := os.LookupEnv(EnvVarSourURLSecret); bool {
		o.Spec.Source.SecretRef = &v1.LocalObjectReference{
			Name: s,
		}
	}
}

func getTaskRun(build *buildv1alpha1.Build, buildRun *buildv1alpha1.BuildRun, f *framework.Framework) (*taskv1.TaskRun, error) {

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
