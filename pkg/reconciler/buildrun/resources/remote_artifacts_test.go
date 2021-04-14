// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package resources

import (
	"fmt"
	"reflect"
	"testing"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

func TestRemoteArtifacts_renderRemoteArtifactsDownloadScript(t *testing.T) {
	url1 := "http://host/path/file.ext"
	url2 := "http://another/path/file.ext"

	tests := []struct {
		description string
		sources     []buildv1alpha1.BuildSource
		want        []string
	}{{
		description: "empty should not render a script",
		sources:     []buildv1alpha1.BuildSource{},
		want:        []string{},
	}, {
		description: "single build-source on default location",
		sources: []buildv1alpha1.BuildSource{{
			Name: "example",
			URL:  url1,
		}},
		want: []string{
			fmt.Sprintf("wget %s", url1),
		},
	}, {
		description: "multiple build-sources",
		sources: []buildv1alpha1.BuildSource{{
			Name: "example-1",
			URL:  url1,
		}, {
			Name: "example-2",
			URL:  url2,
		}},
		want: []string{
			fmt.Sprintf("wget %s", url1),
			fmt.Sprintf("wget %s", url2),
		},
	}}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got := renderRemoteArtifactsDownloadScript(tt.sources)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("renderRemoteArtifactsDownloadScript() = '%v', want '%v'", got, tt.want)
			}
		})
	}
}

func TestRemoteArtifacts_AmendTaskSpecWithRemoteArtifacts(t *testing.T) {
	cfg := &config.Config{RemoteArtifactsContainerImage: "busybox:latest"}
	spec := &v1beta1.TaskSpec{Steps: []v1beta1.Step{}}

	exampleURL := "http://host/path/file.ext"

	b := &buildv1alpha1.Build{Spec: buildv1alpha1.BuildSpec{
		Sources: &[]buildv1alpha1.BuildSource{{
			Name: "example",
			URL:  exampleURL,
		}},
	}}

	AmendTaskSpecWithRemoteArtifacts(cfg, spec, b)

	if len(spec.Steps) != 1 {
		t.Fatalf("remote-artifacts download step is not present")
	}
}
