package e2e

import (
	goctx "context"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	operatorapis "github.com/redhat-developer/build/pkg/apis"
	operator "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	"github.com/stretchr/testify/require"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/scheme"
)

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

func validateController(
	t *testing.T,
	ctx *framework.TestCtx,
	f *framework.Framework,
	testBuild *operator.Build,
	modifySpec bool) {
	buildIdentifier := testBuild.GetName()
	namespace, _ := ctx.GetNamespace()

	err := f.Client.Create(goctx.TODO(), testBuild, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(5 * time.Second)

	//  Ensure that a Task has been created

	generatedTask := &pipelinev1.Task{}
	err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: buildIdentifier, Namespace: namespace}, generatedTask)
	require.NoError(t, err)
	require.NotNil(t, generatedTask)

	// Ensure that a TaskRun has been created and is in pending state

	generatedTaskRun := &pipelinev1.TaskRun{}
	err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: buildIdentifier, Namespace: namespace}, generatedTaskRun)
	require.NoError(t, err)
	require.NotNil(t, generatedTaskRun)

	// Ensure Build is in Pending state
	err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: buildIdentifier, Namespace: namespace}, testBuild)
	require.NoError(t, err)

	pendingOrRunning := false

	if testBuild.Status.Status == "Pending" || testBuild.Status.Status == "Running" {
		pendingOrRunning = true
	}
	require.True(t, pendingOrRunning)

	// Ensure that Build moves to Running State
	foundRunning := false
	for i := 1; i <= 10; i++ {
		time.Sleep(3 * time.Second)
		err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: buildIdentifier, Namespace: namespace}, testBuild)
		require.NoError(t, err)

		if testBuild.Status.Status == "Running" {
			foundRunning = true
			break
		}
	}
	require.True(t, foundRunning)

	if modifySpec {
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
		require.NoError(t, err)
		if testBuild.Status.Status == "Pending" || testBuild.Status.Status == "Running" {
			pendingOrRunning = true
		}
		require.True(t, pendingOrRunning)
	}

	// Ensure that eventually the Build moves to Succeeded.
	foundSuccessful := false
	for i := 1; i <= 5; i++ {
		time.Sleep(20 * time.Second)
		err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: buildIdentifier, Namespace: namespace}, testBuild)
		require.NoError(t, err)

		if testBuild.Status.Status == "Succeeded" {
			foundSuccessful = true
			break
		}
	}
	require.True(t, foundSuccessful)
}

// namespacedBuildStrategyTestData gets the us the BuildStrategy test data set up
func buildStrategyTestData(ns string, buildStrategyCRPath string) (*operator.BuildStrategy, error) {

	decode := scheme.Codecs.UniversalDeserializer().Decode
	operatorapis.AddToScheme(scheme.Scheme)

	yaml, err := ioutil.ReadFile(buildStrategyCRPath)
	if err != nil {
		fmt.Printf("%#v", err)
		return nil, err
	}

	obj, _, err := decode(yaml, nil, nil)
	if err != nil {
		fmt.Printf("%#v", err)
		return nil, err
	}
	buildStrategy := obj.(*operator.BuildStrategy)
	buildStrategy.SetNamespace(ns)
	return buildStrategy, err
}

// clusterBuildStrategyTestData gets the us the ClusterBuildStrategy test data set up
func clusterBuildStrategyTestData(buildStrategyCRPath string) (*operator.ClusterBuildStrategy, error) {

	decode := scheme.Codecs.UniversalDeserializer().Decode
	operatorapis.AddToScheme(scheme.Scheme)

	yaml, err := ioutil.ReadFile(buildStrategyCRPath)
	if err != nil {
		fmt.Printf("%#v", err)
		return nil, err
	}

	obj, _, err := decode(yaml, nil, nil)
	if err != nil {
		fmt.Printf("%#v", err)
		return nil, err
	}
	clusterBuildStrategy := obj.(*operator.ClusterBuildStrategy)
	return clusterBuildStrategy, err
}

// buildTestData gets the us the Build test data set up
func buildTestData(ns string, identifier string, buildCRPath string) (*operator.Build, error) {

	decode := scheme.Codecs.UniversalDeserializer().Decode
	operatorapis.AddToScheme(scheme.Scheme)

	yaml, err := ioutil.ReadFile(buildCRPath)
	if err != nil {
		fmt.Printf("%#v", err)
		return nil, err
	}

	obj, _, err := decode([]byte(yaml), nil, nil)
	if err != nil {
		fmt.Printf("%#v", err)
		return nil, err
	}
	build := obj.(*operator.Build)

	build.SetNamespace(ns)
	build.SetName(identifier)

	return build, err
}
