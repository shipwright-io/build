// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	. "github.com/onsi/gomega"
	knativeapis "knative.dev/pkg/apis"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
	"github.com/shipwright-io/build/test/utils"
)

func generateTestID(id string) string {
	return id + "-" + rand.String(5)
}

func removeTestIDSuffix(id string) string {
	return id[:len(id)-6]
}

func createBuild(testBuild *utils.TestBuild, identifier string, filePath string) *buildv1alpha1.Build {
	build, err := buildTestData(testBuild.Namespace, identifier, filePath)
	Expect(err).ToNot(HaveOccurred(), "Error retrieving build test data")

	amendBuild(identifier, build)

	// For Builds that use a namespaces build strategy, there is a race condition: we just created the
	// build strategy and it might be that the build controller does not yet have this in his cache
	// and therefore marks the build as not registered with reason BuildStrategyNotFound
	Eventually(func() buildv1alpha1.BuildReason {
		// cleanup the build of the previous try
		if build.Status.Registered != nil && *build.Status.Registered != "" {
			err = testBuild.DeleteBuild(build.Name)
			Expect(err).ToNot(HaveOccurred())

			// create a fresh object
			build, err = buildTestData(testBuild.Namespace, identifier, filePath)
			Expect(err).ToNot(HaveOccurred(), "Error retrieving build test data")

			amendBuild(identifier, build)
		}

		err = testBuild.CreateBuild(build)
		Expect(err).ToNot(HaveOccurred(), "Unable to create build %s", identifier)
		Logf("Build %s created", identifier)

		build, err = testBuild.GetBuildTillValidation(build.Name)
		Expect(err).ToNot(HaveOccurred())

		return *build.Status.Reason
	}, time.Duration(10*time.Second), time.Second).Should(Equal(buildv1alpha1.SucceedStatus))

	return build
}

// amendOutputImage amend container image URL based on informed image repository.
func amendOutputImage(b *buildv1alpha1.Build, imageRepo string) {
	if imageRepo == "" {
		return
	}

	// image tag is the build name without the test id suffix as this would pollute the container registry
	imageTag := removeTestIDSuffix(b.Name)

	imageURL := fmt.Sprintf("%s:%s", imageRepo, imageTag)
	b.Spec.Output.Image = imageURL
	Logf("Amended object: name='%s', image-url='%s'", b.Name, imageURL)
}

// amendOutputCredentials amend secret-ref for output image.
func amendOutputCredentials(b *buildv1alpha1.Build, secretName string) {
	if secretName == "" {
		return
	}
	b.Spec.Output.Credentials = &corev1.LocalObjectReference{Name: secretName}
	Logf("Amended object: name='%s', secret-ref='%s'", b.Name, secretName)
}

// amendSourceSecretName patch Build source.Credentials with secret name.
func amendSourceSecretName(b *buildv1alpha1.Build, secretName string) {
	if secretName == "" {
		return
	}
	b.Spec.Source.Credentials = &corev1.LocalObjectReference{Name: secretName}
}

// amendSourceURL patch Build source.URL with informed string.
func amendSourceURL(b *buildv1alpha1.Build, sourceURL string) {
	if sourceURL == "" {
		return
	}
	b.Spec.Source.URL = &sourceURL
}

// amendBuild make changes on build object.
func amendBuild(identifier string, b *buildv1alpha1.Build) {
	if strings.Contains(identifier, "github") {
		amendSourceSecretName(b, os.Getenv(EnvVarSourceURLSecret))
		amendSourceURL(b, os.Getenv(EnvVarSourceURLGithub))
	} else if strings.Contains(identifier, "gitlab") {
		amendSourceSecretName(b, os.Getenv(EnvVarSourceURLSecret))
		amendSourceURL(b, os.Getenv(EnvVarSourceURLGitlab))
	}

	amendOutputImage(b, os.Getenv(EnvVarImageRepo))
	amendOutputCredentials(b, os.Getenv(EnvVarImageRepoSecret))
}

// retrieveBuildAndBuildRun will retrieve the build and buildRun
func retrieveBuildAndBuildRun(testBuild *utils.TestBuild, namespace string, buildRunName string) (*buildv1alpha1.Build, *buildv1alpha1.BuildRun, error) {
	buildRun, err := testBuild.LookupBuildRun(types.NamespacedName{Name: buildRunName, Namespace: namespace})
	if err != nil {
		Logf("Failed to get BuildRun %q: %s", buildRunName, err)
		return nil, nil, err
	}

	var build buildv1alpha1.Build
	if err := resources.GetBuildObject(testBuild.Context, testBuild.ControllerRuntimeClient, buildRun, &build); err != nil {
		Logf("Failed to get Build from BuildRun %s: %s", buildRunName, err)
		return nil, buildRun, err
	}

	return &build, buildRun, nil
}

// printTestFailureDebugInfo will output the status of Build, BuildRun, TaskRun and Pod, also print logs of Pod
func printTestFailureDebugInfo(testBuild *utils.TestBuild, namespace string, buildRunName string) {
	Logf("Print failed BuildRun's log")

	build, buildRun, err := retrieveBuildAndBuildRun(testBuild, namespace, buildRunName)
	if err != nil {
		Logf("Failed to retrieve build and buildrun logs: %v", err)
	}

	if build != nil {
		Logf("The status of Build %s: registered=%s, reason=%s", build.Name, *build.Status.Registered, *build.Status.Reason)
		if buildJSON, err := json.Marshal(build); err == nil {
			Logf("The full Build: %s", string(buildJSON))
		}
	}

	if buildRun != nil {
		brCondition := buildRun.Status.GetCondition(buildv1alpha1.Succeeded)
		if brCondition != nil {
			Logf("The status of BuildRun %s: status=%s, reason=%s", buildRun.Name, brCondition.Status, brCondition.Reason)
		}
		if buildRunJSON, err := json.Marshal(buildRun); err == nil {
			Logf("The full BuildRun: %s", string(buildRunJSON))
		}

		podName := ""

		// Only log details of TaskRun if Tekton objects can be accessed
		if os.Getenv(EnvVarVerifyTektonObjects) == "true" {
			if taskRun, _ := testBuild.LookupTaskRunUsingBuildRun(buildRun); taskRun != nil {
				condition := taskRun.Status.GetCondition(knativeapis.ConditionSucceeded)
				if condition != nil {
					Logf("The status of TaskRun %s: reason=%s, message=%s", taskRun.Name, condition.Reason, condition.Message)
				}

				if taskRunJSON, err := json.Marshal(taskRun); err == nil {
					Logf("The full TaskRun: %s", string(taskRunJSON))
				}

				podName = taskRun.Status.PodName
			}
		}

		// retrieve or query pod depending on whether we have the pod name from the TaskRun
		var pod *corev1.Pod
		if podName != "" {
			pod, err = testBuild.LookupPod(types.NamespacedName{Name: podName, Namespace: namespace})
			if err != nil {
				Logf("Error retrieving pod %s: %v", podName, err)
				pod = nil
			}
		} else {
			podList, err := testBuild.Clientset.CoreV1().Pods(namespace).List(testBuild.Context, metav1.ListOptions{
				LabelSelector: labels.FormatLabels(map[string]string{
					buildv1alpha1.LabelBuildRun: buildRunName,
				}),
			})

			if err == nil && len(podList.Items) > 0 {
				pod = &podList.Items[0]
			}
		}

		if pod != nil {
			Logf("The status of Pod %s: phase=%s, reason=%s, message=%s", pod.Name, pod.Status.Phase, pod.Status.Reason, pod.Status.Message)
			if podJSON, err := json.Marshal(pod); err == nil {
				Logf("The full Pod: %s", string(podJSON))
			}

			// Loop through the containers to print their logs
			for _, container := range pod.Spec.Containers {
				req := testBuild.Clientset.CoreV1().Pods(namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
					TypeMeta:  metav1.TypeMeta{},
					Container: container.Name,
					Follow:    false,
				})

				podLogs, err := req.Stream(testBuild.Context)
				if err != nil {
					Logf("Failed to retrieve the logs of container %s: %v", container.Name, err)
					continue
				}

				buf := new(bytes.Buffer)
				_, err = io.Copy(buf, podLogs)
				if err != nil {
					Logf("Failed to copy logs of container %s to buffer: %v", container.Name, err)
					continue
				}

				Logf("Logs of container %s: %s", container.Name, buf.String())
			}
		}
	}
}
