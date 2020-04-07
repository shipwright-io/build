package test

import (
	build "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
