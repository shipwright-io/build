// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package testbeta

import (
	"context"
	"fmt"
	"strconv"
	"time"

	. "github.com/onsi/gomega"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"knative.dev/pkg/apis"
	knativev1 "knative.dev/pkg/apis/duck/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	build "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

// Catalog allows you to access helper functions
type Catalog struct{}

// SecretWithAnnotation gives you a secret with build annotation
func (c *Catalog) SecretWithAnnotation(name string, ns string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   ns,
			Annotations: map[string]string{build.AnnotationBuildRefSecret: "true"},
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
func (c *Catalog) BuildWithClusterBuildStrategyAndFalseSourceAnnotation(name string, ns string, strategyName string) *build.Build {
	buildStrategy := build.ClusterBuildStrategyKind
	return &build.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   ns,
			Annotations: map[string]string{build.AnnotationBuildVerifyRepository: "false"},
		},
		Spec: build.BuildSpec{
			Source: &build.Source{
				Git: &build.Git{
					URL: "foobar",
				},
				Type: build.GitType,
			},
			Strategy: build.Strategy{
				Name: strategyName,
				Kind: &buildStrategy,
			},
			Output: build.Image{
				Image: "foobar",
			},
		},
	}
}

// BuildWithClusterBuildStrategy gives you an specific Build CRD
func (c *Catalog) BuildWithClusterBuildStrategy(name string, ns string, strategyName string, secretName string) *build.Build {
	buildStrategy := build.ClusterBuildStrategyKind
	return &build.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: build.BuildSpec{
			Source: &build.Source{
				Git: &build.Git{
					URL: "https://github.com/shipwright-io/sample-go",
				},
				Type: build.GitType,
			},
			Strategy: build.Strategy{
				Name: strategyName,
				Kind: &buildStrategy,
			},
			Output: build.Image{
				Image:      "foobar",
				PushSecret: &secretName,
			},
		},
	}
}

// BuildWithClusterBuildStrategyAndSourceSecret gives you an specific Build CRD
func (c *Catalog) BuildWithClusterBuildStrategyAndSourceSecret(name string, ns string, strategyName string) *build.Build {
	buildStrategy := build.ClusterBuildStrategyKind
	secret := "foobar"
	return &build.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: build.BuildSpec{
			Source: &build.Source{
				Git: &build.Git{
					URL:         "https://github.com/shipwright-io/sample-go",
					CloneSecret: &secret,
				},
				Type: build.GitType,
			},
			Strategy: build.Strategy{
				Name: strategyName,
				Kind: &buildStrategy,
			},
			Output: build.Image{
				Image: "foobar",
			},
		},
	}
}

// BuildWithBuildStrategy gives you an specific Build CRD
func (c *Catalog) BuildWithBuildStrategy(name string, ns string, strategyName string) *build.Build {
	buildStrategy := build.NamespacedBuildStrategyKind
	return &build.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: build.BuildSpec{
			Source: &build.Source{
				Git: &build.Git{
					URL: "https://github.com/shipwright-io/sample-go",
				},
				Type: build.GitType,
			},
			Strategy: build.Strategy{
				Name: strategyName,
				Kind: &buildStrategy,
			},
		},
	}
}

// BuildWithNilBuildStrategyKind gives you an Build CRD with nil build strategy kind
func (c *Catalog) BuildWithNilBuildStrategyKind(name string, ns string, strategyName string) *build.Build {
	return &build.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: build.BuildSpec{
			Source: &build.Source{
				Git: &build.Git{
					URL: "https://github.com/shipwright-io/sample-go",
				},
				Type: build.GitType,
			},
			Strategy: build.Strategy{
				Name: strategyName,
			},
		},
	}
}

// BuildWithOutputSecret ....
func (c *Catalog) BuildWithOutputSecret(name string, ns string, secretName string) *build.Build {
	return &build.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: build.BuildSpec{
			Source: &build.Source{
				Git: &build.Git{
					URL: "https://github.com/shipwright-io/sample-go",
				},
				Type: build.GitType,
			},
			Output: build.Image{
				PushSecret: &secretName,
			},
		},
	}
}

// ClusterBuildStrategy to support tests
func (c *Catalog) ClusterBuildStrategy(name string) *build.ClusterBuildStrategy {
	return &build.ClusterBuildStrategy{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: build.BuildStrategySpec{
			Steps: []build.Step{{
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
func (c *Catalog) StubFunc(status corev1.ConditionStatus, reason build.BuildReason, message string) func(context context.Context, object client.Object, _ ...client.SubResourceUpdateOption) error {
	return func(context context.Context, object client.Object, _ ...client.SubResourceUpdateOption) error {
		switch object := object.(type) {
		case *build.Build:
			Expect(*object.Status.Registered).To(Equal(status))
			Expect(*object.Status.Reason).To(Equal(reason))
			Expect(*object.Status.Message).To(Equal(message))
		}
		return nil
	}
}

// StubBuildUpdateOwnerReferences simulates and assert an updated
// BuildRun object ownerreferences
func (c *Catalog) StubBuildUpdateOwnerReferences(ownerKind string, ownerName string, isOwnerController *bool, blockOwnerDeletion *bool) func(context context.Context, object client.Object, _ ...client.UpdateOption) error {
	return func(context context.Context, object client.Object, _ ...client.UpdateOption) error {
		switch object := object.(type) {
		case *build.BuildRun:
			Expect(object.OwnerReferences[0].Kind).To(Equal(ownerKind))
			Expect(object.OwnerReferences[0].Name).To(Equal(ownerName))
			Expect(object.OwnerReferences[0].Controller).To(Equal(isOwnerController))
			Expect(object.OwnerReferences[0].BlockOwnerDeletion).To(Equal(blockOwnerDeletion))
			Expect(len(object.OwnerReferences)).ToNot(Equal(0))
		}
		return nil
	}
}

// StubBuildRun is used to simulate the existence of a BuildRun
// only when there is a client GET on this object type
func (c *Catalog) StubBuildRun(
	b *build.BuildRun,
) func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
	return func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
		switch object := object.(type) {
		case *build.BuildRun:
			b.DeepCopyInto(object)
			return nil
		}
		return errors.NewNotFound(schema.GroupResource{}, nn.Name)
	}
}

// StubBuildRunAndTaskRun is used to simulate the existence of a BuildRun
// and a TaskRun when there is a client GET on this two objects
func (c *Catalog) StubBuildRunAndTaskRun(
	b *build.BuildRun,
	tr *pipelineapi.TaskRun,
) func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
	return func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
		switch object := object.(type) {
		case *build.BuildRun:
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
	b *build.Build,
	tr *pipelineapi.TaskRun,
) func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
	return func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
		switch object := object.(type) {
		case *build.Build:
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
func (c *Catalog) StubBuildStatusReason(reason build.BuildReason, message string) func(context context.Context, object client.Object, _ ...client.SubResourceUpdateOption) error {
	return func(context context.Context, object client.Object, _ ...client.SubResourceUpdateOption) error {
		switch object := object.(type) {
		case *build.Build:
			if object.Status.Message != nil && *object.Status.Message != "" {
				Expect(*object.Status.Message).To(Equal(message))
			}
			if object.Status.Reason != nil && *object.Status.Reason != "" {
				Expect(*object.Status.Reason).To(Equal(reason))
			}
		}
		return nil
	}
}

// StubBuildRunStatus asserts Status fields on a BuildRun resource
func (c *Catalog) StubBuildRunStatus(reason string, name *string, condition build.Condition, status corev1.ConditionStatus, buildSpec build.BuildSpec, tolerateEmptyStatus bool) func(context context.Context, object client.Object, _ ...client.SubResourceUpdateOption) error {
	return func(context context.Context, object client.Object, _ ...client.SubResourceUpdateOption) error {
		switch object := object.(type) {
		case *build.BuildRun:
			if !tolerateEmptyStatus {
				succeededCondition := object.Status.GetCondition(build.Succeeded)
				if succeededCondition != nil {
					Expect(succeededCondition.Status).To(Equal(condition.Status))
					Expect(succeededCondition.Reason).To(Equal(condition.Reason))
				}
				Expect(object.Status.TaskRunName).To(Equal(name)) // nolint:staticcheck
			}
			if object.Status.BuildSpec != nil {
				Expect(*object.Status.BuildSpec).To(Equal(buildSpec))
			}
		}
		return nil
	}
}

// StubBuildRunLabel asserts Label fields on a BuildRun resource
func (c *Catalog) StubBuildRunLabel(buildSample *build.Build) func(context context.Context, object client.Object, _ ...client.UpdateOption) error {
	return func(context context.Context, object client.Object, _ ...client.UpdateOption) error {
		switch object := object.(type) {
		case *build.BuildRun:
			Expect(object.Labels[build.LabelBuild]).To(Equal(buildSample.Name))
			Expect(object.Labels[build.LabelBuildGeneration]).To(Equal(strconv.FormatInt(buildSample.Generation, 10)))
		}
		return nil
	}
}

// StubBuildRunGetWithoutSA simulates the output of client GET calls
// for the BuildRun unit tests
func (c *Catalog) StubBuildRunGetWithoutSA(
	b *build.Build,
	br *build.BuildRun,
) func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
	return func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
		switch object := object.(type) {
		case *build.Build:
			b.DeepCopyInto(object)
			return nil
		case *build.BuildRun:
			br.DeepCopyInto(object)
			return nil
		}
		return errors.NewNotFound(schema.GroupResource{}, nn.Name)
	}
}

// StubBuildRunGetWithTaskRunAndSA returns fake object for different
// client calls
func (c *Catalog) StubBuildRunGetWithTaskRunAndSA(
	b *build.Build,
	br *build.BuildRun,
	tr *pipelineapi.TaskRun,
	sa *corev1.ServiceAccount,
) func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
	return func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
		switch object := object.(type) {
		case *build.Build:
			b.DeepCopyInto(object)
			return nil
		case *build.BuildRun:
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
	b *build.Build,
	br *build.BuildRun,
	sa *corev1.ServiceAccount,
) func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
	return func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
		switch object := object.(type) {
		case *build.Build:
			b.DeepCopyInto(object)
			return nil
		case *build.BuildRun:
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
	b *build.Build,
	br *build.BuildRun,
	sa *corev1.ServiceAccount,
	cb *build.ClusterBuildStrategy,
	bs *build.BuildStrategy,
) func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
	return func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
		switch object := object.(type) {
		case *build.Build:
			if b != nil {
				b.DeepCopyInto(object)
				return nil
			}
		case *build.BuildRun:
			if br != nil {
				br.DeepCopyInto(object)
				return nil
			}
		case *corev1.ServiceAccount:
			if sa != nil {
				sa.DeepCopyInto(object)
				return nil
			}
		case *build.ClusterBuildStrategy:
			if cb != nil {
				cb.DeepCopyInto(object)
				return nil
			}
		case *build.BuildStrategy:
			if bs != nil {
				bs.DeepCopyInto(object)
				return nil
			}
		}
		return errors.NewNotFound(schema.GroupResource{}, nn.Name)
	}
}

func (c *Catalog) StubBuildCRDs(
	b *build.Build,
	br *build.BuildRun,
	sa *corev1.ServiceAccount,
	cb *build.ClusterBuildStrategy,
	bs *build.BuildStrategy,
) func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
	return func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
		switch object := object.(type) {
		case *build.Build:
			b.DeepCopyInto(object)
			return nil
		case *build.BuildRun:
			br.DeepCopyInto(object)
			return nil
		case *corev1.ServiceAccount:
			sa.DeepCopyInto(object)
			return nil
		case *build.ClusterBuildStrategy:
			cb.DeepCopyInto(object)
			return nil
		case *build.BuildStrategy:
			bs.DeepCopyInto(object)
			return nil
		}
		return errors.NewNotFound(schema.GroupResource{}, nn.Name)
	}
}

// StubBuildCRDsPodAndTaskRun stubs different objects in case a client
// GET call is executed against them
func (c *Catalog) StubBuildCRDsPodAndTaskRun(
	b *build.Build,
	br *build.BuildRun,
	sa *corev1.ServiceAccount,
	cb *build.ClusterBuildStrategy,
	bs *build.BuildStrategy,
	tr *pipelineapi.TaskRun,
	pod *corev1.Pod,
) func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
	return func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
		switch object := object.(type) {
		case *build.Build:
			b.DeepCopyInto(object)
			return nil
		case *build.BuildRun:
			br.DeepCopyInto(object)
			return nil
		case *corev1.ServiceAccount:
			sa.DeepCopyInto(object)
			return nil
		case *build.ClusterBuildStrategy:
			cb.DeepCopyInto(object)
			return nil
		case *build.BuildStrategy:
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
func (c *Catalog) DefaultBuild(buildName string, strategyName string, strategyKind build.BuildStrategyKind) *build.Build {
	return &build.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildName,
		},
		Spec: build.BuildSpec{
			Strategy: build.Strategy{
				Name: strategyName,
				Kind: &strategyKind,
			},
		},
		Status: build.BuildStatus{
			Registered: ptr.To(corev1.ConditionTrue),
		},
	}
}

// BuildWithoutStrategyKind returns a minimal Build object without an strategy kind definition
func (c *Catalog) BuildWithoutStrategyKind(buildName string, strategyName string) *build.Build {
	return &build.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildName,
		},
		Spec: build.BuildSpec{
			Strategy: build.Strategy{
				Name: strategyName,
			},
		},
		Status: build.BuildStatus{
			Registered: ptr.To(corev1.ConditionTrue),
		},
	}
}

// BuildWithBuildRunDeletions returns a minimal Build object with the
// build.shipwright.io/build-run-deletion annotation set to true
func (c *Catalog) BuildWithBuildRunDeletions(buildName string, strategyName string, strategyKind build.BuildStrategyKind) *build.Build {
	return &build.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildName,
		},
		Spec: build.BuildSpec{
			Strategy: build.Strategy{
				Name: strategyName,
				Kind: &strategyKind,
			},
			Retention: &build.BuildRetention{
				AtBuildDeletion: ptr.To(true),
			},
		},
		Status: build.BuildStatus{
			Registered: ptr.To(corev1.ConditionTrue),
		},
	}
}

// BuildWithBuildRunDeletionsAndFakeNS returns a minimal Build object with the
// build.shipwright.io/build-run-deletion annotation set to true in a fake namespace
func (c *Catalog) BuildWithBuildRunDeletionsAndFakeNS(buildName string, strategyName string, strategyKind build.BuildStrategyKind) *build.Build {
	return &build.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildName,
			Namespace: "fakens",
		},
		Spec: build.BuildSpec{
			Strategy: build.Strategy{
				Name: strategyName,
				Kind: &strategyKind,
			},
			Retention: &build.BuildRetention{
				AtBuildDeletion: ptr.To(true),
			},
		},
		Status: build.BuildStatus{
			Registered: ptr.To(corev1.ConditionTrue),
		},
	}
}

// DefaultBuildWithFalseRegistered returns a minimal Build object with a FALSE Registered
func (c *Catalog) DefaultBuildWithFalseRegistered(buildName string, strategyName string, strategyKind build.BuildStrategyKind) *build.Build {
	return &build.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildName,
		},
		Spec: build.BuildSpec{
			Strategy: build.Strategy{
				Name: strategyName,
				Kind: &strategyKind,
			},
		},
		Status: build.BuildStatus{
			Registered: ptr.To(corev1.ConditionFalse),
			Reason:     ptr.To[build.BuildReason]("something bad happened"),
		},
	}
}

// DefaultBuildRun returns a minimal BuildRun object
func (c *Catalog) DefaultBuildRun(buildRunName string, buildName string) *build.BuildRun {
	var defaultBuild = c.DefaultBuild(buildName, "foobar-strategy", build.ClusterBuildStrategyKind)
	return &build.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildRunName,
		},
		Spec: build.BuildRunSpec{
			Build: build.ReferencedBuild{
				Name: &buildName,
			},
		},
		Status: build.BuildRunStatus{
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
func (c *Catalog) BuildRunWithBuildSnapshot(buildRunName string, buildName string) *build.BuildRun {
	return &build.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildRunName,
			CreationTimestamp: metav1.Time{
				Time: time.Now(),
			},
		},
		Status: build.BuildRunStatus{
			BuildSpec: &build.BuildSpec{
				Strategy: build.Strategy{
					Name: "foobar",
				},
			},
		},
		Spec: build.BuildRunSpec{
			Build: build.ReferencedBuild{
				Name: &buildName,
			},
		},
	}
}

// BuildRunWithExistingOwnerReferences returns a BuildRun object that is
// already owned by some fake object
func (c *Catalog) BuildRunWithExistingOwnerReferences(buildRunName string, buildName string, ownerName string) *build.BuildRun {

	managingController := true

	fakeOwnerRef := metav1.OwnerReference{
		APIVersion: ownerName,
		Kind:       ownerName,
		Controller: &managingController,
	}

	return &build.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:            buildRunName,
			OwnerReferences: []metav1.OwnerReference{fakeOwnerRef},
		},
		Spec: build.BuildRunSpec{
			Build: build.ReferencedBuild{
				Name: &buildName,
			},
		},
	}
}

// BuildRunWithFakeNamespace returns a BuildRun object with
// a namespace that does not exist
func (c *Catalog) BuildRunWithFakeNamespace(buildRunName string, buildName string) *build.BuildRun {
	return &build.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildRunName,
			Namespace: "foobarns",
		},
		Spec: build.BuildRunSpec{
			Build: build.ReferencedBuild{
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
func (c *Catalog) DefaultClusterBuildStrategy() *build.ClusterBuildStrategy {
	return &build.ClusterBuildStrategy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foobar",
		},
	}
}

// DefaultNamespacedBuildStrategy returns a minimal BuildStrategy
// object with a inmutable name
func (c *Catalog) DefaultNamespacedBuildStrategy() *build.BuildStrategy {
	return &build.BuildStrategy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foobar",
		},
	}
}

// BuildRunWithNodeSelectorOverride returns a customized BuildRun object
// that defines a buildspec and overrides the nodeSelector
func (c *Catalog) BuildRunWithNodeSelectorOverride(buildRunName string, buildName string, nodeSelector map[string]string) *build.BuildRun {
	return &build.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildRunName,
		},
		Spec: build.BuildRunSpec{
			Build: build.ReferencedBuild{
				Name: &buildName,
				Spec: &build.BuildSpec{Strategy: build.Strategy{Name: "foobar"}},
			},
			NodeSelector: nodeSelector,
		},
		Status: build.BuildRunStatus{},
	}
}

// BuildRunWithTolerationsOverride returns a customized BuildRun object
// that defines a buildspec and overrides the tolerations
func (c *Catalog) BuildRunWithTolerationsOverride(buildRunName string, buildName string, tolerations []corev1.Toleration) *build.BuildRun {
	return &build.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildRunName,
		},
		Spec: build.BuildRunSpec{
			Build: build.ReferencedBuild{
				Name: &buildName,
				Spec: &build.BuildSpec{Strategy: build.Strategy{Name: "foobar"}},
			},
			Tolerations: tolerations,
		},
		Status: build.BuildRunStatus{},
	}
}

// BuildRunWithSchedulerNameOverride returns a customized BuildRun object
// that defines a buildspec and overrides the schedulerName
func (c *Catalog) BuildRunWithSchedulerNameOverride(buildRunName string, buildName string, schedulerName string) *build.BuildRun {
	return &build.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildRunName,
		},
		Spec: build.BuildRunSpec{
			Build: build.ReferencedBuild{
				Name: &buildName,
				Spec: &build.BuildSpec{Strategy: build.Strategy{Name: "foobar"}},
			},
			SchedulerName: &schedulerName,
		},
		Status: build.BuildRunStatus{},
	}
}

// BuildRunWithSucceededCondition returns a BuildRun with a single condition
// of the type Succeeded
func (c *Catalog) BuildRunWithSucceededCondition() *build.BuildRun {
	return &build.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foobar",
		},
		Status: build.BuildRunStatus{
			Conditions: build.Conditions{
				build.Condition{
					Type:    build.Succeeded,
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
func (c *Catalog) BuildRunWithSA(buildRunName string, buildName string, saName string) *build.BuildRun {
	return &build.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildRunName,
		},
		Spec: build.BuildRunSpec{
			Build: build.ReferencedBuild{
				Name: &buildName,
			},
			ServiceAccount: &saName,
		},
		Status: build.BuildRunStatus{},
	}
}

// BuildRunWithoutSA returns a buildrun without serviceAccountName and generate serviceAccount is false
func (c *Catalog) BuildRunWithoutSA(buildRunName string, buildName string) *build.BuildRun {
	return &build.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildRunName,
		},
		Spec: build.BuildRunSpec{
			Build: build.ReferencedBuild{
				Name: &buildName,
			},
			ServiceAccount: nil,
		},
	}
}

// BuildRunWithSAGenerate returns a customized BuildRun object that defines a
// service account
func (c *Catalog) BuildRunWithSAGenerate(buildRunName string, buildName string) *build.BuildRun {
	return &build.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildRunName,
		},
		Spec: build.BuildRunSpec{
			Build: build.ReferencedBuild{
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
func (c *Catalog) LoadBuildYAML(d []byte) (*build.Build, error) {
	b := &build.Build{}
	err := yaml.Unmarshal(d, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// LoadBuildWithNameAndStrategy returns a populated Build with name and a referenced strategy
func (c *Catalog) LoadBuildWithNameAndStrategy(name string, strategy string, d []byte) (*build.Build, error) {
	b := &build.Build{}
	err := yaml.Unmarshal(d, b)
	if err != nil {
		return nil, err
	}
	b.Name = name
	b.Spec.Strategy.Name = strategy
	return b, nil
}

// LoadBuildRunFromBytes returns a populated BuildRun
func (c *Catalog) LoadBuildRunFromBytes(d []byte) (*build.BuildRun, error) {
	b := &build.BuildRun{}
	err := yaml.Unmarshal(d, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// LoadBRWithNameAndRef returns a populated BuildRun with a name and a referenced Build
func (c *Catalog) LoadBRWithNameAndRef(name string, buildName string, d []byte) (*build.BuildRun, error) {
	b := &build.BuildRun{}
	err := yaml.Unmarshal(d, b)
	if err != nil {
		return nil, err
	}
	b.Name = name
	b.Spec.Build.Name = &buildName
	return b, nil
}

func (c *Catalog) LoadStandAloneBuildRunWithNameAndStrategy(name string, strategy *build.ClusterBuildStrategy, d []byte) (*build.BuildRun, error) {
	b := &build.BuildRun{}
	err := yaml.Unmarshal(d, b)
	if err != nil {
		return nil, err
	}
	b.Name = name
	b.Spec.Build.Spec.Strategy = build.Strategy{Kind: (*build.BuildStrategyKind)(&strategy.Kind), Name: strategy.Name}

	return b, nil
}

// LoadBuildStrategyFromBytes returns a populated BuildStrategy
func (c *Catalog) LoadBuildStrategyFromBytes(d []byte) (*build.BuildStrategy, error) {
	b := &build.BuildStrategy{}
	err := yaml.Unmarshal(d, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// LoadCBSWithName returns a populated ClusterBuildStrategy with a name
func (c *Catalog) LoadCBSWithName(name string, d []byte) (*build.ClusterBuildStrategy, error) {
	b := &build.ClusterBuildStrategy{}
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
func (c *Catalog) StubBuildAndPipelineRun(b *build.Build, pr *pipelineapi.PipelineRun) func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
	return func(context context.Context, nn types.NamespacedName, object client.Object, getOptions ...client.GetOption) error {
		switch object := object.(type) {
		case *build.Build:
			b.DeepCopyInto(object)
			return nil
		case *pipelineapi.PipelineRun:
			pr.DeepCopyInto(object)
			return nil
		}
		return errors.NewNotFound(schema.GroupResource{}, nn.Name)
	}
}
