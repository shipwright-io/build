package build

import (
	"testing"

	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_getBuildStrategyNamespace(t *testing.T) {
	type args struct {
		instance *buildv1alpha1.Build
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "different namespace",
			args: args{
				instance: &buildv1alpha1.Build{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "build-sample",
						Namespace: "build-ns",
					},
					Spec: buildv1alpha1.BuildSpec{
						StrategyRef: metav1.ObjectMeta{
							Name:      "my-strategy",
							Namespace: "buildstrategy-ns",
						},
					},
				},
			},
			want: "buildstrategy-ns",
		},
		{
			name: "same namespace",
			args: args{
				instance: &buildv1alpha1.Build{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "build-sample",
						Namespace: "build-ns",
					},
					Spec: buildv1alpha1.BuildSpec{
						StrategyRef: metav1.ObjectMeta{
							Name:      "my-strategy",
							Namespace: "build-ns",
						},
					},
				},
			},
			want: "build-ns",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getBuildStrategyNamespace(tt.args.instance); got != tt.want {
				t.Errorf("getBuildStrategyNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}
