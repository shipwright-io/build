package e2e

import (
	"os"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
)

// regularTestCases contains all data in samples, the strategy shipped by the operator are verified
// during CI.
var regularTestCases = map[string]*SampleFiles{
	"kaniko": {
		ClusterBuildStrategy: "samples/buildstrategy/kaniko/buildstrategy_kaniko_cr.yaml",
		Build:                "samples/build/build_kaniko_cr.yaml",
		BuildRun:             "samples/buildrun/buildrun_kaniko_cr.yaml",
	},
	"kaniko-custom-context-dockerfile": {
		ClusterBuildStrategy: "samples/buildstrategy/kaniko/buildstrategy_kaniko_cr.yaml",
		Build:                "test/data/build_kaniko_cr_custom_context+dockerfile.yaml",
		BuildRun:             "test/data/buildrun_kaniko_cr_custom_context+dockerfile.yaml",
	},
	"s2i": {
		ClusterBuildStrategy: "samples/buildstrategy/source-to-image/buildstrategy_source-to-image_cr.yaml",
		Build:                "samples/build/build_source-to-image_cr.yaml",
		BuildRun:             "samples/buildrun/buildrun_source-to-image_cr.yaml",
	},
	"buildah": {
		ClusterBuildStrategy: "samples/buildstrategy/buildah/buildstrategy_buildah_cr.yaml",
		Build:                "samples/build/build_buildah_cr.yaml",
		BuildRun:             "samples/buildrun/buildrun_buildah_cr.yaml",
	},
	"buildah-custom-context-dockerfile": {
		ClusterBuildStrategy: "samples/buildstrategy/buildah/buildstrategy_buildah_cr.yaml",
		Build:                "test/data/build_buildah_cr_custom_context+dockerfile.yaml",
		BuildRun:             "test/data/buildrun_buildah_cr_custom_context+dockerfile.yaml",
	},
	"buildpacks-v3": {
		ClusterBuildStrategy: "samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_cr.yaml",
		Build:                "samples/build/build_buildpacks-v3_cr.yaml",
		BuildRun:             "samples/buildrun/buildrun_buildpacks-v3_cr.yaml",
	},
	"buildpacks-v3-namespaced": {
		BuildStrategy: "samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_namespaced_cr.yaml",
		Build:         "samples/build/build_buildpacks-v3_namespaced_cr.yaml",
		BuildRun:      "samples/buildrun/buildrun_buildpacks-v3_namespaced_cr.yaml",
	},
}

// privateTestCases contains all test cases design to run against private git repositories, those
// tests are enabled via environment variable ("TEST_PRIVATE_REPO").
var privateTestCases = map[string]*SampleFiles{
	"private-github-kaniko": {
		ClusterBuildStrategy: "samples/buildstrategy/kaniko/buildstrategy_kaniko_cr.yaml",
		Build:                "test/data/build_kaniko_cr_private_github.yaml",
		BuildRun:             "samples/buildrun/buildrun_kaniko_cr.yaml",
	},
	"private-gitlab-kaniko": {
		ClusterBuildStrategy: "samples/buildstrategy/kaniko/buildstrategy_kaniko_cr.yaml",
		Build:                "test/data/build_kaniko_cr_private_gitlab.yaml",
		BuildRun:             "samples/buildrun/buildrun_kaniko_cr.yaml",
	},
	"private-github-buildah": {
		ClusterBuildStrategy: "samples/buildstrategy/buildah/buildstrategy_buildah_cr.yaml",
		Build:                "test/data/build_buildah_cr_private_github.yaml",
		BuildRun:             "samples/buildrun/buildrun_buildah_cr.yaml",
	},
	"private-gitlab-buildah": {
		ClusterBuildStrategy: "samples/buildstrategy/buildah/buildstrategy_buildah_cr.yaml",
		Build:                "test/data/build_buildah_cr_private_gitlab.yaml",
		BuildRun:             "samples/buildrun/buildrun_buildah_cr.yaml",
	},
	"private-github-buildpacks-v3": {
		ClusterBuildStrategy: "samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_cr.yaml",
		Build:                "test/data/build_buildpacks-v3_cr_private_github.yaml",
		BuildRun:             "samples/buildrun/buildrun_buildpacks-v3_cr.yaml",
	},
	"private-github-s2i": {
		ClusterBuildStrategy: "samples/buildstrategy/source-to-image/buildstrategy_source-to-image_cr.yaml",
		Build:                "test/data/build_source-to-image_cr_private_github.yaml",
		BuildRun:             "samples/buildrun/buildrun_source-to-image_cr.yaml",
	},
}

// OperatorTests execute test cases.
func OperatorTests(t *testing.T, ctx *framework.TestCtx, f *framework.Framework) {
	createContainerRegistrySecret(t, ctx, f)

	samplesTesting := NewSamplesTesting(t, ctx, f)
	samplesTesting.TestAll(regularTestCases)
	if os.Getenv(EnvVarEnablePrivateRepos) == "true" {
		samplesTesting.TestAll(privateTestCases)
	}
}
