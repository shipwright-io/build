package build

import (
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"

	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO this unit test should be refined by using the operator-sdk fake client
// After separate build definition from build run: https://github.com/redhat-developer/build/issues/65
func Test_getBuildByDifferentBuildStrategy(t *testing.T) {
	type args struct {
		instance *buildv1alpha1.Build
	}
	buildStrategy := buildv1alpha1.ClusterBuildStrategyKind
	clustertBuildStrategy := buildv1alpha1.ClusterBuildStrategyKind
	wrongBuildStrategy := buildv1alpha1.BuildStrategyKind("WrongBuildStrategy")
	var tests = []struct {
		name string
		args args
		want buildv1alpha1.StrategyRef
	}{
		{
			name: "namespaced scope buildstrategy",
			args: args{
				instance: &buildv1alpha1.Build{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "build-sample",
						Namespace: "build-ns",
					},
					Spec: buildv1alpha1.BuildSpec{
						StrategyRef: &buildv1alpha1.StrategyRef{
							Name: "my-strategy",
							Kind: &buildStrategy,
						},
					},
				},
			},
			want: buildv1alpha1.StrategyRef{
				Name: "my-strategy",
				Kind: &buildStrategy,
			},
		},
		{
			name: "empty buildstrategy",
			args: args{
				instance: &buildv1alpha1.Build{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "build-sample",
						Namespace: "build-ns",
					},
					Spec: buildv1alpha1.BuildSpec{
						StrategyRef: &buildv1alpha1.StrategyRef{
							Name: "my-strategy",
						},
					},
				},
			},
			want: buildv1alpha1.StrategyRef{
				Name: "my-strategy",
			},
		},
		{
			name: "cluster scope buildstrategy",
			args: args{
				instance: &buildv1alpha1.Build{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "build-sample",
						Namespace: "build-ns",
					},
					Spec: buildv1alpha1.BuildSpec{
						StrategyRef: &buildv1alpha1.StrategyRef{
							Name: "my-clusterstrategy",
							Kind: &clustertBuildStrategy,
						},
					},
				},
			},
			want: buildv1alpha1.StrategyRef{
				Name: "my-clusterstrategy",
				Kind: &clustertBuildStrategy,
			},
		},
		{
			name: "wrong buildstrategy",
			args: args{
				instance: &buildv1alpha1.Build{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "build-sample",
						Namespace: "build-ns",
					},
					Spec: buildv1alpha1.BuildSpec{
						StrategyRef: &buildv1alpha1.StrategyRef{
							Name: "my-strategy",
							Kind: &wrongBuildStrategy,
						},
					},
				},
			},
			want: buildv1alpha1.StrategyRef{
				Name: "my-strategy",
				Kind: &wrongBuildStrategy,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, reflect.DeepEqual(tt.args.instance.Spec.StrategyRef, &tt.want))
		})
	}
}
