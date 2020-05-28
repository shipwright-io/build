package e2e

import (
	//"bytes"
	goctx "context"
	"fmt"
	//"io"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/apimachinery/pkg/types"
	"os"
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

//func getFailedPodLogs(namespace string, buildRunName string, f *framework.Framework) {
//	var BuildRunPodName string
//	var BuildRunPodContainersList []string
//	buildRunNsName := types.NamespacedName{Name: buildRunName, Namespace: namespace}
//	err := f.Client.Get(goctx.TODO(),)
//	PodList, _ := f.KubeClient.CoreV1().Pods(namespace).List(metav1.ListOptions{})
//	for _, pod := range PodList.Items {
//		if strings.Contains(pod.Name, buildRun.Name) {
//			BuildRunPodName = pod.Name
//			for _, container := range pod.Spec.Containers {
//				BuildRunPodContainersList = append(BuildRunPodContainersList, container.Name)
//			}
//		}
//	}
//	for _, container := range BuildRunPodContainersList {
//		req := f.KubeClient.CoreV1().Pods(namespace).GetLogs(BuildRunPodName, &v1.PodLogOptions{
//			TypeMeta:                     metav1.TypeMeta{},
//			Container:                    container,
//			Follow:                       false,
//		})
//		podLogs, err := req.Stream()
//		if err != nil {
//			Logf("error in opening stream")
//		}
//		buf := new(bytes.Buffer)
//		_, err = io.Copy(buf, podLogs)
//		if err != nil {
//			Logf("error in copy information from podLogs to buf")
//		}
//		str := buf.String()
//		Logf("container %s log is %s", container, str)
//	}
//}
