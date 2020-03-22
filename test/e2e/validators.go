package e2e

import (
	goctx "context"
	"fmt"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
	"time"

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
	testBuildRun *operator.BuildRun,
	modifySpec bool) {
	namespace, _ := ctx.GetNamespace()
	pendingStatus := "Pending"
	runningStatus := "Running"
	trueCondition := v1.ConditionTrue

	// Modify the Output image URL for testing
	if modifySpec {
		testBuild.Spec.Output.ImageURL = fmt.Sprintf("image-registry.openshift-image-registry.svc:5000/%s/foo", namespace)
	}

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
	for i := 1; i <= 6; i++ {
		time.Sleep(20 * time.Second)
		err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: testBuildRun.Name, Namespace: namespace}, testBuildRun)
		require.NoError(t, err)

		if testBuildRun.Status.Succeeded == trueCondition {
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

// buildTestData gets the us the Build test data set up
func buildRunTestData(ns string, identifier string, buildRunCRPath string) (*operator.BuildRun, error) {

	decode := scheme.Codecs.UniversalDeserializer().Decode
	operatorapis.AddToScheme(scheme.Scheme)

	yaml, err := ioutil.ReadFile(buildRunCRPath)
	if err != nil {
		fmt.Printf("%#v", err)
		return nil, err
	}

	obj, _, err := decode([]byte(yaml), nil, nil)
	if err != nil {
		fmt.Printf("%#v", err)
		return nil, err
	}
	buildRun := obj.(*operator.BuildRun)

	buildRun.SetNamespace(ns)
	buildRun.SetName(identifier)
	buildRun.Spec.BuildRef.Name = identifier

	return buildRun, err
}

func getTaskRun(build *buildv1alpha1.Build, buildRun *buildv1alpha1.BuildRun, f *framework.Framework) (*taskv1.TaskRun, error) {

	taskRunList := &taskv1.TaskRunList{}

	lbls := map[string]string{
		buildv1alpha1.LabelBuild: build.Name,
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
