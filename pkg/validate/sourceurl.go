// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"fmt"

	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/ctxlog"
	"github.com/shipwright-io/build/pkg/git"
)

// SourceURLRef contains all required fields
// to validate a Build spec source definition
type SourceURLRef struct {
	Build  *buildapi.Build
	Client client.Client
}

func NewSourceURL(client client.Client, build *buildapi.Build) *SourceURLRef {
	return &SourceURLRef{build, client}
}

// ValidatePath implements BuildPath interface and validates
// that the spec.source.url exists. This validation only applies
// to endpoints that do not require authentication.
func (s SourceURLRef) ValidatePath(ctx context.Context) error {
	if s.Build.Spec.Source != nil && s.Build.Spec.Source.Type == buildapi.GitType && s.Build.Spec.Source.Git != nil {
		Git := s.Build.Spec.Source.Git
		if Git.CloneSecret == nil {
			switch s.Build.GetAnnotations()[buildapi.AnnotationBuildVerifyRepository] {
			case "true":
				if err := git.ValidateGitURLExists(ctx, Git.URL); err != nil {
					s.MarkBuildStatus(s.Build, buildapi.RemoteRepositoryUnreachable, err.Error())
					return err
				}
			case "", "false":
				ctxlog.Info(ctx, fmt.Sprintf("the annotation %s is set to %s, nothing to do", buildapi.AnnotationBuildVerifyRepository, s.Build.GetAnnotations()[buildapi.AnnotationBuildVerifyRepository]), namespace, s.Build.Namespace, name, s.Build.Name)

			default:
				var annoErr = fmt.Errorf("the annotation %s was not properly defined, supported values are true or false", buildapi.AnnotationBuildVerifyRepository)
				ctxlog.Error(ctx, annoErr, namespace, s.Build.Namespace, name, s.Build.Name)
				s.MarkBuildStatus(s.Build, buildapi.RemoteRepositoryUnreachable, annoErr.Error())
				return annoErr
			}
		}
	}

	return nil
}

// MarkBuildStatus updates a Build Status fields
func (s SourceURLRef) MarkBuildStatus(b *buildapi.Build, reason buildapi.BuildReason, msg string) {
	b.Status.Reason = buildapi.BuildReasonPtr(reason)
	b.Status.Message = pointer.String(msg)
}
