// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package validate

import (
	"context"
	"testing"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

func TestSourcesRef_ValidatePath(t *testing.T) {
	testCases := []struct {
		description string
		expectError bool
		b           *build.Build
	}{{
		description: "empty sources slice",
		expectError: false,
		b:           &build.Build{},
	}, {
		description: "name is not informed",
		expectError: true,
		b: &build.Build{Spec: build.BuildSpec{Sources: []build.BuildSource{{
			Name: "",
		}}}},
	}, {
		description: "URL is not informed",
		expectError: true,
		b: &build.Build{Spec: build.BuildSpec{Sources: []build.BuildSource{{
			Name: "name",
			URL:  "",
		}}}},
	}, {
		description: "invalid URL",
		expectError: true,
		b: &build.Build{Spec: build.BuildSpec{Sources: []build.BuildSource{{
			Name: "name",
			URL:  "invalid URL",
		}}}},
	}}

	for _, tc := range testCases {
		s := &SourcesRef{Build: tc.b}
		ctx := context.TODO()
		err := s.ValidatePath(ctx)

		if (tc.expectError && err == nil) || (!tc.expectError && err != nil) {
			t.Fatalf("%s: expectError='%v', err='%v'", tc.description, tc.expectError, err)
		}
	}
}
