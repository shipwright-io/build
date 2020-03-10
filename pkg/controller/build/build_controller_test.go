package build

import (
	"github.com/stretchr/testify/assert"
	"reflect"
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
							Kind: "BuildStrategy",
						},
					},
				},
			},
			want: buildv1alpha1.StrategyRef{
				Name: "my-strategy",
				Kind: "BuildStrategy",
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
							Kind: "ClusterBuildStrategy",
						},
					},
				},
			},
			want: buildv1alpha1.StrategyRef{
				Name: "my-clusterstrategy",
				Kind: "ClusterBuildStrategy",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, reflect.DeepEqual(tt.args.instance.Spec.StrategyRef, &tt.want))
		})
	}
}
