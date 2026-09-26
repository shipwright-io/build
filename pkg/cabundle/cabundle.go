// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package cabundle

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"hash/fnv"
	"regexp"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

const VolumePath = "ca.crt"
const CACertFile = "/etc/ssl/certs/ca-certificates.crt"

var dnsLabel1123Forbidden = regexp.MustCompile("[^a-zA-Z0-9-]+")

// DefaultBundlePaths contains default cert directories for linux distributions to search. Order must be maintained.
// https://go.dev/src/crypto/x509/root_linux.go
var DefaultBundlePaths = []string{
	"/etc/ssl/certs/ca-certificates.crt", // Debian/Ubuntu/Gentoo
	"/etc/pki/tls/certs/ca-bundle.crt",   // Fedora/RHEL
}

var EnvVars = []string{
	"SSL_CERT_FILE",       // OS trust store
	"NODE_EXTRA_CA_CERTS", // Node.js uses this to append additional certificates
	"REQUESTS_CA_BUNDLE",  // Python requests library
	"CURL_CA_BUNDLE",      // curl CA bundle
}

func Validate(ctx context.Context, c client.Client, ca *buildapi.CABundle, namespace string) error {
	var object client.Object
	var name string
	var data []byte

	if ca == nil {
		return fmt.Errorf("no CA bundle provided")
	}

	switch {
	case ca.Secret != nil:
		object = &corev1.Secret{}
		name = ca.Secret.Name
	case ca.ConfigMap != nil:
		object = &corev1.ConfigMap{}
		name = ca.ConfigMap.Name
	}

	// Check if resource exists
	if err := c.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, object); err != nil {
		return err
	}

	// Check if CA Bundle data is valid
	switch o := object.(type) {
	case *corev1.Secret:
		data = o.Data[ca.Secret.Key]
	case *corev1.ConfigMap:
		data = []byte(o.Data[ca.ConfigMap.Key])
	}

	// Parse and validate each certificate in the PEM data
	block, rest := pem.Decode(data)
	if block == nil {
		return fmt.Errorf("no certificate present in CA bundle")
	}

	for block != nil {
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return fmt.Errorf("unable to parse x509 certificate data: %v", err)
		}
		if !cert.IsCA {
			return fmt.Errorf("invalid certificate data, not CA")
		}
		block, rest = pem.Decode(rest)
	}

	return nil
}

func NewVolume(ca *buildapi.CABundle) *corev1.Volume {
	var name string
	v := &corev1.Volume{}
	switch {
	case ca.Secret != nil:
		name = ca.Secret.Name
		v.Secret = &corev1.SecretVolumeSource{
			SecretName: ca.Secret.Name,
			Items: []corev1.KeyToPath{
				{Key: ca.Secret.Key, Path: VolumePath},
			},
		}
	case ca.ConfigMap != nil:
		name = ca.ConfigMap.Name
		v.ConfigMap = &corev1.ConfigMapVolumeSource{
			LocalObjectReference: corev1.LocalObjectReference{Name: ca.ConfigMap.Name},
			Items: []corev1.KeyToPath{
				{Key: ca.ConfigMap.Key, Path: VolumePath},
			},
		}
	}
	v.Name = getHashedName(name)
	return v
}

func getHashedName(name string) string {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(name))
	hash := strconv.FormatUint(uint64(hasher.Sum32()), 10)

	// Convert to lowercase and remove forbidden characters
	sanitizedName := strings.ToLower(dnsLabel1123Forbidden.ReplaceAllString(name, "-"))

	// Remove both leading and trailing hyphens
	sanitizedName = strings.Trim(sanitizedName, "-")

	// Ensure maximum length, leaving space for the hash
	maxLength := 63 - len(hash) - 1 // -1 for the hyphen separator
	if len(sanitizedName) > maxLength {
		sanitizedName = sanitizedName[:maxLength]
	}

	return fmt.Sprintf("%s-%s", sanitizedName, hash)
}

func NewVolumeMount(volume *corev1.Volume) []corev1.VolumeMount {
	var vm []corev1.VolumeMount
	for _, bundle := range DefaultBundlePaths {
		vm = append(vm, corev1.VolumeMount{
			Name:              volume.Name,
			MountPath:         bundle,
			SubPath:           VolumePath,
			ReadOnly:          true,
			RecursiveReadOnly: ptr.To(corev1.RecursiveReadOnlyIfPossible),
		})
	}
	return vm
}

func NewEnvVar() []corev1.EnvVar {
	var ev []corev1.EnvVar
	for _, envVar := range EnvVars {
		ev = append(ev, corev1.EnvVar{
			Name:  envVar,
			Value: CACertFile,
		})
	}
	return ev
}
