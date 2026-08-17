// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestSingleValueOmitsNilFields verifies that the mutually-exclusive optional
// fields of SingleValue are omitted from the serialized output when unset,
// instead of being rendered as explicit nulls (e.g. "configMapValue": null).
func TestSingleValueOmitsNilFields(t *testing.T) {
	value := "BUILD_VERSION=1.0.0"

	tests := []struct {
		name        string
		param       ParamValue
		wantPresent []string
		wantAbsent  []string
	}{
		{
			name:        "literal value set",
			param:       ParamValue{Name: "build-args", SingleValue: &SingleValue{Value: &value}},
			wantPresent: []string{`"value":"BUILD_VERSION=1.0.0"`},
			wantAbsent:  []string{`"configMapValue"`, `"secretValue"`},
		},
		{
			name:        "configMap value set",
			param:       ParamValue{Name: "p", SingleValue: &SingleValue{ConfigMapValue: &ObjectKeyRef{Name: "cm", Key: "k"}}},
			wantPresent: []string{`"configMapValue"`},
			wantAbsent:  []string{`"value"`, `"secretValue"`},
		},
		{
			name:        "secret value set",
			param:       ParamValue{Name: "p", SingleValue: &SingleValue{SecretValue: &ObjectKeyRef{Name: "s", Key: "k"}}},
			wantPresent: []string{`"secretValue"`},
			wantAbsent:  []string{`"value"`, `"configMapValue"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := json.Marshal(tt.param)
			if err != nil {
				t.Fatalf("failed to marshal ParamValue: %v", err)
			}
			got := string(out)
			for _, want := range tt.wantPresent {
				if !strings.Contains(got, want) {
					t.Errorf("expected %q in output, got: %s", want, got)
				}
			}
			for _, absent := range tt.wantAbsent {
				if strings.Contains(got, absent) {
					t.Errorf("expected %q to be omitted, got: %s", absent, got)
				}
			}
		})
	}
}
