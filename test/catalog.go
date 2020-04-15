package test

import (
	"context"

	. "github.com/onsi/gomega"
	build "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
		}
		return nil
	}
}
