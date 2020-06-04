package e2e

import (
	goctx "context"
	"io/ioutil"
	"os"
	"time"

	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	operatorapis "github.com/redhat-developer/build/pkg/apis"
	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	operator "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/scheme"
)

const (
	EnvVarCreateGlobalObjects  = "TEST_E2E_CREATE_GLOBALOBJECTS"
	EnvVarOperator             = "TEST_E2E_OPERATOR"
	EnvVarServiceAccountName   = "TEST_E2E_SERVICEACCOUNT_NAME"
	EnvVarVerifyTektonObjects  = "TEST_E2E_VERIFY_TEKTONOBJECTS"
	EnvVarImageRepo            = "TEST_IMAGE_REPO"
	EnvVarEnablePrivateRepos   = "TEST_PRIVATE_REPO"
	EnvVarImageRepoSecret      = "TEST_IMAGE_REPO_SECRET"
	EnvVarSourceRepoSecretJSON = "TEST_IMAGE_REPO_DOCKERCONFIGJSON"
	EnvVarSourceURLGithub      = "TEST_PRIVATE_GITHUB"
	EnvVarSourceURLGitlab      = "TEST_PRIVATE_GITLAB"
	EnvVarSourceURLSecret      = "TEST_SOURCE_SECRET"
)

// cleanupOptions return a CleanupOptions instance.
func cleanupOptions(ctx *framework.TestCtx) *framework.CleanupOptions {
	return &framework.CleanupOptions{
		TestContext:   ctx,
		Timeout:       cleanupTimeout,
		RetryInterval: cleanupRetryInterval,
	}
}

// clientGet is a wrapper around f.Client.Get that performs retries in case of retryable errors
func clientGet(key types.NamespacedName, obj runtime.Object) error {
	f := framework.Global

	return wait.PollImmediate(4*time.Second, 60*time.Second, func() (bool, error) {
		if err := f.Client.Get(goctx.TODO(), key, obj); err != nil {
			// check if we have an error that we want to retry
			if apierrors.IsServerTimeout(err) || apierrors.IsTimeout(err) || apierrors.IsTooManyRequests(err) || err.Error() == "etcdserver: request timed out" {
				Logf("Error during client get is retried: '%s'", err.Error())
				return false, nil
			} else {
				Logf("Error during client get is not retried: '%s'", err.Error())
			}

			// return all other errors directly
			return true, err
		}

		// successful call
		return true, nil
	})
}

// createPipelineServiceAccount reads the TEST_E2E_SERVICEACCOUNT_NAME environment variable. If the value is "generated", then nothing is done.
// Otherwise it will create the service account. No error occurs if the service account already exists.
func createPipelineServiceAccount(ctx *framework.TestCtx, f *framework.Framework, namespace string) {
	serviceAccountName := os.Getenv(EnvVarServiceAccountName)

	if serviceAccountName == "generated" {
		Logf("Skipping creation of service account, generated one will be used per build run.")
	} else {
		serviceAccount := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      serviceAccountName,
			},
		}
		Logf("Creating '%s' service-account", serviceAccountName)
		err := f.Client.Create(goctx.TODO(), serviceAccount, cleanupOptions(ctx))
		if err != nil && !apierrors.IsAlreadyExists(err) {
			Expect(err).ToNot(HaveOccurred(), "Error creating service account")
		}
	}
}

// createContainerRegistrySecret use environment variables to check for container registry
// credentials secret, when not found a new secret is created.
func createContainerRegistrySecret(ctx *framework.TestCtx, f *framework.Framework, namespace string) {
	secretName := os.Getenv(EnvVarImageRepoSecret)
	secretPayload := os.Getenv(EnvVarSourceRepoSecretJSON)
	if secretName == "" || secretPayload == "" {
		Logf("Container registry secret won't be created.")
		return
	}

	secretNsName := types.NamespacedName{Namespace: namespace, Name: secretName}
	secret := &corev1.Secret{}
	if err := clientGet(secretNsName, secret); err == nil {
		Logf("Container registry secret is found at '%s/%s'", namespace, secretName)
		return
	}

	payload := []byte(secretPayload)
	secret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      secretName,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			".dockerconfigjson": payload,
		},
	}
	Logf("Creating container-registry secret '%s/%s' (%d bytes)", namespace, secretName, len(payload))
	err := f.Client.Create(goctx.TODO(), secret, cleanupOptions(ctx))
	Expect(err).ToNot(HaveOccurred(), "on creating container registry secret")
}

// createNamespacedBuildStrategy create a namespaced BuildStrategy.
func createNamespacedBuildStrategy(
	ctx *framework.TestCtx,
	f *framework.Framework,
	testBuildStrategy *operator.BuildStrategy,
) {
	err := f.Client.Create(goctx.TODO(), testBuildStrategy, cleanupOptions(ctx))
	if err != nil {
		Expect(err).NotTo(HaveOccurred())
	}
}

// createClusterBuildStrategy create ClusterBuildStrategy resource.
func createClusterBuildStrategy(
	ctx *framework.TestCtx,
	f *framework.Framework,
	testBuildStrategy *operator.ClusterBuildStrategy,
) {
	err := f.Client.Create(goctx.TODO(), testBuildStrategy, cleanupOptions(ctx))
	if err != nil && !apierrors.IsAlreadyExists(err) {
		Expect(err).NotTo(HaveOccurred())
	}
}

// validateBuildRunToSucceed creates the build run and watches its flow until it succeeds.
func validateBuildRunToSucceed(
	namespace string,
	testBuildRun *operator.BuildRun,
) {
	f := framework.Global

	pendingStatus := "Pending"
	runningStatus := "Running"
	trueCondition := v1.ConditionTrue
	pendingAndRunningStatues := []string{pendingStatus, runningStatus}

	// Ensure the BuildRun has been created
	err := f.Client.Create(goctx.TODO(), testBuildRun, cleanupOptions(ctx))
	Expect(err).ToNot(HaveOccurred(), "Failed to create build run.")

	// Ensure that a TaskRun has been created and is in pending or running state
	if os.Getenv(EnvVarVerifyTektonObjects) == "true" {
		Eventually(func() string {
			taskRun, err := getTaskRun(f, testBuildRun)
			if err != nil {
				Logf("Retrieving TaskRun error: '%s'", err)
				return ""
			}
			if taskRun == nil {
				Logf("TaskRun is not yet generated!")
				return ""
			}
			if len(taskRun.Status.Conditions) == 0 {
				Logf("TaskRun has not yet conditions.")
				return ""
			}
			return taskRun.Status.Conditions[0].Reason
		}, 300*time.Second, 5*time.Second).Should(BeElementOf(pendingAndRunningStatues), "TaskRun not pending or running")
	} else {
		Logf("TaskRun verification skipped.")
	}

	// Ensure BuildRun is in pending or running state
	buildRunNsName := types.NamespacedName{Name: testBuildRun.Name, Namespace: namespace}
	Eventually(func() string {
		err = clientGet(buildRunNsName, testBuildRun)
		if err != nil {
			Logf("Retrieving BuildRun error: '%s'", err)
			return ""
		}
		return testBuildRun.Status.Reason
	}, 30*time.Second, 2*time.Second).Should(BeElementOf(pendingAndRunningStatues), "BuildRun not pending or running")

	// Verify that the BuildSpec is available in the status
	Expect(testBuildRun.Status.BuildSpec).ToNot(BeNil())

	// Ensure that BuildRun moves to Running State
	Eventually(func() string {
		err = clientGet(buildRunNsName, testBuildRun)
		Expect(err).ToNot(HaveOccurred(), "Error retrieving build run")

		return testBuildRun.Status.Reason
	}, 180*time.Second, 3*time.Second).Should(Equal(runningStatus), "BuildRun not running")

	// Verify that the BuildSpec is still available in the status
	Expect(testBuildRun.Status.BuildSpec).ToNot(BeNil())

	// Ensure that eventually the BuildRun moves to Succeeded.
	Eventually(func() v1.ConditionStatus {
		err = clientGet(buildRunNsName, testBuildRun)
		Expect(err).ToNot(HaveOccurred(), "Error retrieving build run")

		return testBuildRun.Status.Succeeded
	}, 550*time.Second, 5*time.Second).Should(Equal(trueCondition), "BuildRun did not succeed")

	// Verify that the BuildSpec is still available in the status
	Expect(testBuildRun.Status.BuildSpec).ToNot(BeNil())

	Logf("Test build '%s' is completed after %v !", testBuildRun.GetName(), testBuildRun.Status.CompletionTime.Time.Sub(testBuildRun.Status.StartTime.Time))
}

// validateBuildRunToFail creates the build run and watches its flow until it fails
// and verifies the reason using a regular expression.
func validateBuildRunToFail(
	namespace string,
	testBuildRun *operator.BuildRun,
	expectedReasonRegexp string,
) {
	f := framework.Global
	falseCondition := v1.ConditionFalse

	// Create the BuildRun
	err := f.Client.Create(goctx.TODO(), testBuildRun, cleanupOptions(ctx))
	Expect(err).ToNot(HaveOccurred(), "Failed to create build run.")

	// Ensure that eventually the BuildRun moves to Failed.
	buildRunNsName := types.NamespacedName{Name: testBuildRun.Name, Namespace: namespace}
	Eventually(func() v1.ConditionStatus {
		err = clientGet(buildRunNsName, testBuildRun)
		Expect(err).ToNot(HaveOccurred(), "Error retrieving build run")

		return testBuildRun.Status.Succeeded
	}, 550*time.Second, 5*time.Second).Should(Equal(falseCondition), "BuildRun did not fail")

	// Verify that the BuildSpec is available in the status
	Expect(testBuildRun.Status.BuildSpec).ToNot(BeNil())

	// Verify the build run failure
	Expect(testBuildRun.Status.Reason).To(MatchRegexp(expectedReasonRegexp))
}

// validateBuildDeletion verifies if the BuildRun is deleted after Build is deleted.
func validateBuildDeletion(
	namespace string,
	testBuildName string,
	testBuildRun *operator.BuildRun,
	expectedDeletion bool,
) {
	f := framework.Global

	// Delete the Build
	buildNsName := types.NamespacedName{Name: testBuildName, Namespace: namespace}
	testBuild := &buildv1alpha1.Build{}
	err := clientGet(buildNsName, testBuild)
	Expect(err).ToNot(HaveOccurred(), "Build doesn't exist")
	err = f.Client.Delete(goctx.TODO(), testBuild)
	Expect(err).ToNot(HaveOccurred(), "Failed to delete build")
	Logf("Build is deleted!")

	Eventually(func() error {
		err = clientGet(buildNsName, testBuild)
		return err
	}, 30*time.Second, 3*time.Second).ShouldNot(BeNil(), "Build is not deleted yet")

	buildRunNsName := types.NamespacedName{Name: testBuildRun.Name, Namespace: namespace}
	err = clientGet(buildRunNsName, testBuildRun)
	if expectedDeletion {
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), "BuildRun was not deleted together with the Build")
	} else {
		Expect(err).ToNot(HaveOccurred(), "BuildRun was deleted together with the Build")
	}
}

// readAndDecode read file path and decode.
func readAndDecode(filePath string) (runtime.Object, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	err := operatorapis.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}

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
	rootDir, err := getRootDir()
	if err != nil {
		return nil, err
	}

	obj, err := readAndDecode(rootDir + "/" + buildRunCRPath)
	if err != nil {
		return nil, err
	}

	buildRun := obj.(*operator.BuildRun)
	buildRun.SetNamespace(ns)
	buildRun.SetName(identifier)
	buildRun.Spec.BuildRef.Name = identifier

	serviceAccountName := os.Getenv(EnvVarServiceAccountName)

	if serviceAccountName == "generated" {
		buildRun.Spec.ServiceAccount = &buildv1alpha1.ServiceAccount{
			Generate: true,
		}
	} else {
		buildRun.Spec.ServiceAccount = &buildv1alpha1.ServiceAccount{
			Name: &serviceAccountName,
		}
	}

	return buildRun, err
}

// getTaskRun retrieve Tekton's Task based on BuildRun instance.
func getTaskRun(
	f *framework.Framework,
	buildRun *buildv1alpha1.BuildRun,
) (*v1beta1.TaskRun, error) {
	taskRunList := &v1beta1.TaskRunList{}
	lbls := map[string]string{
		buildv1alpha1.LabelBuild:    buildRun.Spec.BuildRef.Name,
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
