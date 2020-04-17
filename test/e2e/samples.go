package e2e

import (
	"fmt"
	"os"
	"strings"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	operator "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
)

// SamplesTesting encapsulate the probes executed against sample files in build-operator.
type SamplesTesting struct {
	t   *testing.T           // testing instance
	ctx *framework.TestCtx   // operator-sdk test context
	f   *framework.Framework // operator-sdk test framework parameters
	ns  string               // namespace
}

// SampleFiles group resources employed during each test interaction
type SampleFiles struct {
	ClusterBuildStrategy string // ClusterBuildStrategy file path
	BuildStrategy        string // BuildStrategy file path
	Build                string // Build file path
	BuildRun             string // BuildRun  file path
}

// amendOutputImageURL amend container image URL based on informed image repository.
func (s *SamplesTesting) amendOuputImageURL(b *operator.Build, imageRepo string) {
	if imageRepo == "" {
		return
	}
	imageURL := fmt.Sprintf("%s:%s", imageRepo, b.Name)
	b.Spec.Output.ImageURL = imageURL
	s.t.Logf("Amended object: name='%s', image-url='%s'", b.Name, imageURL)
}

// amendOutputSecretRef amend secret-ref for output image.
func (s *SamplesTesting) amendOutputSecretRef(b *operator.Build, secretName string) {
	if secretName == "" {
		return
	}
	b.Spec.Output.SecretRef = &v1.LocalObjectReference{Name: secretName}
	s.t.Logf("Amended object: name='%s', secret-ref='%s'", b.Name, secretName)
}

// amendSourceSecretName patch Build source.SecretRef with secret name.
func (s *SamplesTesting) amendSourceSecretName(b *operator.Build, secretName string) {
	if secretName == "" {
		return
	}
	b.Spec.Source.SecretRef = &v1.LocalObjectReference{Name: secretName}
}

// amendSourceURL patch Build source.URL with informed string.
func (s *SamplesTesting) amendSourceURL(b *operator.Build, sourceURL string) {
	if sourceURL == "" {
		return
	}
	b.Spec.Source.URL = sourceURL
}

// amendBuild make changes on build object.
func (s *SamplesTesting) amendBuild(identifier string, b *operator.Build) {
	s.amendSourceSecretName(b, os.Getenv(EnvVarSourceURLSecret))
	if strings.Contains(identifier, "github") {
		s.amendSourceURL(b, os.Getenv(EnvVarSourceURLGithub))
	} else if strings.Contains(identifier, "gitlab") {
		s.amendSourceURL(b, os.Getenv(EnvVarSourceURLGitlab))
	}

	s.amendOuputImageURL(b, os.Getenv(EnvVarImageRepo))
	s.amendOutputSecretRef(b, os.Getenv(EnvVarImageRepoSecret))
}

// Test execute the test against the informed resources.
func (s *SamplesTesting) Test(identifier string, files *SampleFiles) {
	s.t.Logf("Testing '%s' using files '%#v'", identifier, files)

	b, err := buildTestData(s.ns, identifier, files.Build)
	require.NoError(s.t, err)
	s.amendBuild(identifier, b)

	br, err := buildRunTestData(s.ns, identifier, files.BuildRun)
	require.NoError(s.t, err)

	// on using an empty ClusterBuildStrategy it will fall back on namespaced BuildStrategy
	if files.ClusterBuildStrategy != "" {
		s.t.Log("Using a cluster wide build-strategy...")

		cbs, err := clusterBuildStrategyTestData(files.ClusterBuildStrategy)
		require.NoError(s.t, err)
		cbs.SetNamespace(s.ns)

		createClusterBuildStrategy(s.t, s.ctx, s.f, cbs)
	} else {
		s.t.Log("Using a namespaced build-strategy...")

		bs, err := buildStrategyTestData(s.ns, files.BuildStrategy)
		require.NoError(s.t, err)
		bs.SetNamespace(s.ns)

		createNamespacedBuildStrategy(s.t, s.ctx, s.f, bs)
	}

	validateController(s.t, s.ctx, s.f, b, br)
}

// TestAll iterate through test cases.
func (s *SamplesTesting) TestAll(cases map[string]*SampleFiles) {
	for identifier, files := range cases {
		s.t.Logf("Running tests: '%s'", identifier)
		s.Test(identifier, files)
	}
}

// NewSamplesTesting instantiate SamplesTesting sharing testing components.
func NewSamplesTesting(
	t *testing.T,
	ctx *framework.TestCtx,
	f *framework.Framework,
) *SamplesTesting {
	ns, err := ctx.GetNamespace()
	require.NoError(t, err)
	return &SamplesTesting{t: t, ctx: ctx, f: f, ns: ns}
}
