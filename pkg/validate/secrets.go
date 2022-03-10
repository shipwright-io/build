// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"fmt"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

// Credentials contains all required fields
// to validate a Build spec secrets definitions
type Credentials struct {
	Build  *build.Build
	Client client.Client
}

func NewCredentials(client client.Client, build *build.Build) *Credentials {
	return &Credentials{build, client}
}

// ValidatePath implements BuildPath interface and validates
// that all referenced secrets under spec exists
func (s Credentials) ValidatePath(ctx context.Context) error {
	var missingSecrets []string
	secret := &corev1.Secret{}

	secretNames := s.buildCredentialserences()

	for refSecret, secretType := range secretNames {
		if err := s.Client.Get(ctx, types.NamespacedName{Name: refSecret, Namespace: s.Build.Namespace}, secret); err != nil && !apierrors.IsNotFound(err) {
			return err
		} else if apierrors.IsNotFound(err) {
			s.Build.Status.Reason = build.BuildReasonPtr(secretType)
			s.Build.Status.Message = pointer.String(fmt.Sprintf("referenced secret %s not found", refSecret))
			missingSecrets = append(missingSecrets, refSecret)
		}
	}

	// sorts a list of secret names in increasing order
	sort.Strings(missingSecrets)

	if len(missingSecrets) > 1 {
		s.Build.Status.Reason = build.BuildReasonPtr(build.MultipleSecretRefNotFound)
		s.Build.Status.Message = pointer.String(fmt.Sprintf("missing secrets are %s", strings.Join(missingSecrets, ",")))
	}
	return nil
}

func (s Credentials) buildCredentialserences() map[string]build.BuildReason {
	// Validate if the referenced secrets exist in the namespace
	secretRefMap := map[string]build.BuildReason{}
	if s.Build.Spec.Output.Credentials != nil && s.Build.Spec.Output.Credentials.Name != "" {
		secretRefMap[s.Build.Spec.Output.Credentials.Name] = build.SpecOutputSecretRefNotFound
	}
	if s.Build.Spec.Source.Credentials != nil && s.Build.Spec.Source.Credentials.Name != "" {
		secretRefMap[s.Build.Spec.Source.Credentials.Name] = build.SpecSourceSecretRefNotFound
	}
	if s.Build.Spec.Builder != nil && s.Build.Spec.Builder.Credentials != nil && s.Build.Spec.Builder.Credentials.Name != "" {
		secretRefMap[s.Build.Spec.Builder.Credentials.Name] = build.SpecBuilderSecretRefNotFound
	}
	return secretRefMap
}
