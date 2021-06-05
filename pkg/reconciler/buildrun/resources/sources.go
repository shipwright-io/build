// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources/sources"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

// AmendTaskSpecWithSources adds steps, results and volumes for spec.source and spec.sources
func AmendTaskSpecWithSources(
	cfg *config.Config,
	taskSpec *v1beta1.TaskSpec,
	build *buildv1alpha1.Build,
) error {
	// create the step for spec.source, this is always Git
	sources.AppendGitSourceStep(cfg, taskSpec, build.Spec.Source, "default")

	// create the step for spec.sources, this will eventually change into different steps depending on the type of the source
	if build.Spec.Sources != nil {
		for _, source := range *build.Spec.Sources {
			switch source.Type {
			case buildv1alpha1.BuildSourceTypeGit:
				if err := sources.AppendGitStep(cfg, taskSpec, source); err != nil {
					return err
				}
			case buildv1alpha1.BuildSourceTypeHTTP:
				if err := sources.AppendHTTPStep(cfg, taskSpec, source); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
