// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package testbeta

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/onsi/gomega"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"knative.dev/pkg/apis"
	knativev1 "knative.dev/pkg/apis/duck/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

// Catalog allows you to access helper functions
type Catalog struct{}

// SecretWithAnnotation gives you a secret with build annotation
func (c *Catalog) SecretWithAnnotation(name string, ns string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   ns,
			Annotations: map[string]string{buildapi.AnnotationBuildRefSecret: "true"},
		},
	}
}

// SecretWithoutAnnotation gives you a secret without build annotation
func (c *Catalog) SecretWithoutAnnotation(name string, ns string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
}

// SecretWithStringData creates a Secret with stringData (not base64 encoded)
func (c *Catalog) SecretWithStringData(name string, ns string, stringData map[string]string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		StringData: stringData,
	}
}

// SecretWithDockerConfigJson creates a secret of type dockerconfigjson
func (c *Catalog) SecretWithDockerConfigJson(name string, ns string, host string, username string, password string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		StringData: map[string]string{
			".dockerconfigjson": fmt.Sprintf("{\"auths\":{%q:{\"username\":%q,\"password\":%q}}}", host, username, password),
		},
	}
}

// ConfigMapWithData creates a ConfigMap with data
func (c *Catalog) ConfigMapWithData(name string, ns string, data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Data: data,
	}
}

// BuildWithClusterBuildStrategyAndFalseSourceAnnotation gives you an specific Build CRD
func (c *Catalog) BuildWithClusterBuildStrategyAndFalseSourceAnnotation(name string, ns string, strategyName string) *buildapi.Build {
	buildStrategy := buildapi.ClusterBuildStrategyKind
	return &buildapi.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   ns,
			Annotations: map[string]string{buildapi.AnnotationBuildVerifyRepository: "false"},
		},
		Spec: buildapi.BuildSpec{
			Source: &buildapi.Source{
				Git: &buildapi.Git{
					URL: "foobar",
				},
				Type: buildapi.GitType,
			},
			Strategy: buildapi.Strategy{
				Name: strategyName,
				Kind: &buildStrategy,
			},
			Output: buildapi.Image{
				Image: "foobar",
			},
		},
	}
}

// BuildWithClusterBuildStrategy gives you an specific Build CRD
func (c *Catalog) BuildWithClusterBuildStrategy(name string, ns string, strategyName string, secretName string) *buildapi.Build {
	buildStrategy := buildapi.ClusterBuildStrategyKind
	return &buildapi.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: buildapi.BuildSpec{
			Source: &buildapi.Source{
				Git: &buildapi.Git{
					URL: "https://github.com/shipwright-io/sample-go",
				},
				Type: buildapi.GitType,
			},
			Strategy: buildapi.Strategy{
				Name: strategyName,
				Kind: &buildStrategy,
			},
			Output: buildapi.Image{
				Image:      "foobar",
				PushSecret: &secretName,
			},
		},
	}
}

// BuildWithClusterBuildStrategyAndSourceSecret gives you an specific Build CRD
func (c *Catalog) BuildWithClusterBuildStrategyAndSourceSecret(name string, ns string, strategyName string) *buildapi.Build {
	buildStrategy := buildapi.ClusterBuildStrategyKind
	secret := "foobar"
	return &buildapi.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: buildapi.BuildSpec{
			Source: &buildapi.Source{
				Git: &buildapi.Git{
					URL:         "https://github.com/shipwright-io/sample-go",
					CloneSecret: &secret,
				},
				Type: buildapi.GitType,
			},
			Strategy: buildapi.Strategy{
				Name: strategyName,
				Kind: &buildStrategy,
			},
			Output: buildapi.Image{
				Image: "foobar",
			},
		},
	}
}

// BuildWithBuildStrategy gives you an specific Build CRD
func (c *Catalog) BuildWithBuildStrategy(name string, ns string, strategyName string) *buildapi.Build {
	buildStrategy := buildapi.NamespacedBuildStrategyKind
	return &buildapi.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: buildapi.BuildSpec{
			Source: &buildapi.Source{
				Git: &buildapi.Git{
					URL: "https://github.com/shipwright-io/sample-go",
				},
				Type: buildapi.GitType,
			},
			Strategy: buildapi.Strategy{
				Name: strategyName,
				Kind: &buildStrategy,
			},
		},
	}
}

// BuildWithNilBuildStrategyKind gives you an Build CRD with nil build strategy kind
func (c *Catalog) BuildWithNilBuildStrategyKind(name string, ns string, strategyName string) *buildapi.Build {
	return &buildapi.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: buildapi.BuildSpec{
			Source: &buildapi.Source{
				Git: &buildapi.Git{
					URL: "https://github.com/shipwright-io/sample-go",
				},
				Type: buildapi.GitType,
			},
			Strategy: buildapi.Strategy{
				Name: strategyName,
			},
		},
	}
}

// BuildWithOutputSecret ....
func (c *Catalog) BuildWithOutputSecret(name string, ns string, secretName string) *buildapi.Build {
	return &buildapi.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: buildapi.BuildSpec{
			Source: &buildapi.Source{
				Git: &buildapi.Git{
					URL: "https://github.com/shipwright-io/sample-go",
				},
				Type: buildapi.GitType,
			},
			Output: buildapi.Image{
				PushSecret: &secretName,
			},
		},
	}
}

// ClusterBuildStrategy to support tests
func (c *Catalog) ClusterBuildStrategy(name string) *buildapi.ClusterBuildStrategy {
	return &buildapi.ClusterBuildStrategy{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: buildapi.BuildStrategySpec{
			Steps: []buildapi.Step{{
				Name:    "dummy",
				Image:   "alpine",
				Command: []string{"true"},
			}},
		},
	}
}

// FakeClusterBuildStrategyNotFound returns a not found error
func (c *Catalog) FakeClusterBuildStrategyNotFound(name string) error {
	return errors.NewNotFound(schema.GroupResource{}, name)
}

// StubFunc is used to simulate the status of the Build
// after a .Status().Update() call in the controller. This
// receives a parameter to return an specific status state
func (c *Catalog) StubFunc(status corev1.ConditionStatus, reason buildapi.BuildReason, message string) func(context context.Context, object client.Object, _ ...client.SubResourceUpdateOption) error {
	return func(context context.Context, object client.Object, _ ...client.SubResourceUpdateOption) error {
		switch object := object.(type) {
		case *buildapi.Build:
			gomega.Expect(*object.Status.Registered).To(gomega.Equal(status))
			gomega.Expect(*object.Status.Reason).To(gomega.Equal(reason))
			gomega.Expect(*object.Status.Message).To(gomega.Equal(message))
		}
		return nil
	}
}

// StubBuildUpdateOwnerReferences simulates and assert an updated
// BuildRun object ownerreferences
func (c *Catalog) StubBuildUpdateOwnerReferences(ownerKind string, ownerName string, isOwnerController *bool, blockOwnerDeletion *bool) func(context context.Context, object client.Object, _ ...client.UpdateOption) error {
	return func(context context.Context, object client.Object, _ ...client.UpdateOption) error {
		switch object := object.(type) {
		case *buildapi.BuildRun:
			gomega.Expect(object.OwnerReferences[0].Kind).To(gomega.Equal(ownerKind))
			gomega.Expect(object.OwnerReferences[0].Name).To(gomega.Equal(ownerName))
			gomega.Expect(object.OwnerReferences[0].Controller).To(gomega.Equal(isOwnerController))
			gomega.Expect(object.OwnerReferences[0].BlockOwnerDeletion).To(gomega.Equal(blockOwnerDeletion))
			gomega.Expect(len(object.OwnerReferences)).ToNot(gomega.Equal(0))
		}
		return nil
	}
}

// StubBuildRun is used to simulate the existence of a BuildRun
// only when there is a client GET on this object type
func (c *Catalog) StubBuildRun(
	b *buildapi.BuildRun,
) func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
	return func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
		switch object := object.(type) {
		case *buildapi.BuildRun:
			b.DeepCopyInto(object)
			return nil
		}
		return errors.NewNotFound(schema.GroupResource{}, nn.Name)
	}
}

// StubBuildRunAndTaskRun is used to simulate the existence of a BuildRun
// and a TaskRun when there is a client GET on this two objects
func (c *Catalog) StubBuildRunAndTaskRun(
	b *buildapi.BuildRun,
	tr *pipelineapi.TaskRun,
) func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
	return func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
		switch object := object.(type) {
		case *buildapi.BuildRun:
			b.DeepCopyInto(object)
			return nil
		case *pipelineapi.TaskRun:
			tr.DeepCopyInto(object)
			return nil
		}
		return errors.NewNotFound(schema.GroupResource{}, nn.Name)
	}
}

// StubBuildAndTaskRun is used to simulate the existence of a Build
// and a TaskRun when there is a client GET on this two objects
func (c *Catalog) StubBuildAndTaskRun(
	b *buildapi.Build,
	tr *pipelineapi.TaskRun,
) func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
	return func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
		switch object := object.(type) {
		case *buildapi.Build:
			b.DeepCopyInto(object)
			return nil
		case *pipelineapi.TaskRun:
			tr.DeepCopyInto(object)
			return nil
		}
		return errors.NewNotFound(schema.GroupResource{}, nn.Name)
	}
}

// StubBuildStatusReason asserts Status fields on a Build resource
func (c *Catalog) StubBuildStatusReason(reason buildapi.BuildReason, message string) func(context context.Context, object client.Object, _ ...client.SubResourceUpdateOption) error {
	return func(context context.Context, object client.Object, _ ...client.SubResourceUpdateOption) error {
		switch object := object.(type) {
		case *buildapi.Build:
			if object.Status.Message != nil && *object.Status.Message != "" {
				gomega.Expect(*object.Status.Message).To(gomega.Equal(message))
			}
			if object.Status.Reason != nil && *object.Status.Reason != "" {
				gomega.Expect(*object.Status.Reason).To(gomega.Equal(reason))
			}
		}
		return nil
	}
}

// StubBuildRunStatus asserts Status fields on a BuildRun resource
func (c *Catalog) StubBuildRunStatus(reason string, name *string, condition buildapi.Condition, status corev1.ConditionStatus, buildSpec buildapi.BuildSpec, tolerateEmptyStatus bool) func(context context.Context, object client.Object, _ ...client.SubResourceUpdateOption) error {
	return func(context context.Context, object client.Object, _ ...client.SubResourceUpdateOption) error {
		switch object := object.(type) {
		case *buildapi.BuildRun:
			if !tolerateEmptyStatus {
				succeededCondition := object.Status.GetCondition(buildapi.Succeeded)
				if succeededCondition != nil {
					gomega.Expect(succeededCondition.Status).To(gomega.Equal(condition.Status))
					gomega.Expect(succeededCondition.Reason).To(gomega.Equal(condition.Reason))
				}
				gomega.Expect(object.Status.TaskRunName).To(gomega.Equal(name)) // nolint:staticcheck
			}
			if object.Status.BuildSpec != nil {
				gomega.Expect(*object.Status.BuildSpec).To(gomega.Equal(buildSpec))
			}
		}
		return nil
	}
}

// StubBuildRunLabel asserts Label fields on a BuildRun resource
func (c *Catalog) StubBuildRunLabel(buildSample *buildapi.Build) func(context context.Context, object client.Object, _ ...client.UpdateOption) error {
	return func(context context.Context, object client.Object, _ ...client.UpdateOption) error {
		switch object := object.(type) {
		case *buildapi.BuildRun:
			gomega.Expect(object.Labels[buildapi.LabelBuild]).To(gomega.Equal(buildSample.Name))
			gomega.Expect(object.Labels[buildapi.LabelBuildGeneration]).To(gomega.Equal(strconv.FormatInt(buildSample.Generation, 10)))
		}
		return nil
	}
}

// StubBuildRunGetWithoutSA simulates the output of client GET calls
// for the BuildRun unit tests
func (c *Catalog) StubBuildRunGetWithoutSA(
	b *buildapi.Build,
	br *buildapi.BuildRun,
) func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
	return func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
		switch object := object.(type) {
		case *buildapi.Build:
			b.DeepCopyInto(object)
			return nil
		case *buildapi.BuildRun:
			br.DeepCopyInto(object)
			return nil
		}
		return errors.NewNotFound(schema.GroupResource{}, nn.Name)
	}
}

// StubBuildRunGetWithTaskRunAndSA returns fake object for different
// client calls
func (c *Catalog) StubBuildRunGetWithTaskRunAndSA(
	b *buildapi.Build,
	br *buildapi.BuildRun,
	tr *pipelineapi.TaskRun,
	sa *corev1.ServiceAccount,
) func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
	return func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
		switch object := object.(type) {
		case *buildapi.Build:
			b.DeepCopyInto(object)
			return nil
		case *buildapi.BuildRun:
			br.DeepCopyInto(object)
			return nil
		case *pipelineapi.TaskRun:
			tr.DeepCopyInto(object)
			return nil
		case *corev1.ServiceAccount:
			sa.DeepCopyInto(object)
			return nil
		}
		return errors.NewNotFound(schema.GroupResource{}, nn.Name)
	}
}

// StubBuildRunGetWithSA returns fake object for different
// client calls
func (c *Catalog) StubBuildRunGetWithSA(
	b *buildapi.Build,
	br *buildapi.BuildRun,
	sa *corev1.ServiceAccount,
) func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
	return func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
		switch object := object.(type) {
		case *buildapi.Build:
			b.DeepCopyInto(object)
			return nil
		case *buildapi.BuildRun:
			br.DeepCopyInto(object)
			return nil
		case *corev1.ServiceAccount:
			sa.DeepCopyInto(object)
			return nil
		}
		return errors.NewNotFound(schema.GroupResource{}, nn.Name)
	}
}

// StubBuildRunGetWithSAandStrategies simulates the output of client GET
// calls for the BuildRun unit tests
func (c *Catalog) StubBuildRunGetWithSAandStrategies(
	b *buildapi.Build,
	br *buildapi.BuildRun,
	sa *corev1.ServiceAccount,
	cb *buildapi.ClusterBuildStrategy,
	bs *buildapi.BuildStrategy,
) func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
	return func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
		switch object := object.(type) {
		case *buildapi.Build:
			if b != nil {
				b.DeepCopyInto(object)
				return nil
			}
		case *buildapi.BuildRun:
			if br != nil {
				br.DeepCopyInto(object)
				return nil
			}
		case *corev1.ServiceAccount:
			if sa != nil {
				sa.DeepCopyInto(object)
				return nil
			}
		case *buildapi.ClusterBuildStrategy:
			if cb != nil {
				cb.DeepCopyInto(object)
				return nil
			}
		case *buildapi.BuildStrategy:
			if bs != nil {
				bs.DeepCopyInto(object)
				return nil
			}
		}
		return errors.NewNotFound(schema.GroupResource{}, nn.Name)
	}
}

func (c *Catalog) StubBuildCRDs(
	b *buildapi.Build,
	br *buildapi.BuildRun,
	sa *corev1.ServiceAccount,
	cb *buildapi.ClusterBuildStrategy,
	bs *buildapi.BuildStrategy,
) func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
	return func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
		switch object := object.(type) {
		case *buildapi.Build:
			b.DeepCopyInto(object)
			return nil
		case *buildapi.BuildRun:
			br.DeepCopyInto(object)
			return nil
		case *corev1.ServiceAccount:
			sa.DeepCopyInto(object)
			return nil
		case *buildapi.ClusterBuildStrategy:
			cb.DeepCopyInto(object)
			return nil
		case *buildapi.BuildStrategy:
			bs.DeepCopyInto(object)
			return nil
		}
		return errors.NewNotFound(schema.GroupResource{}, nn.Name)
	}
}

// StubBuildCRDsPodAndTaskRun stubs different objects in case a client
// GET call is executed against them
func (c *Catalog) StubBuildCRDsPodAndTaskRun(
	b *buildapi.Build,
	br *buildapi.BuildRun,
	sa *corev1.ServiceAccount,
	cb *buildapi.ClusterBuildStrategy,
	bs *buildapi.BuildStrategy,
	tr *pipelineapi.TaskRun,
	pod *corev1.Pod,
) func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
	return func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
		switch object := object.(type) {
		case *buildapi.Build:
			b.DeepCopyInto(object)
			return nil
		case *buildapi.BuildRun:
			br.DeepCopyInto(object)
			return nil
		case *corev1.ServiceAccount:
			sa.DeepCopyInto(object)
			return nil
		case *buildapi.ClusterBuildStrategy:
			cb.DeepCopyInto(object)
			return nil
		case *buildapi.BuildStrategy:
			bs.DeepCopyInto(object)
			return nil
		case *pipelineapi.TaskRun:
			tr.DeepCopyInto(object)
			return nil
		case *corev1.Pod:
			pod.DeepCopyInto(object)
			return nil
		}
		return errors.NewNotFound(schema.GroupResource{}, nn.Name)
	}
}

// TaskRunWithStatus returns a minimal tekton TaskRun with an Status
func (c *Catalog) TaskRunWithStatus(trName string, ns string) *pipelineapi.TaskRun {
	return &pipelineapi.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      trName,
			Namespace: ns,
		},
		Spec: pipelineapi.TaskRunSpec{
			Timeout: &metav1.Duration{
				Duration: time.Minute * 2,
			},
		},
		Status: pipelineapi.TaskRunStatus{
			Status: knativev1.Status{
				Conditions: knativev1.Conditions{
					{
						Type:   apis.ConditionSucceeded,
						Reason: "Unknown",
						Status: corev1.ConditionUnknown,
					},
				},
			},
			TaskRunStatusFields: pipelineapi.TaskRunStatusFields{
				PodName: "foobar-pod",
				StartTime: &metav1.Time{
					Time: time.Now(),
				},
				CompletionTime: &metav1.Time{
					Time: time.Now(),
				},
			},
		},
	}
}

// DefaultTaskRunWithStatus returns a minimal tekton TaskRun with an Status
func (c *Catalog) DefaultTaskRunWithStatus(trName string, buildRunName string, ns string, status corev1.ConditionStatus, reason string) *pipelineapi.TaskRun {
	return &pipelineapi.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      trName,
			Namespace: ns,
			Labels:    map[string]string{"buildrun.shipwright.io/name": buildRunName},
		},
		Spec: pipelineapi.TaskRunSpec{},
		Status: pipelineapi.TaskRunStatus{
			Status: knativev1.Status{
				Conditions: knativev1.Conditions{
					{
						Type:   apis.ConditionSucceeded,
						Reason: reason,
						Status: status,
					},
				},
			},
			TaskRunStatusFields: pipelineapi.TaskRunStatusFields{
				StartTime: &metav1.Time{
					Time: time.Now(),
				},
			},
		},
	}
}

// TaskRunWithCompletionAndStartTime provides a TaskRun object with a
// Completion and StartTime
func (c *Catalog) TaskRunWithCompletionAndStartTime(trName string, buildRunName string, ns string) *pipelineapi.TaskRun {
	return &pipelineapi.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      trName,
			Namespace: ns,
			Labels:    map[string]string{"buildrun.shipwright.io/name": buildRunName},
		},
		Spec: pipelineapi.TaskRunSpec{},
		Status: pipelineapi.TaskRunStatus{
			TaskRunStatusFields: pipelineapi.TaskRunStatusFields{
				CompletionTime: &metav1.Time{
					Time: time.Now(),
				},
				StartTime: &metav1.Time{
					Time: time.Now(),
				},
				PodName: "foobar",
			},
			Status: knativev1.Status{
				Conditions: knativev1.Conditions{
					{
						Type:    apis.ConditionSucceeded,
						Reason:  "something bad happened",
						Status:  corev1.ConditionFalse,
						Message: "some message",
					},
				},
			},
		},
	}
}

// DefaultTaskRunWithFalseStatus returns a minimal tektont TaskRun with a FALSE status
func (c *Catalog) DefaultTaskRunWithFalseStatus(trName string, buildRunName string, ns string) *pipelineapi.TaskRun {
	return &pipelineapi.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      trName,
			Namespace: ns,
			Labels:    map[string]string{"buildrun.shipwright.io/name": buildRunName},
		},
		Spec: pipelineapi.TaskRunSpec{},
		Status: pipelineapi.TaskRunStatus{
			Status: knativev1.Status{
				Conditions: knativev1.Conditions{
					{
						Type:    apis.ConditionSucceeded,
						Reason:  "something bad happened",
						Status:  corev1.ConditionFalse,
						Message: "some message",
					},
				},
			},
			TaskRunStatusFields: pipelineapi.TaskRunStatusFields{
				StartTime: &metav1.Time{
					Time: time.Now(),
				},
			},
		},
	}
}

// DefaultBuild returns a minimal Build object
func (c *Catalog) DefaultBuild(buildName string, strategyName string, strategyKind buildapi.BuildStrategyKind) *buildapi.Build {
	return &buildapi.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildName,
		},
		Spec: buildapi.BuildSpec{
			Strategy: buildapi.Strategy{
				Name: strategyName,
				Kind: &strategyKind,
			},
		},
		Status: buildapi.BuildStatus{
			Registered: ptr.To(corev1.ConditionTrue),
		},
	}
}

// BuildWithoutStrategyKind returns a minimal Build object without an strategy kind definition
func (c *Catalog) BuildWithoutStrategyKind(buildName string, strategyName string) *buildapi.Build {
	return &buildapi.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildName,
		},
		Spec: buildapi.BuildSpec{
			Strategy: buildapi.Strategy{
				Name: strategyName,
			},
		},
		Status: buildapi.BuildStatus{
			Registered: ptr.To(corev1.ConditionTrue),
		},
	}
}

// BuildWithBuildRunDeletions returns a minimal Build object with the
// buildapi.shipwright.io/build-run-deletion annotation set to true
func (c *Catalog) BuildWithBuildRunDeletions(buildName string, strategyName string, strategyKind buildapi.BuildStrategyKind) *buildapi.Build {
	return &buildapi.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildName,
		},
		Spec: buildapi.BuildSpec{
			Strategy: buildapi.Strategy{
				Name: strategyName,
				Kind: &strategyKind,
			},
			Retention: &buildapi.BuildRetention{
				AtBuildDeletion: ptr.To(true),
			},
		},
		Status: buildapi.BuildStatus{
			Registered: ptr.To(corev1.ConditionTrue),
		},
	}
}

// BuildWithBuildRunDeletionsAndFakeNS returns a minimal Build object with the
// buildapi.shipwright.io/build-run-deletion annotation set to true in a fake namespace
func (c *Catalog) BuildWithBuildRunDeletionsAndFakeNS(buildName string, strategyName string, strategyKind buildapi.BuildStrategyKind) *buildapi.Build {
	return &buildapi.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildName,
			Namespace: "fakens",
		},
		Spec: buildapi.BuildSpec{
			Strategy: buildapi.Strategy{
				Name: strategyName,
				Kind: &strategyKind,
			},
			Retention: &buildapi.BuildRetention{
				AtBuildDeletion: ptr.To(true),
			},
		},
		Status: buildapi.BuildStatus{
			Registered: ptr.To(corev1.ConditionTrue),
		},
	}
}

// DefaultBuildWithFalseRegistered returns a minimal Build object with a FALSE Registered
func (c *Catalog) DefaultBuildWithFalseRegistered(buildName string, strategyName string, strategyKind buildapi.BuildStrategyKind) *buildapi.Build {
	return &buildapi.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildName,
		},
		Spec: buildapi.BuildSpec{
			Strategy: buildapi.Strategy{
				Name: strategyName,
				Kind: &strategyKind,
			},
		},
		Status: buildapi.BuildStatus{
			Registered: ptr.To(corev1.ConditionFalse),
			Reason:     ptr.To[buildapi.BuildReason]("something bad happened"),
		},
	}
}

// DefaultBuildRun returns a minimal BuildRun object
func (c *Catalog) DefaultBuildRun(buildRunName string, buildName string) *buildapi.BuildRun {
	var defaultBuild = c.DefaultBuild(buildName, "foobar-strategy", buildapi.ClusterBuildStrategyKind)
	return &buildapi.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildRunName,
		},
		Spec: buildapi.BuildRunSpec{
			Build: buildapi.ReferencedBuild{
				Name: &buildName,
			},
		},
		Status: buildapi.BuildRunStatus{
			BuildSpec: &defaultBuild.Spec,
		},
	}
}

// PodWithInitContainerStatus returns a pod with a single
// entry under the Status field for InitContainer Status
func (c *Catalog) PodWithInitContainerStatus(podName string, initContainerName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
		},
		Status: corev1.PodStatus{
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					Name: initContainerName,
				},
			},
		},
	}
}

// BuildRunWithBuildSnapshot returns BuildRun Object with a populated
// BuildSpec in the Status field
func (c *Catalog) BuildRunWithBuildSnapshot(buildRunName string, buildName string) *buildapi.BuildRun {
	return &buildapi.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildRunName,
			CreationTimestamp: metav1.Time{
				Time: time.Now(),
			},
		},
		Status: buildapi.BuildRunStatus{
			BuildSpec: &buildapi.BuildSpec{
				Strategy: buildapi.Strategy{
					Name: "foobar",
				},
			},
		},
		Spec: buildapi.BuildRunSpec{
			Build: buildapi.ReferencedBuild{
				Name: &buildName,
			},
		},
	}
}

// BuildRunWithExistingOwnerReferences returns a BuildRun object that is
// already owned by some fake object
func (c *Catalog) BuildRunWithExistingOwnerReferences(buildRunName string, buildName string, ownerName string) *buildapi.BuildRun {

	managingController := true

	fakeOwnerRef := metav1.OwnerReference{
		APIVersion: ownerName,
		Kind:       ownerName,
		Controller: &managingController,
	}

	return &buildapi.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:            buildRunName,
			OwnerReferences: []metav1.OwnerReference{fakeOwnerRef},
		},
		Spec: buildapi.BuildRunSpec{
			Build: buildapi.ReferencedBuild{
				Name: &buildName,
			},
		},
	}
}

// BuildRunWithFakeNamespace returns a BuildRun object with
// a namespace that does not exist
func (c *Catalog) BuildRunWithFakeNamespace(buildRunName string, buildName string) *buildapi.BuildRun {
	return &buildapi.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildRunName,
			Namespace: "foobarns",
		},
		Spec: buildapi.BuildRunSpec{
			Build: buildapi.ReferencedBuild{
				Name: &buildName,
			},
		},
	}
}

// DefaultTaskRun returns a minimal TaskRun object
func (c *Catalog) DefaultTaskRun(taskRunName string, ns string) *pipelineapi.TaskRun {
	return &pipelineapi.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      taskRunName,
			Namespace: ns,
		},
	}
}

// DefaultServiceAccount returns a minimal SA object
func (c *Catalog) DefaultServiceAccount(name string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

// ServiceAccountWithControllerRef ... TODO
func (c *Catalog) ServiceAccountWithControllerRef(name string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: "ss",
					Kind: "BuildRun",
				},
			},
		},
	}
}

// DefaultClusterBuildStrategy returns a minimal ClusterBuildStrategy
// object with a inmutable name
func (c *Catalog) DefaultClusterBuildStrategy() *buildapi.ClusterBuildStrategy {
	return &buildapi.ClusterBuildStrategy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foobar",
		},
	}
}

// DefaultNamespacedBuildStrategy returns a minimal BuildStrategy
// object with a inmutable name
func (c *Catalog) DefaultNamespacedBuildStrategy() *buildapi.BuildStrategy {
	return &buildapi.BuildStrategy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foobar",
		},
	}
}

// BuildRunWithNodeSelectorOverride returns a customized BuildRun object
// that defines a buildspec and overrides the nodeSelector
func (c *Catalog) BuildRunWithNodeSelectorOverride(buildRunName string, buildName string, nodeSelector map[string]string) *buildapi.BuildRun {
	return &buildapi.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildRunName,
		},
		Spec: buildapi.BuildRunSpec{
			Build: buildapi.ReferencedBuild{
				Name: &buildName,
				Spec: &buildapi.BuildSpec{Strategy: buildapi.Strategy{Name: "foobar"}},
			},
			NodeSelector: nodeSelector,
		},
		Status: buildapi.BuildRunStatus{},
	}
}

// BuildRunWithTolerationsOverride returns a customized BuildRun object
// that defines a buildspec and overrides the tolerations
func (c *Catalog) BuildRunWithTolerationsOverride(buildRunName string, buildName string, tolerations []corev1.Toleration) *buildapi.BuildRun {
	return &buildapi.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildRunName,
		},
		Spec: buildapi.BuildRunSpec{
			Build: buildapi.ReferencedBuild{
				Name: &buildName,
				Spec: &buildapi.BuildSpec{Strategy: buildapi.Strategy{Name: "foobar"}},
			},
			Tolerations: tolerations,
		},
		Status: buildapi.BuildRunStatus{},
	}
}

// BuildRunWithSchedulerNameOverride returns a customized BuildRun object
// that defines a buildspec and overrides the schedulerName
func (c *Catalog) BuildRunWithSchedulerNameOverride(buildRunName string, buildName string, schedulerName string) *buildapi.BuildRun {
	return &buildapi.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildRunName,
		},
		Spec: buildapi.BuildRunSpec{
			Build: buildapi.ReferencedBuild{
				Name: &buildName,
				Spec: &buildapi.BuildSpec{Strategy: buildapi.Strategy{Name: "foobar"}},
			},
			SchedulerName: &schedulerName,
		},
		Status: buildapi.BuildRunStatus{},
	}
}

// BuildRunWithStepResourcesOverride returns a customized BuildRun object
// that defines a buildspec and overrides the stepResources
func (c *Catalog) BuildRunWithStepResourcesOverride(buildRunName string, buildName string, stepResources []buildapi.StepResourceOverride) *buildapi.BuildRun {
	return &buildapi.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildRunName,
		},
		Spec: buildapi.BuildRunSpec{
			Build: buildapi.ReferencedBuild{
				Name: &buildName,
				Spec: &buildapi.BuildSpec{Strategy: buildapi.Strategy{Name: "foobar"}},
			},
			StepResources: stepResources,
		},
		Status: buildapi.BuildRunStatus{},
	}
}

// BuildRunWithSucceededCondition returns a BuildRun with a single condition
// of the type Succeeded
func (c *Catalog) BuildRunWithSucceededCondition() *buildapi.BuildRun {
	return &buildapi.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foobar",
		},
		Status: buildapi.BuildRunStatus{
			Conditions: buildapi.Conditions{
				buildapi.Condition{
					Type:    buildapi.Succeeded,
					Reason:  "foobar",
					Message: "foo is not bar",
					Status:  corev1.ConditionUnknown,
				},
			},
		},
	}
}

// BuildRunWithSA returns a customized BuildRun object that defines a
// service account
func (c *Catalog) BuildRunWithSA(buildRunName string, buildName string, saName string) *buildapi.BuildRun {
	return &buildapi.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildRunName,
		},
		Spec: buildapi.BuildRunSpec{
			Build: buildapi.ReferencedBuild{
				Name: &buildName,
			},
			ServiceAccount: &saName,
		},
		Status: buildapi.BuildRunStatus{},
	}
}

// BuildRunWithoutSA returns a buildrun without serviceAccountName and generate serviceAccount is false
func (c *Catalog) BuildRunWithoutSA(buildRunName string, buildName string) *buildapi.BuildRun {
	return &buildapi.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildRunName,
		},
		Spec: buildapi.BuildRunSpec{
			Build: buildapi.ReferencedBuild{
				Name: &buildName,
			},
			ServiceAccount: nil,
		},
	}
}

// BuildRunWithSAGenerate returns a customized BuildRun object that defines a
// service account
func (c *Catalog) BuildRunWithSAGenerate(buildRunName string, buildName string) *buildapi.BuildRun {
	return &buildapi.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildRunName,
		},
		Spec: buildapi.BuildRunSpec{
			Build: buildapi.ReferencedBuild{
				Name: &buildName,
			},
			ServiceAccount: ptr.To(".generate"),
		},
	}
}

// LoadCustomResources returns a container set of resources based on cpu and memory
func (c *Catalog) LoadCustomResources(cpu string, mem string) corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(cpu),
			corev1.ResourceMemory: resource.MustParse(mem),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(cpu),
			corev1.ResourceMemory: resource.MustParse(mem),
		},
	}
}

// LoadBuildYAML parses YAML bytes into JSON and from JSON
// into a Build struct
func (c *Catalog) LoadBuildYAML(d []byte) (*buildapi.Build, error) {
	b := &buildapi.Build{}
	err := yaml.Unmarshal(d, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// LoadBuildWithNameAndStrategy returns a populated Build with name and a referenced strategy
func (c *Catalog) LoadBuildWithNameAndStrategy(name string, strategy string, d []byte) (*buildapi.Build, error) {
	b := &buildapi.Build{}
	err := yaml.Unmarshal(d, b)
	if err != nil {
		return nil, err
	}
	b.Name = name
	b.Spec.Strategy.Name = strategy
	return b, nil
}

// LoadBuildRunFromBytes returns a populated BuildRun
func (c *Catalog) LoadBuildRunFromBytes(d []byte) (*buildapi.BuildRun, error) {
	b := &buildapi.BuildRun{}
	err := yaml.Unmarshal(d, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// LoadBRWithNameAndRef returns a populated BuildRun with a name and a referenced Build
func (c *Catalog) LoadBRWithNameAndRef(name string, buildName string, d []byte) (*buildapi.BuildRun, error) {
	b := &buildapi.BuildRun{}
	err := yaml.Unmarshal(d, b)
	if err != nil {
		return nil, err
	}
	b.Name = name
	b.Spec.Build.Name = &buildName
	return b, nil
}

func (c *Catalog) LoadStandAloneBuildRunWithNameAndStrategy(name string, strategy *buildapi.ClusterBuildStrategy, d []byte) (*buildapi.BuildRun, error) {
	b := &buildapi.BuildRun{}
	err := yaml.Unmarshal(d, b)
	if err != nil {
		return nil, err
	}
	b.Name = name
	b.Spec.Build.Spec.Strategy = buildapi.Strategy{Kind: (*buildapi.BuildStrategyKind)(&strategy.Kind), Name: strategy.Name}

	return b, nil
}

// LoadBuildStrategyFromBytes returns a populated BuildStrategy
func (c *Catalog) LoadBuildStrategyFromBytes(d []byte) (*buildapi.BuildStrategy, error) {
	b := &buildapi.BuildStrategy{}
	err := yaml.Unmarshal(d, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// LoadCBSWithName returns a populated ClusterBuildStrategy with a name
func (c *Catalog) LoadCBSWithName(name string, d []byte) (*buildapi.ClusterBuildStrategy, error) {
	b := &buildapi.ClusterBuildStrategy{}
	err := yaml.Unmarshal(d, b)
	if err != nil {
		return nil, err
	}
	b.Name = name
	return b, nil
}

// DefaultPipelineRunWithStatus returns a minimal tekton PipelineRun with a Status
func (c *Catalog) DefaultPipelineRunWithStatus(prName string, buildRunName string, ns string, status corev1.ConditionStatus, reason string) *pipelineapi.PipelineRun {
	return &pipelineapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prName,
			Namespace: ns,
			Labels:    map[string]string{"buildrun.shipwright.io/name": buildRunName},
		},
		Spec: pipelineapi.PipelineRunSpec{},
		Status: pipelineapi.PipelineRunStatus{
			Status: knativev1.Status{
				Conditions: knativev1.Conditions{
					{
						Type:   apis.ConditionSucceeded,
						Reason: reason,
						Status: status,
					},
				},
			},
			PipelineRunStatusFields: pipelineapi.PipelineRunStatusFields{
				StartTime: &metav1.Time{
					Time: time.Now(),
				},
			},
		},
	}
}

// DefaultPipelineRunWithFalseStatus returns a minimal tekton PipelineRun with a FALSE status
func (c *Catalog) DefaultPipelineRunWithFalseStatus(prName string, buildRunName string, ns string) *pipelineapi.PipelineRun {
	return &pipelineapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prName,
			Namespace: ns,
			Labels:    map[string]string{"buildrun.shipwright.io/name": buildRunName},
		},
		Spec: pipelineapi.PipelineRunSpec{},
		Status: pipelineapi.PipelineRunStatus{
			Status: knativev1.Status{
				Conditions: knativev1.Conditions{
					{
						Type:    apis.ConditionSucceeded,
						Reason:  "something bad happened",
						Status:  corev1.ConditionFalse,
						Message: "some message",
					},
				},
			},
			PipelineRunStatusFields: pipelineapi.PipelineRunStatusFields{
				StartTime: &metav1.Time{
					Time: time.Now(),
				},
			},
		},
	}
}

// StubBuildAndPipelineRun returns a stub function that handles GET calls for Build and PipelineRun
func (c *Catalog) StubBuildAndPipelineRun(b *buildapi.Build, pr *pipelineapi.PipelineRun) func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
	return func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
		switch object := object.(type) {
		case *buildapi.Build:
			b.DeepCopyInto(object)
			return nil
		case *pipelineapi.PipelineRun:
			pr.DeepCopyInto(object)
			return nil
		}
		return errors.NewNotFound(schema.GroupResource{}, nn.Name)
	}
}
