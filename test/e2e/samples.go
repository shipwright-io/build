// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	. "github.com/onsi/gomega"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"knative.dev/pkg/apis"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operator "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

// amendOutputImageURL amend container image URL based on informed image repository.
func amendOutputImageURL(b *operator.Build, imageRepo string) {
	if imageRepo == "" {
		return
	}

	// image tag is the build name without the test id suffix as this would pollute the container registry
	imageTag := removeTestIDSuffix(b.Name)

	imageURL := fmt.Sprintf("%s:%s", imageRepo, imageTag)
	b.Spec.Output.ImageURL = imageURL
	Logf("Amended object: name='%s', image-url='%s'", b.Name, imageURL)
}

// amendOutputSecretRef amend secret-ref for output image.
func amendOutputSecretRef(b *operator.Build, secretName string) {
	if secretName == "" {
		return
	}
	b.Spec.Output.SecretRef = &v1.LocalObjectReference{Name: secretName}
	Logf("Amended object: name='%s', secret-ref='%s'", b.Name, secretName)
}

// amendSourceSecretName patch Build source.SecretRef with secret name.
func amendSourceSecretName(b *operator.Build, secretName string) {
	if secretName == "" {
		return
	}
	b.Spec.Source.SecretRef = &v1.LocalObjectReference{Name: secretName}
}

// amendSourceURL patch Build source.URL with informed string.
func amendSourceURL(b *operator.Build, sourceURL string) {
	if sourceURL == "" {
		return
	}
	b.Spec.Source.URL = sourceURL
}

// amendBuild make changes on build object.
func amendBuild(identifier string, b *operator.Build) {
	amendSourceSecretName(b, os.Getenv(EnvVarSourceURLSecret))
	if strings.Contains(identifier, "github") {
		amendSourceURL(b, os.Getenv(EnvVarSourceURLGithub))
	} else if strings.Contains(identifier, "gitlab") {
		amendSourceURL(b, os.Getenv(EnvVarSourceURLGitlab))
	}

	amendOutputImageURL(b, os.Getenv(EnvVarImageRepo))
	amendOutputSecretRef(b, os.Getenv(EnvVarImageRepoSecret))
}

// CreateBuild loads the builds definition from the file path, unifies the output image based on
// the identifier, creates it in a namespace and waits for it to be registered
func createBuild(ctx *framework.Context, namespace string, identifier string, filePath string, timeout time.Duration, retry time.Duration) {
	Logf("Creating build %s", identifier)

	rootDir, err := getRootDir()
	Expect(err).ToNot(HaveOccurred(), "Unable to get root dir")

	b, err := buildTestData(namespace, identifier, rootDir+"/"+filePath)
	Expect(err).ToNot(HaveOccurred(), "Error retrieving buildrun test data")

	amendBuild(identifier, b)

	f := framework.Global
	err = f.Client.Create(context.TODO(), b, cleanupOptions(ctx, timeout, retry))
	Expect(err).ToNot(HaveOccurred(), "Unable to create build %s", identifier)

	Logf("Build %s created", identifier)

	buildName := types.NamespacedName{
		Namespace: namespace,
		Name:      b.Name,
	}

	trueCondition := corev1.ConditionTrue
	falseCondition := corev1.ConditionFalse

	Eventually(func() corev1.ConditionStatus {
		err = clientGet(buildName, b)
		Expect(err).ToNot(HaveOccurred(), "Error retrieving a build")

		Expect(b.Status.Registered).ToNot(Equal(falseCondition), "Build registered status is false")

		return b.Status.Registered
	}, time.Duration(20*getTimeoutMultiplier())*time.Second, time.Second).Should(Equal(trueCondition), "Build was not registered")
}

// retrieveBuildAndBuildRun will retrieve the build and buildRun
func retrieveBuildAndBuildRun(namespace string, buildRunName string) (*operator.BuildRun, *operator.Build, error) {
	buildRun := &operator.BuildRun{}
	build := &operator.Build{}
	err := clientGet(types.NamespacedName{Name: buildRunName, Namespace: namespace}, buildRun)
	if err != nil {
		Logf("Failed to get BuildRun %s: %s", buildRunName, err)
		return nil, nil, err
	}
	buildName := buildRun.Spec.BuildRef.Name
	err = clientGet(types.NamespacedName{Name: buildName, Namespace: namespace}, build)
	if err != nil {
		Logf("Failed to get Build %s: %s", buildName, err)
		return buildRun, nil, err
	}
	return buildRun, build, nil
}

// printTestFailureDebugInfo will output the status of Build, BuildRun, TaskRun and Pod, also print logs of Pod
func printTestFailureDebugInfo(namespace string, buildRunName string) {
	Logf("Print failed BuildRun's log")

	f := framework.Global

	buildRun, build, err := retrieveBuildAndBuildRun(namespace, buildRunName)
	if err != nil {
		Logf("Failed to retrieve build and buildrun logs: %w", err)
	}

	if build != nil {
		Logf("The status of Build %s: registered=%s, reason=%s", build.Name, build.Status.Registered, build.Status.Reason)
		if buildJSON, err := json.Marshal(build); err == nil {
			Logf("The full Build: %s", string(buildJSON))
		}
	}

	if buildRun != nil {
		brCondition := buildRun.Status.GetCondition(operator.Succeeded)
		if brCondition != nil {
			Logf("The status of BuildRun %s: status=%s, reason=%s", buildRun.Name, brCondition.Status, brCondition.Reason)
		}
		if buildRunJSON, err := json.Marshal(buildRun); err == nil {
			Logf("The full BuildRun: %s", string(buildRunJSON))
		}

		podName := ""

		// Only log details of TaskRun if Tekton objects can be accessed
		if os.Getenv(EnvVarVerifyTektonObjects) == "true" {
			if taskRun, _ := getTaskRun(f, buildRun); taskRun != nil {
				condition := taskRun.Status.GetCondition(apis.ConditionSucceeded)
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
		var pod *v1.Pod
		if podName != "" {
			pod = &v1.Pod{}
			if err = clientGet(types.NamespacedName{Name: podName, Namespace: namespace}, pod); err != nil {
				Logf("Error retrieving pod %s: %v", podName, err)
				pod = nil
			}
		} else {
			listOptions := client.ListOptions{
				Namespace: namespace,
				LabelSelector: labels.SelectorFromSet(map[string]string{
					operator.LabelBuildRun: buildRunName,
				}),
			}

			podList := &v1.PodList{}
			if err = f.Client.List(context.TODO(), podList, &listOptions); err == nil && len(podList.Items) > 0 {
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
				req := f.KubeClient.CoreV1().Pods(namespace).GetLogs(pod.Name, &v1.PodLogOptions{
					TypeMeta:  metav1.TypeMeta{},
					Container: container.Name,
					Follow:    false,
				})

				podLogs, err := req.Stream(context.TODO())
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

func generateTestID(id string) string {
	return id + "-" + utilrand.String(5)
}

func removeTestIDSuffix(id string) string {
	return id[:len(id)-6]
}
