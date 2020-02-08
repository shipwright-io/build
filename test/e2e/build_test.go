package e2e

import (
	goctx "context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/redhat-developer/build/pkg/apis"
	operator "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	controller "github.com/redhat-developer/build/pkg/controller/build"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	retryInterval        = time.Second * 5
	timeout              = time.Second * 60
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 5
)

func TestBuild(t *testing.T) {
	buildInstance := &operator.Build{}
	err := framework.AddToFrameworkScheme(apis.AddToScheme, buildInstance)
	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}
	// run subtests
	t.Run("build-group", func(t *testing.T) {
		t.Run("Cluster", BuildCluster)
	})
}

func buildTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	buildImageString := "docker.io/centos/nodejs-10-centos7"

	// create exampleBuild custom resource
	exampleBuild := &operator.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-build",
			Namespace: namespace,
		},
		Spec: operator.BuildSpec{
			BuilderImage: &buildImageString,
			StrategyRef:  controller.StrategySourceToImage,
			OutputImage:  "image-registry.openshift-image-registry.svc:5000/sbose/nodejs-ex",
		},
	}
	err = f.Client.Create(goctx.TODO(), exampleBuild, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		return err
	}

	time.Sleep(3 * time.Second)

	err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: "example-build", Namespace: namespace}, exampleBuild)
	if err != nil {
		return err
	}

	if exampleBuild.Status.Status == "in-progress" {
		return nil
	}

	// TODO: Validate creation of tekton resources

	return errors.New("build status not available")
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

	// wait for operator to be ready
	err = e2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, "build-operator", 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}
}
