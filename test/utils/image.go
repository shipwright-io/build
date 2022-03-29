// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	containerreg "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/gomega"
)

func getImageURL(buildRun *buildv1alpha1.BuildRun) string {
	image := ""
	if buildRun.Spec.Output != nil {
		image = buildRun.Spec.Output.Image
	} else {
		image = buildRun.Status.BuildSpec.Output.Image
	}

	if buildRun.Status.Output != nil && buildRun.Status.Output.Digest != "" {
		image = fmt.Sprintf("%s@%s", image, buildRun.Status.Output.Digest)
	}

	// In the GitHub action, we are using a registry inside the cluster to
	// push the image created by `buildRun`. The registry inside the cluster
	// is not directly accessible from the local, so that we have mapped
	// the cluster registry port to the local system
	// by providing `test/kind/config.yaml` config to the kind
	return strings.Replace(image, "registry.registry.svc.cluster.local", "localhost", 1)
}

// GetImage loads the image manifest for the image produced by a BuildRun
func (t *TestBuild) GetImage(buildRun *buildv1alpha1.BuildRun) containerreg.Image {
	ref, err := name.ParseReference(getImageURL(buildRun))
	Expect(err).ToNot(HaveOccurred())

	img, err := remote.Image(ref, remote.WithAuth(t.getRegistryAuthentication(buildRun, ref)))
	Expect(err).ToNot(HaveOccurred())

	return img
}

func (t *TestBuild) getRegistryAuthentication(
	buildRun *buildv1alpha1.BuildRun,
	ref name.Reference,
) authn.Authenticator {
	secretName := ""
	if buildRun.Spec.Output != nil && buildRun.Spec.Output.Credentials != nil && buildRun.Spec.Output.Credentials.Name != "" {
		secretName = buildRun.Spec.Output.Credentials.Name
	} else if buildRun.Status.BuildSpec.Output.Credentials != nil && buildRun.Status.BuildSpec.Output.Credentials.Name != "" {
		secretName = buildRun.Status.BuildSpec.Output.Credentials.Name
	}

	// In case no secret is mounted, use anonymous
	if secretName == "" {
		log.Println("No access credentials provided, using anonymous mode")
		return authn.Anonymous
	}

	secret, err := t.LookupSecret(
		types.NamespacedName{
			Namespace: buildRun.Namespace,
			Name:      secretName,
		},
	)
	Expect(err).ToNot(HaveOccurred(), "Error retrieving registry secret")

	type auth struct {
		Auths map[string]authn.AuthConfig `json:"auths,omitempty"`
	}

	var authConfig auth

	Expect(json.Unmarshal(secret.Data[".dockerconfigjson"], &authConfig)).ToNot(HaveOccurred(), "Error parsing secrets docker config")

	// Look-up the respective registry server inside the credentials
	registryName := ref.Context().RegistryStr()
	if registryName == name.DefaultRegistry {
		registryName = authn.DefaultAuthKey
	}

	return authn.FromConfig(authConfig.Auths[registryName])
}

// ValidateImagePlatformsExist that the image produced by a BuildRun exists for a set of platforms
func (t *TestBuild) ValidateImagePlatformsExist(buildRun *buildv1alpha1.BuildRun, expectedPlatforms []containerreg.Platform) {
	ref, err := name.ParseReference(getImageURL(buildRun))
	Expect(err).ToNot(HaveOccurred())

	for _, expectedPlatform := range expectedPlatforms {
		_, err := remote.Image(ref, remote.WithAuth(t.getRegistryAuthentication(buildRun, ref)), remote.WithPlatform(expectedPlatform))
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to validate %s/%s", expectedPlatform.OS, expectedPlatform.Architecture))
	}
}

// ValidateImageDigest ensures that an image digest is set in the BuildRun status and that this digest is pointing to an image
func (t *TestBuild) ValidateImageDigest(buildRun *buildv1alpha1.BuildRun) {
	// Verify that the status contains a digest
	Expect(buildRun.Status.Output).NotTo(BeNil(), ".status.output is nil")
	Expect(buildRun.Status.Output.Digest).NotTo(Equal(""), ".status.output.digest is empty")

	// Verify that the digest is valid by retrieving the image manifest
	t.GetImage(buildRun)
}
