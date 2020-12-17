// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	goctx "context"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/gomega"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorapis "github.com/shipwright-io/build/pkg/apis"
	operator "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

const (
	EnvVarCreateGlobalObjects  = "TEST_E2E_CREATE_GLOBALOBJECTS"
	EnvVarOperator             = "TEST_E2E_OPERATOR"
	EnvVarServiceAccountName   = "TEST_E2E_SERVICEACCOUNT_NAME"
	EnvVarVerifyTektonObjects  = "TEST_E2E_VERIFY_TEKTONOBJECTS"
	EnvVarTimeoutMultiplier    = "TEST_E2E_TIMEOUT_MULTIPLIER"
	EnvVarImageRepo            = "TEST_IMAGE_REPO"
	EnvVarEnablePrivateRepos   = "TEST_PRIVATE_REPO"
	EnvVarImageRepoSecret      = "TEST_IMAGE_REPO_SECRET"
	EnvVarSourceRepoSecretJSON = "TEST_IMAGE_REPO_DOCKERCONFIGJSON"
	EnvVarSourceURLGithub      = "TEST_PRIVATE_GITHUB"
	EnvVarSourceURLGitlab      = "TEST_PRIVATE_GITLAB"
	EnvVarSourceURLSecret      = "TEST_SOURCE_SECRET"
)

// cleanupOptions return a CleanupOptions instance.
func cleanupOptions(ctx *framework.Context, timeout time.Duration, retry time.Duration) *framework.CleanupOptions {
	return &framework.CleanupOptions{
		TestContext:   ctx,
		Timeout:       timeout,
		RetryInterval: retry,
	}
}

// clientGet is a wrapper around f.Client.Get that performs retries in case of retryable errors
func clientGet(key types.NamespacedName, obj runtime.Object) error {
	f := framework.Global

	return wait.PollImmediate(4*time.Second, 60*time.Second, func() (bool, error) {
		if err := f.Client.Get(goctx.TODO(), key, obj); err != nil {
			// check if we have an error that we want to retry
			if isRetryableError(err) {
				Logf("Error during client get is retried: '%s'", err.Error())
				return false, nil
			}

			Logf("Error during client get is not retried: '%s'", err.Error())

			// return all other errors directly
			return true, err
		}

		// successful call
		return true, nil
	})
}

func isRetryableError(err error) bool {
	if apierrors.IsServerTimeout(err) ||
		apierrors.IsTimeout(err) ||
		apierrors.IsTooManyRequests(err) ||
		err.Error() == "etcdserver: request timed out" ||
		err.Error() == "rpc error: code = Unavailable desc = transport is closing" ||
		strings.Contains(err.Error(), "net/http: request canceled while waiting for connection") {
		return true
	}
	return false
}

// createPipelineServiceAccount reads the TEST_E2E_SERVICEACCOUNT_NAME environment variable. If the value is "generated", then nothing is done.
// Otherwise it will create the service account. No error occurs if the service account already exists.
func createPipelineServiceAccount(ctx *framework.Context, f *framework.Framework, namespace string, timeout time.Duration, retry time.Duration) {
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
		err := f.Client.Create(goctx.TODO(), serviceAccount, cleanupOptions(ctx, timeout, retry))
		if err != nil && !apierrors.IsAlreadyExists(err) {
			Expect(err).ToNot(HaveOccurred(), "Error creating service account")
		}
	}
}

// createContainerRegistrySecret use environment variables to check for container registry
// credentials secret, when not found a new secret is created.
func createContainerRegistrySecret(ctx *framework.Context, f *framework.Framework, namespace string, timeout time.Duration, retry time.Duration) {
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
	err := f.Client.Create(goctx.TODO(), secret, cleanupOptions(ctx, timeout, retry))
	Expect(err).ToNot(HaveOccurred(), "on creating container registry secret")
}

// createNamespacedBuildStrategy create a namespaced BuildStrategy.
func createNamespacedBuildStrategy(
	ctx *framework.Context,
	f *framework.Framework,
	testBuildStrategy *operator.BuildStrategy,
	timeout time.Duration,
	retry time.Duration,
) {
	err := f.Client.Create(goctx.TODO(), testBuildStrategy, cleanupOptions(ctx, timeout, retry))
	if err != nil {
		Expect(err).NotTo(HaveOccurred())
	}
}

// createClusterBuildStrategy create ClusterBuildStrategy resource.
func createClusterBuildStrategy(
	ctx *framework.Context,
	f *framework.Framework,
	testBuildStrategy *operator.ClusterBuildStrategy,
	timeout time.Duration,
	retry time.Duration,
) {
	err := f.Client.Create(goctx.TODO(), testBuildStrategy, cleanupOptions(ctx, timeout, retry))
	if err != nil && !apierrors.IsAlreadyExists(err) {
		Expect(err).NotTo(HaveOccurred())
	}
}

// validateBuildRunToSucceed creates the build run and watches its flow until it succeeds.
func validateBuildRunToSucceed(
	ctx *framework.Context,
	namespace string,
	testBuildRun *operator.BuildRun,
	timeout time.Duration,
	retry time.Duration,
) {
	f := framework.Global

	trueCondition := corev1.ConditionTrue
	falseCondition := corev1.ConditionFalse

	// Ensure the BuildRun has been created
	err := f.Client.Create(goctx.TODO(), testBuildRun, cleanupOptions(ctx, timeout, retry))
	Expect(err).ToNot(HaveOccurred(), "Failed to create build run.")

	buildRunNsName := types.NamespacedName{Name: testBuildRun.Name, Namespace: namespace}

	// Ensure a BuildRun eventually moves to a succeeded TRUE status
	nextStatusLog := time.Now().Add(60 * time.Second)
	Eventually(func() corev1.ConditionStatus {
		err = clientGet(buildRunNsName, testBuildRun)
		Expect(err).ToNot(HaveOccurred(), "Error retrieving a buildRun")

		if testBuildRun.Status.GetCondition(operator.Succeeded) == nil {
			return corev1.ConditionUnknown
		}

		Expect(testBuildRun.Status.GetCondition(operator.Succeeded).Status).ToNot(Equal(falseCondition))

		now := time.Now()
		if now.After(nextStatusLog) {
			Logf("Still waiting for build run '%s' to succeed.", testBuildRun.Name)
			nextStatusLog = time.Now().Add(60 * time.Second)
		}

		return testBuildRun.Status.GetCondition(operator.Succeeded).Status

	}, time.Duration(1100*getTimeoutMultiplier())*time.Second, 5*time.Second).Should(Equal(trueCondition), "BuildRun did not succeed")

	// Verify that the BuildSpec is still available in the status
	Expect(testBuildRun.Status.BuildSpec).ToNot(BeNil())

	Logf("Test build '%s' is completed after %v !", testBuildRun.GetName(), testBuildRun.Status.CompletionTime.Time.Sub(testBuildRun.Status.StartTime.Time))
}

// validateBuildRunToFail creates the build run and watches its flow until it fails
// and verifies the reason using a regular expression.
func validateBuildRunToFail(
	ctx *framework.Context,
	namespace string,
	testBuildRun *operator.BuildRun,
	expectedMessageRegexp string,
	timeout time.Duration,
	retry time.Duration,
) {
	f := framework.Global

	trueCondition := corev1.ConditionTrue
	falseCondition := corev1.ConditionFalse

	// Create the BuildRun
	err := f.Client.Create(goctx.TODO(), testBuildRun, cleanupOptions(ctx, timeout, retry))
	Expect(err).ToNot(HaveOccurred(), "Failed to create build run.")

	// Ensure that eventually the BuildRun moves to Failed.
	buildRunNsName := types.NamespacedName{Name: testBuildRun.Name, Namespace: namespace}
	nextStatusLog := time.Now().Add(60 * time.Second)
	Eventually(func() corev1.ConditionStatus {
		err = clientGet(buildRunNsName, testBuildRun)
		Expect(err).ToNot(HaveOccurred(), "Error retrieving build run")

		if testBuildRun.Status.GetCondition(operator.Succeeded) == nil {
			return corev1.ConditionUnknown
		}

		Expect(testBuildRun.Status.GetCondition(operator.Succeeded).Status).ToNot(Equal(trueCondition))

		now := time.Now()
		if now.After(nextStatusLog) {
			Logf("Still waiting for build run '%s' to fail.", testBuildRun.Name)
			nextStatusLog = time.Now().Add(60 * time.Second)
		}

		return testBuildRun.Status.GetCondition(operator.Succeeded).Status
	}, time.Duration(550*getTimeoutMultiplier())*time.Second, 5*time.Second).Should(Equal(falseCondition), "BuildRun did not fail")

	// Verify that the BuildSpec is available in the status
	Expect(testBuildRun.Status.BuildSpec).ToNot(BeNil())

	// Verify the build run failure
	Expect(testBuildRun.Status.GetCondition(operator.Succeeded).Message).To(MatchRegexp(expectedMessageRegexp))
}

// validateServiceAccountDeletion validates that a service account is correctly deleted after the end of
// a build run and depending on the state of the build run
func validateServiceAccountDeletion(buildRun *operator.BuildRun, namespace string) {
	buildRunCondition := buildRun.Status.GetCondition(operator.Succeeded)
	if buildRunCondition != nil {
		if buildRunCondition.Status == "" || buildRunCondition.Status == corev1.ConditionUnknown {
			Logf("Skipping validation of service account deletion because build run did not end.")
			return
		}
	}

	if buildRun.Spec.ServiceAccount == nil || !buildRun.Spec.ServiceAccount.Generate {
		Logf("Skipping validation of service account deletion because service account is not generated")
		return
	}

	saNamespacedName := types.NamespacedName{
		Name:      buildRun.Name + "-sa",
		Namespace: namespace,
	}

	serviceAccount := &corev1.ServiceAccount{}

	Logf("Verifying that service account '%s' has been deleted.", saNamespacedName.Name)

	err := clientGet(saNamespacedName, serviceAccount)
	Expect(err).To(HaveOccurred(), "Expected error to retrieve the generated service account after build run completion.")
	Expect(apierrors.IsNotFound(err)).To(BeTrue(), "Expected service account to be deleted.")
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
		buildRun.Spec.ServiceAccount = &operator.ServiceAccount{
			Generate: true,
		}
	} else {
		buildRun.Spec.ServiceAccount = &operator.ServiceAccount{
			Name: &serviceAccountName,
		}
	}

	return buildRun, err
}

// getTaskRun retrieve Tekton's Task based on BuildRun instance.
func getTaskRun(
	f *framework.Framework,
	buildRun *operator.BuildRun,
) (*v1beta1.TaskRun, error) {
	taskRunList := &v1beta1.TaskRunList{}
	lbls := map[string]string{
		operator.LabelBuild:    buildRun.Spec.BuildRef.Name,
		operator.LabelBuildRun: buildRun.Name,
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

func getTimeoutMultiplier() int64 {
	value := os.Getenv(EnvVarTimeoutMultiplier)
	if value == "" {
		return 1
	}

	intValue, err := strconv.ParseInt(value, 10, 64)
	Expect(err).ToNot(HaveOccurred())
	return intValue
}
