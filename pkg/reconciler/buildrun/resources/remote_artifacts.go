// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"fmt"
	"strings"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	v1 "k8s.io/api/core/v1"
)

// renderRemoteArtifactsDownloadScript returns a slice of commands, a shell script, based on informed
// BuildSources slice. Scripting lines are bind together with "&&".
func renderRemoteArtifactsDownloadScript(sources []buildv1alpha1.BuildSource) []string {
	script := []string{}
	for _, source := range sources {
		cmd := fmt.Sprintf("wget %s", source.URL)
		script = append(script, cmd)
	}
	return script
}

// remoteArtifactStep generates a Tekton Step to execute the remote artifacts download script.
func remoteArtifactStep(b *buildv1alpha1.Build, image string) v1beta1.Step {
	script := renderRemoteArtifactsDownloadScript(*b.Spec.Sources)
	container := v1.Container{
		Name:       "remote-artifacts",
		Image:      image,
		WorkingDir: fmt.Sprintf("$(params.%s%s)", prefixParamsResults, paramSourceRoot),
		Command:    []string{"/bin/sh"},
		Args:       append([]string{"-e", "-x", "-c"}, strings.Join(script, " ; ")),
	}
	return v1beta1.Step{Container: container}
}

// AmendTaskSpecWithRemoteArtifacts will amend a TaskSpec reference with remote-artifacts download
// step, taking place before all others.
func AmendTaskSpecWithRemoteArtifacts(
	cfg *config.Config,
	spec *v1beta1.TaskSpec,
	b *buildv1alpha1.Build,
) {
	step := remoteArtifactStep(b, cfg.RemoteArtifactsContainerImage)
	steps := append([]v1beta1.Step{step}, spec.Steps...)
	spec.Steps = steps
}

// IsSourcesDefined checks if ``spec.sources` is defined, returns a boolean.
func IsSourcesDefined(b *buildv1alpha1.Build) bool {
	return b.Spec.Sources != nil && len(*b.Spec.Sources) > 0
}
