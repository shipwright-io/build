package buildrun

import (
	"reflect"
	"testing"

	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"

	corev1 "k8s.io/api/core/v1"
)

func TestApplyCredentials(t *testing.T) {

	type args struct {
		build  *buildv1alpha1.Build
		buildRun  *buildv1alpha1.BuildRun
		serviceAccount *corev1.ServiceAccount
	}
	tests := []struct {
		name string
		args args
		want *corev1.ServiceAccount
	}{
		{
			"secrets were not present",
			args{
				build: &buildv1alpha1.Build{
					Spec: buildv1alpha1.BuildSpec{
						Source: buildv1alpha1.GitSource{
							URL: "a/b/c",
							SecretRef: &corev1.LocalObjectReference{
								Name: "secret_a",
							},
						},
						BuilderImage: &buildv1alpha1.Image{
							ImageURL: "quay.io/namespace/image",
							SecretRef: &corev1.LocalObjectReference{
								Name: "secret_docker.io",
							},
						},
						Output: buildv1alpha1.Image{
							ImageURL: "quay.io/namespace/image",
							SecretRef: &corev1.LocalObjectReference{
								Name: "secret_quay.io",
							},
						},
					},
				},
				serviceAccount: &corev1.ServiceAccount{
					Secrets: []corev1.ObjectReference{
						{Name: "secret_b"}, {Name: "secret_c"},
					},
				},
			},
			&corev1.ServiceAccount{
				Secrets: []corev1.ObjectReference{
					{Name: "secret_b"}, {Name: "secret_c"}, {Name: "secret_a"}, {Name: "secret_docker.io"}, {Name: "secret_quay.io"},
				},
			},
		},
		{
			"secret was already present",
			args{
				build: &buildv1alpha1.Build{
					Spec: buildv1alpha1.BuildSpec{
						Source: buildv1alpha1.GitSource{
							URL: "a/b/c",
							SecretRef: &corev1.LocalObjectReference{
								Name: "secret_a",
							},
						},
					},
				},
				buildRun: &buildv1alpha1.BuildRun{},
				serviceAccount: &corev1.ServiceAccount{
					Secrets: []corev1.ObjectReference{
						{Name: "secret_b"}, {Name: "secret_a"},
					},
				},
			},
			&corev1.ServiceAccount{
				Secrets: []corev1.ObjectReference{
					{Name: "secret_b"}, {Name: "secret_a"},
				},
			},
		},
		{
			"public repo, no source secret",
			args{
				build: &buildv1alpha1.Build{
					Spec: buildv1alpha1.BuildSpec{
						Source: buildv1alpha1.GitSource{
							URL:       "a/b/c",
							SecretRef: nil,
						},
					},
				},
				buildRun: &buildv1alpha1.BuildRun{},
				serviceAccount: &corev1.ServiceAccount{
					Secrets: []corev1.ObjectReference{
						{Name: "secret_b"}, {Name: "secret_a"},
					},
				},
			},
			&corev1.ServiceAccount{
				Secrets: []corev1.ObjectReference{
					{Name: "secret_b"}, {Name: "secret_a"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := applyCredentials(tt.args.build, tt.args.serviceAccount); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("applyCredentials() = %v, want %v", got, tt.want)
			}
		})
	}

}
