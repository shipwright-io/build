package e2e

import (
	"bytes"
	goctx "context"
	"fmt"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	operator "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	v1 "k8s.io/api/core/v1"

	. "github.com/onsi/gomega"
)

// amendOutputImageURL amend container image URL based on informed image repository.
func amendOuputImageURL(b *operator.Build, imageRepo string) {
	if imageRepo == "" {
		return
	}
	imageURL := fmt.Sprintf("%s:%s", imageRepo, b.Name)
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

	amendOuputImageURL(b, os.Getenv(EnvVarImageRepo))
	amendOutputSecretRef(b, os.Getenv(EnvVarImageRepoSecret))
}

// CreateBuild loads the builds definition from the file path, unifies the output image based on
// the identifier and creates it in a namespace
func createBuild(namespace string, identifier string, filePath string) {
	Logf("Creating build %s", identifier)

	rootDir, err := getRootDir()
	Expect(err).ToNot(HaveOccurred())

	b, err := buildTestData(namespace, identifier, rootDir+"/"+filePath)
	Expect(err).ToNot(HaveOccurred())

	amendBuild(identifier, b)

	f := framework.Global
	err = f.Client.Create(goctx.TODO(), b, cleanupOptions(ctx))
	Expect(err).ToNot(HaveOccurred(), "Unable to create build %s", identifier)

	Logf("Build %s created", identifier)
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
		return nil, nil, err
	}
	return buildRun, build, nil
}

// outputBuildAndBuildRunStatusAndPodLogs will output the status of build and buildRun, print logs of taskRun pod
func outputBuildAndBuildRunStatusAndPodLogs(namespace string, buildRunName string,) {
	f := framework.Global
	buildRun, build, err := retrieveBuildAndBuildRun(namespace, buildRunName)
	Expect(err).ToNot(HaveOccurred(), "Failed to retrieve build and buildRun")
	Logf("The status of Build %s: %s, %s", build.Name, build.Status.Reason, build.Status.Registered)
	Logf("The status of BuildRun %s: %s", buildRun.Name, buildRun.Status.Succeeded)
	Logf("The reason of BuildRun %s: %s", buildRun.Name, buildRun.Status.Reason)

	lbls := map[string]string{
		operator.LabelBuildRun: buildRunName,
	}
	listOptions := client.ListOptions{
		Namespace: namespace,
		LabelSelector: labels.SelectorFromSet(lbls),
	}

	podList := &v1.PodList{}
	err = f.Client.List(goctx.TODO(), podList, &listOptions)
	Expect(err).ToNot(HaveOccurred(), "Failed to retrieve pods.")
	Expect(len(podList.Items)).To(Equal(1), "Did not retrieve one pod.")

	pod := &podList.Items[0]

	for _, container := range pod.Spec.Containers {
		req := f.KubeClient.CoreV1().Pods(namespace).GetLogs(pod.Name, &v1.PodLogOptions{
			TypeMeta:                     metav1.TypeMeta{},
			Container:                    container.Name,
			Follow:                       false,
		})
		podLogs, err := req.Stream()
		Expect(err).ToNot(HaveOccurred(), "Failed to retrieve logs of container.")
		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, podLogs)
		Expect(err).ToNot(HaveOccurred(), "Failed to copy container logs to buffer.")
		strLogs := buf.String()
		Logf("Logs of container %s: %s", container.Name, strLogs)
	}
}
