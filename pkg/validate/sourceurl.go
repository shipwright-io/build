// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"fmt"

	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/ctxlog"
	"github.com/shipwright-io/build/pkg/git"
)

// SourceURLRef contains all required fields
// to validate a Build spec source definition
type SourceURLRef struct {
	Build  *build.Build
	Client client.Client
}

func NewSourceURL(client client.Client, build *build.Build) *SourceURLRef {
	return &SourceURLRef{build, client}
}

// ValidatePath implements BuildPath interface and validates
// that the spec.source.url exists. This validation only applies
// to endpoints that do not require authentication.
func (s SourceURLRef) ValidatePath(ctx context.Context) error {
	if s.Build.Spec.Source.Credentials == nil && s.Build.Spec.Source.URL != nil {
		switch s.Build.GetAnnotations()[build.AnnotationBuildVerifyRepository] {
		case "true":
			if err := git.ValidateGitURLExists(ctx, *s.Build.Spec.Source.URL); err != nil {
				s.MarkBuildStatus(s.Build, build.RemoteRepositoryUnreachable, err.Error())
				return err
			}

		case "", "false":
			ctxlog.Info(ctx, fmt.Sprintf("the annotation %s is set to %s, nothing to do", build.AnnotationBuildVerifyRepository, s.Build.GetAnnotations()[build.AnnotationBuildVerifyRepository]), namespace, s.Build.Namespace, name, s.Build.Name)

		default:
			var annoErr = fmt.Errorf("the annotation %s was not properly defined, supported values are true or false", build.AnnotationBuildVerifyRepository)
			ctxlog.Error(ctx, annoErr, namespace, s.Build.Namespace, name, s.Build.Name)
			s.MarkBuildStatus(s.Build, build.RemoteRepositoryUnreachable, annoErr.Error())
			return annoErr
		}
	}

	return nil
}

// MarkBuildStatus updates a Build Status fields
func (s SourceURLRef) MarkBuildStatus(b *build.Build, reason build.BuildReason, msg string) {
	b.Status.Reason = build.BuildReasonPtr(reason)
	b.Status.Message = pointer.String(msg)
}
