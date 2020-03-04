package e2e

import (
	operator "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// buildahBuild Test data setup
func buildahBuildTestData(ns string, identifier string) (*operator.Build, *operator.BuildStrategy) {

	truePtr := true

	exampleBuildStrategy := &operator.BuildStrategy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "buildah",
			Namespace: ns,
		},
		Spec: operator.BuildStrategySpec{
			BuildSteps: []operator.BuildStep{
				operator.BuildStep{
					Container: corev1.Container{
						Name:       "build",
						Image:      "quay.io/buildah/stable",
						WorkingDir: "/workspace/source",
						Command: []string{
							"buildah", "bud", "--tls-verify=false", "--layers", "-f", "$(build.dockerfile)", "-t", "$(build.output.image)", "$(build.pathContext)",
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "varlibcontainers",
								MountPath: "/var/lib/containers",
							},
						},
						SecurityContext: &corev1.SecurityContext{
							Privileged: &truePtr,
						},
					},
				},
			},
		},
	}

	dockerfile := "Dockerfile"
	outputPath := "image-registry.openshift-image-registry.svc:5000/example/taxi-app"
	pathContext := "."
	// create build custom resource
	exampleBuild := &operator.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      identifier,
			Namespace: ns,
		},
		Spec: operator.BuildSpec{
			Source: operator.GitSource{
				URL:        "https://github.com/sbose78/taxi",
				ContextDir: &pathContext,
			},
			StrategyRef: metav1.ObjectMeta{
				Name:      "buildah",
				Namespace: ns,
			},
			Dockerfile: &dockerfile,
			Output: operator.Output{
				ImageURL: outputPath,
			},
		},
	}

	return exampleBuild, exampleBuildStrategy
}
