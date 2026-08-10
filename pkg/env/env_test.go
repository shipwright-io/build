// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package env_test

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"

	_ "github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/env"
)

func TestIsForbiddenEnvVar(t *testing.T) {
	tests := []struct {
		name      string
		envName   string
		forbidden bool
	}{
		{"LD_PRELOAD is forbidden", "LD_PRELOAD", true},
		{"LD_LIBRARY_PATH is forbidden", "LD_LIBRARY_PATH", true},
		{"LD_AUDIT is forbidden", "LD_AUDIT", true},
		{"LD_DEBUG is forbidden", "LD_DEBUG", true},
		{"LD_PROFILE is forbidden", "LD_PROFILE", true},
		{"LD_ prefix catches unknown vars", "LD_CUSTOM", true},
		{"BASH_ENV is forbidden", "BASH_ENV", true},
		{"BASH_FUNC_ prefix is forbidden", "BASH_FUNC_exploit%%", true},
		{"ENV is forbidden", "ENV", true},
		{"CDPATH is forbidden", "CDPATH", true},
		{"PYTHONSTARTUP is forbidden", "PYTHONSTARTUP", true},
		{"PERL5OPT is forbidden", "PERL5OPT", true},
		{"PERLLIB is forbidden", "PERLLIB", true},
		{"PERL5LIB is forbidden", "PERL5LIB", true},
		{"RUBYOPT is forbidden", "RUBYOPT", true},
		{"NODE_OPTIONS is forbidden", "NODE_OPTIONS", true},
		{"MY_VAR is allowed", "MY_VAR", false},
		{"BUILD_LOGLEVEL is allowed", "BUILD_LOGLEVEL", false},
		{"HTTP_PROXY is allowed", "HTTP_PROXY", false},
		{"HOME is allowed", "HOME", false},
		{"PATH is allowed", "PATH", false},
		{"empty string is allowed", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := env.IsForbiddenEnvVar(tt.envName)
			if got != tt.forbidden {
				t.Errorf("IsForbiddenEnvVar(%q) = %v, want %v", tt.envName, got, tt.forbidden)
			}
		})
	}
}

func TestMergeEnvVars(t *testing.T) {
	type args struct {
		new             []corev1.EnvVar
		into            []corev1.EnvVar
		overwriteValues bool
	}
	tests := []struct {
		name    string
		args    args
		want    []corev1.EnvVar
		wantErr bool
	}{
		{
			name: "should not fail with nil inputs",
			args: args{
				new:             nil,
				into:            nil,
				overwriteValues: false,
			},
			want:    []corev1.EnvVar{},
			wantErr: false,
		},
		{
			name: "empty new and into should return empty",
			args: args{
				new:             []corev1.EnvVar{},
				into:            []corev1.EnvVar{},
				overwriteValues: true,
			},
			want:    []corev1.EnvVar{},
			wantErr: false,
		},
		{
			name: "empty new should return into",
			args: args{
				new: []corev1.EnvVar{},
				into: []corev1.EnvVar{
					{Name: "ONE", Value: "oneValue"},
					{Name: "TWO", Value: "twoValue"},
				},
				overwriteValues: true,
			},
			want: []corev1.EnvVar{
				{Name: "ONE", Value: "oneValue"},
				{Name: "TWO", Value: "twoValue"},
			},
			wantErr: false,
		},
		{
			name: "empty into should return new",
			args: args{
				new: []corev1.EnvVar{
					{Name: "ONE", Value: "oneValue"},
					{Name: "TWO", Value: "twoValue"},
				},
				into:            []corev1.EnvVar{},
				overwriteValues: true,
			},
			want: []corev1.EnvVar{
				{Name: "ONE", Value: "oneValue"},
				{Name: "TWO", Value: "twoValue"},
			},
			wantErr: false,
		},
		{
			name: "duplicate names should fail with overwriteValues false",
			args: args{
				new: []corev1.EnvVar{
					{Name: "TWO", Value: "twoValueNew"},
				},
				into: []corev1.EnvVar{
					{Name: "ONE", Value: "oneValue"},
					{Name: "TWO", Value: "twoValue"},
				},
				overwriteValues: false,
			},
			want: []corev1.EnvVar{
				{Name: "ONE", Value: "oneValue"},
				{Name: "TWO", Value: "twoValue"},
			},
			wantErr: true,
		},
		{
			name: "duplicate names should fail with overwriteValues false using valueFrom",
			args: args{
				new: []corev1.EnvVar{
					{
						Name: "TWO",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "my-field-path",
							},
						},
					},
				},
				into: []corev1.EnvVar{
					{Name: "ONE", Value: "oneValue"},
					{Name: "TWO", Value: "twoValue"},
				},
				overwriteValues: false,
			},
			want: []corev1.EnvVar{
				{Name: "ONE", Value: "oneValue"},
				{Name: "TWO", Value: "twoValue"},
			},
			wantErr: true,
		},
		{
			name: "duplicate names should succeed with overwriteValues true",
			args: args{
				new: []corev1.EnvVar{
					{Name: "TWO", Value: "newTwoValue"},
				},
				into: []corev1.EnvVar{
					{Name: "ONE", Value: "oneValue"},
					{Name: "TWO", Value: "twoValue"},
				},
				overwriteValues: true,
			},
			want: []corev1.EnvVar{
				{Name: "ONE", Value: "oneValue"},
				{Name: "TWO", Value: "newTwoValue"},
			},
			wantErr: false,
		},
		{
			name: "duplicate names should succeed with overwriteValues true using valueFrom",
			args: args{
				new: []corev1.EnvVar{
					{
						Name: "TWO",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "my-field-path",
							},
						},
					},
				},
				into: []corev1.EnvVar{
					{Name: "ONE", Value: "oneValue"},
					{Name: "TWO", Value: "twoValue"},
				},
				overwriteValues: true,
			},
			want: []corev1.EnvVar{
				{Name: "ONE", Value: "oneValue"},
				{
					Name: "TWO",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "my-field-path",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "non-duplicate should succeed with overwriteValues false",
			args: args{
				new: []corev1.EnvVar{
					{Name: "THREE", Value: "threeValue"},
					{Name: "FOUR", Value: "fourValue"},
				},
				into: []corev1.EnvVar{
					{Name: "ONE", Value: "oneValue"},
					{Name: "TWO", Value: "twoValue"},
				},
				overwriteValues: false,
			},
			want: []corev1.EnvVar{
				{Name: "ONE", Value: "oneValue"},
				{Name: "TWO", Value: "twoValue"},
				{Name: "THREE", Value: "threeValue"},
				{Name: "FOUR", Value: "fourValue"},
			},
			wantErr: false,
		},
		{
			name: "non-duplicate should succeed with overwriteValues false using valueFrom",
			args: args{
				new: []corev1.EnvVar{
					{
						Name: "THREE",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "my-field-path",
							},
						},
					},
					{Name: "FOUR", Value: "fourValue"},
				},
				into: []corev1.EnvVar{
					{Name: "ONE", Value: "oneValue"},
					{Name: "TWO", Value: "twoValue"},
				},
				overwriteValues: false,
			},
			want: []corev1.EnvVar{
				{Name: "ONE", Value: "oneValue"},
				{Name: "TWO", Value: "twoValue"},
				{
					Name: "THREE",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "my-field-path",
						},
					},
				},
				{Name: "FOUR", Value: "fourValue"},
			},
			wantErr: false,
		},
		{
			name: "non-duplicate should succeed with overwriteValues true",
			args: args{
				new: []corev1.EnvVar{
					{Name: "THREE", Value: "threeValue"},
					{Name: "FOUR", Value: "fourValue"},
				},
				into: []corev1.EnvVar{
					{Name: "ONE", Value: "oneValue"},
					{Name: "TWO", Value: "twoValue"},
				},
				overwriteValues: true,
			},
			want: []corev1.EnvVar{
				{Name: "ONE", Value: "oneValue"},
				{Name: "TWO", Value: "twoValue"},
				{Name: "THREE", Value: "threeValue"},
				{Name: "FOUR", Value: "fourValue"},
			},
			wantErr: false,
		},
		{
			name: "non-duplicate should succeed with overwriteValues true using valueFrom",
			args: args{
				new: []corev1.EnvVar{
					{
						Name: "THREE",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "my-field-path",
							},
						},
					},
					{Name: "FOUR", Value: "fourValue"},
				},
				into: []corev1.EnvVar{
					{Name: "ONE", Value: "oneValue"},
					{Name: "TWO", Value: "twoValue"},
				},
				overwriteValues: true,
			},
			want: []corev1.EnvVar{
				{Name: "ONE", Value: "oneValue"},
				{Name: "TWO", Value: "twoValue"},
				{
					Name: "THREE",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "my-field-path",
						},
					},
				},
				{Name: "FOUR", Value: "fourValue"},
			},
			wantErr: false,
		},
		{
			name: "forbidden env var LD_PRELOAD should fail",
			args: args{
				new: []corev1.EnvVar{
					{Name: "LD_PRELOAD", Value: "/tmp/malicious.so"},
				},
				into: []corev1.EnvVar{
					{Name: "ONE", Value: "oneValue"},
				},
				overwriteValues: true,
			},
			want: []corev1.EnvVar{
				{Name: "ONE", Value: "oneValue"},
			},
			wantErr: true,
		},
		{
			name: "forbidden env var with LD_ prefix should fail",
			args: args{
				new: []corev1.EnvVar{
					{Name: "LD_LIBRARY_PATH", Value: "/tmp"},
				},
				into: []corev1.EnvVar{
					{Name: "ONE", Value: "oneValue"},
				},
				overwriteValues: true,
			},
			want: []corev1.EnvVar{
				{Name: "ONE", Value: "oneValue"},
			},
			wantErr: true,
		},
		{
			name: "forbidden env var BASH_ENV should fail",
			args: args{
				new: []corev1.EnvVar{
					{Name: "BASH_ENV", Value: "/tmp/evil.sh"},
				},
				into: []corev1.EnvVar{
					{Name: "ONE", Value: "oneValue"},
				},
				overwriteValues: false,
			},
			want: []corev1.EnvVar{
				{Name: "ONE", Value: "oneValue"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := env.MergeEnvVars(tt.args.new, tt.args.into, tt.args.overwriteValues)
			if (err != nil) != tt.wantErr {
				t.Errorf("MergeEnvVars() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeEnvVars() = %v, want %v", got, tt.want)
			}
		})
	}
}
