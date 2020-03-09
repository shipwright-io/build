package build

import (
	"reflect"
	"testing"

	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"

	corev1 "k8s.io/api/core/v1"
)

func TestApplyCredentials(t *testing.T) {

	type args struct {
		buildInstance  *buildv1alpha1.Build
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
				buildInstance: &buildv1alpha1.Build{
					Spec: buildv1alpha1.BuildSpec{
						Source: buildv1alpha1.GitSource{
							URL: "a/b/c",
							SecretRef: &corev1.LocalObjectReference{
								Name: "secret_a",
							},
						},
						Output: buildv1alpha1.Image{
							ImageURL: "quay.io/namespace/image",
							SecretRef: &corev1.LocalObjectReference{
								Name: "secret_quay.io",
							},
						},
						BuilderImage: &buildv1alpha1.Image{
							ImageURL: "quay.io/namespace/image",
							SecretRef: &corev1.LocalObjectReference{
								Name: "secret_docker.io",
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
					{Name: "secret_b"}, {Name: "secret_c"}, {Name: "secret_a"}, {Name: "secret_quay.io"}, {Name: "secret_docker.io"},
				},
			},
		},
		{
			"secret was already present",
			args{
				buildInstance: &buildv1alpha1.Build{
					Spec: buildv1alpha1.BuildSpec{
						Source: buildv1alpha1.GitSource{
							URL: "a/b/c",
							SecretRef: &corev1.LocalObjectReference{
								Name: "secret_a",
							},
						},
					},
				},
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
				buildInstance: &buildv1alpha1.Build{
					Spec: buildv1alpha1.BuildSpec{
						Source: buildv1alpha1.GitSource{
							URL:       "a/b/c",
							SecretRef: nil,
						},
					},
				},
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
			if got := applyCredentials(tt.args.buildInstance, tt.args.serviceAccount); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("applyCredentials() = %v, want %v", got, tt.want)
			}
		})
	}

}
