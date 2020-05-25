package test

import (
	"context"
	"strconv"

	knativev1beta1 "knative.dev/pkg/apis/duck/v1beta1"

	. "github.com/onsi/gomega"
	build "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	crc "sigs.k8s.io/controller-runtime/pkg/client"
)

// Catalog allows you to access helper functions
type Catalog struct{}

// BuildWithClusterBuildStrategy gives you an specific Build CRD
func (c *Catalog) BuildWithClusterBuildStrategy(name string, ns string, strategyName string, secretName string) *build.Build {
	buildStrategy := build.ClusterBuildStrategyKind
	return &build.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: build.BuildSpec{
			Source: build.GitSource{
				URL: "foobar",
			},
			StrategyRef: &build.StrategyRef{
				Name: strategyName,
				Kind: &buildStrategy,
			},
			Output: build.Image{
				ImageURL: "foobar",
				SecretRef: &corev1.LocalObjectReference{
					Name: secretName,
				},
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
			Source: build.GitSource{
				URL: "foobar",
			},
			StrategyRef: &build.StrategyRef{
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
			Source: build.GitSource{
				URL: "foobar",
			},
			StrategyRef: &build.StrategyRef{
				Name: strategyName,
			},
		},
	}
}

// ClusterBuildStrategyList to support tests
func (c *Catalog) ClusterBuildStrategyList(name string) *build.ClusterBuildStrategyList {
	return &build.ClusterBuildStrategyList{
		Items: []build.ClusterBuildStrategy{
			build.ClusterBuildStrategy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: "build-examples",
				},
			},
		},
	}
}

// FakeClusterBuildStrategyList to support tests
func (c *Catalog) FakeClusterBuildStrategyList() *build.ClusterBuildStrategyList {
	return &build.ClusterBuildStrategyList{
		Items: []build.ClusterBuildStrategy{
			build.ClusterBuildStrategy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foobar",
					Namespace: "build-examples",
				},
			},
		},
	}
}

// BuildStrategyList to support tests
func (c *Catalog) BuildStrategyList(name string, ns string) *build.BuildStrategyList {
	return &build.BuildStrategyList{
		Items: []build.BuildStrategy{
			build.BuildStrategy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: ns,
				},
			},
		},
	}
}

// FakeBuildStrategyList to support tests
func (c *Catalog) FakeBuildStrategyList() *build.BuildStrategyList {
	return &build.BuildStrategyList{
		Items: []build.BuildStrategy{
			build.BuildStrategy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foobar",
				},
			},
		},
	}
}

// FakeSecretList to support tests
func (c *Catalog) FakeSecretList() corev1.SecretList {
	return corev1.SecretList{
		Items: []corev1.Secret{
			corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foobar",
				},
			},
		},
	}
}

// SecretList to support tests
func (c *Catalog) SecretList(name string) corev1.SecretList {
	return corev1.SecretList{
		Items: []corev1.Secret{
			corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
			},
		},
	}
}

// StubFunc is used to simulate the status of the Build
// after a .Status().Update() call in the controller. This
// receives a parameter to return an specific status state
func (c *Catalog) StubFunc(status corev1.ConditionStatus, reason string) func(context context.Context, object runtime.Object, _ ...crc.UpdateOption) error {
	return func(context context.Context, object runtime.Object, _ ...crc.UpdateOption) error {
		switch object := object.(type) {
		case *build.Build:
			Expect(object.Status.Registered).To(Equal(status))
			Expect(object.Status.Reason).To(ContainSubstring(reason))
			if object.Annotations != nil && object.Annotations[build.AnnotationBuildRunDeletion] == "true" {
				Expect(object.Finalizers[0]).To(Equal(build.BuildFinalizer))
			}
		}
		return nil
	}
}

// StubBuildRunStatus asserts Status fields on a BuildRun resource
func (c *Catalog) StubBuildRunStatus(reason string, name *string, status corev1.ConditionStatus, buildSample *build.Build) func(context context.Context, object runtime.Object, _ ...crc.UpdateOption) error {
	return func(context context.Context, object runtime.Object, _ ...crc.UpdateOption) error {
		switch object := object.(type) {
		case *build.BuildRun:
			Expect(object.Status.Reason).To(Equal(reason))
			Expect(object.Status.Succeeded).To(Equal(status))
			Expect(object.Status.LatestTaskRunRef).To(Equal(name))
			if object.Status.BuildSpec != nil {
				Expect(*object.Status.BuildSpec).To(Equal(buildSpec))
			}
		}
		return nil
	}
}

// StubBuildRunLabel asserts Label fields on a BuildRun resource
func (c *Catalog) StubBuildRunLabel(buildSample *build.Build) func(context context.Context, object runtime.Object, _ ...crc.UpdateOption) error {
	return func(context context.Context, object runtime.Object, _ ...crc.UpdateOption) error {
		switch object := object.(type) {
		case *build.BuildRun:
			Expect(object.Labels[build.LabelBuild]).To(Equal(buildSample.Name))
			Expect(object.Labels[build.LabelBuildGeneration]).To(Equal(strconv.FormatInt(buildSample.Generation, 10)))
		}
		return nil
	}
}

// StubBuildRunGetWithSA simulates the output of client GET calls
// for the BuildRun unit tests
func (c *Catalog) StubBuildRunGetWithSA(
	b *build.Build,
	br *build.BuildRun,
	sa *corev1.ServiceAccount,
) func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
	return func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
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

// StubBuildRunGetWithSAandStrategies simulates the ouput of client GET
// calls for the BuildRun unit tests
func (c *Catalog) StubBuildRunGetWithSAandStrategies(
	b *build.Build,
	br *build.BuildRun,
	sa *corev1.ServiceAccount,
	cb *buildv1alpha1.ClusterBuildStrategy,
	bs *buildv1alpha1.BuildStrategy,
) func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
	return func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
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
		case *buildv1alpha1.ClusterBuildStrategy:
			cb.DeepCopyInto(object)
			return nil
		case *buildv1alpha1.BuildStrategy:
			bs.DeepCopyInto(object)
			return nil
		}
		return errors.NewNotFound(schema.GroupResource{}, nn.Name)
	}
}

// DefaultTaskRunList returns a minimal tekton TaskRunList
func (c *Catalog) DefaultTaskRunList(tr *v1beta1.TaskRun) *v1beta1.TaskRunList {
	return &v1beta1.TaskRunList{
		Items: []v1beta1.TaskRun{*tr},
	}
}

// DefaultTaskRunWithStatus returns a minimal tektont TaskRun with an Status
func (c *Catalog) DefaultTaskRunWithStatus(trName string, status corev1.ConditionStatus, reason string) *v1beta1.TaskRun {
	return &v1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: trName,
		},
		Spec: v1beta1.TaskRunSpec{},
		Status: v1beta1.TaskRunStatus{
			Status: knativev1beta1.Status{
				Conditions: knativev1beta1.Conditions{
					{
						Reason: reason,
						Status: status,
					},
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
			StrategyRef: &build.StrategyRef{
				Name: strategyName,
				Kind: &strategyKind,
			},
		},
	}
}

// DefaultBuildRun returns a minimal BuildRun object
func (c *Catalog) DefaultBuildRun(buildRunName string, buildName string) *build.BuildRun {
	return &build.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildRunName,
		},
		Spec: build.BuildRunSpec{
			BuildRef: &build.BuildRef{
				Name: buildName,
			},
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

// DefaultClusterBuildStrategy returns a minimal ClusterBuildStrategy
// object with a inmutable name
func (c *Catalog) DefaultClusterBuildStrategy() *buildv1alpha1.ClusterBuildStrategy {
	return &buildv1alpha1.ClusterBuildStrategy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foobar",
		},
	}
}

// DefaultNamespacedBuildStrategy returns a minimal BuildStrategy
// object with a inmutable name
func (c *Catalog) DefaultNamespacedBuildStrategy() *buildv1alpha1.BuildStrategy {
	return &buildv1alpha1.BuildStrategy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foobar",
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
			BuildRef: &build.BuildRef{
				Name: buildName,
			},
			ServiceAccount: &build.ServiceAccount{
				Name:     &saName,
				Generate: false,
			},
		},
	}
}
