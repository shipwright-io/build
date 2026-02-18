// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package cabundle

import (
	"context"
	"crypto/x509"
	"fmt"
	"hash/fnv"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const Path = "ca.crt"
const File = "/etc/ssl/certs/ca-certificates.crt"

// Bundles contains default cert directories for linux distributions to search. Order must be maintained.
// https://go.dev/src/crypto/x509/root_linux.go
var Bundles = []string{
	"/etc/ssl/certs/ca-certificates.crt", // Debian/Ubuntu/Gentoo
	"/etc/pki/tls/certs/ca-bundle.crt",   // Fedora/RHEL
}

var EnvVars = []string{
	"SSL_CERT_FILE",       // OS trust store
	"NODE_EXTRA_CA_CERTS", // Node.js uses this to append additional certificates
	"REQUESTS_CA_BUNDLE",  // Python requests library
	"CURL_CA_BUNDLE",      // curl CA bundle
}

func Validate(ctx context.Context, c client.Client, ca *buildv1beta1.CABundle, namespace string) error {
	var object client.Object
	var name string
	var data []byte

	if ca == nil {
		return fmt.Errorf("no certificate provided")
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
	certificate, err := x509.ParseCertificate(data)
	if err != nil {
		return err
	}
	if !certificate.IsCA {
		return fmt.Errorf("invalid certificate data, not CA")
	}

	return nil
}

func NewVolume(ca *buildv1beta1.CABundle) *corev1.Volume {
	var name string
	v := &corev1.Volume{}
	switch {
	case ca.Secret != nil:
		name = ca.Secret.Name
		v.Secret = &corev1.SecretVolumeSource{
			SecretName:  ca.Secret.Name,
			DefaultMode: ptr.To[int32](0444),
			Items: []corev1.KeyToPath{
				{Key: ca.Secret.Key, Path: Path},
			},
		}
	case ca.ConfigMap != nil:
		name = ca.ConfigMap.Name
		v.ConfigMap = &corev1.ConfigMapVolumeSource{
			LocalObjectReference: corev1.LocalObjectReference{Name: ca.ConfigMap.Name},
			DefaultMode:          ptr.To[int32](0444),
			Items: []corev1.KeyToPath{
				{Key: ca.ConfigMap.Key, Path: Path},
			},
		}
	}
	v.Name = getHashedName(name)
	return v
}

func getHashedName(name string) string {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(name))
	return fmt.Sprintf("%s-%s", name, rand.SafeEncodeString(fmt.Sprint(hasher.Sum32())))
}

func NewVolumeMount(volume *corev1.Volume) []corev1.VolumeMount {
	var vm []corev1.VolumeMount
	for _, bundle := range Bundles {
		vm = append(vm, corev1.VolumeMount{
			Name:      volume.Name,
			MountPath: bundle,
			SubPath:   Path,
			ReadOnly:  true,
		})
	}
	return vm
}

func NewEnvVar() []corev1.EnvVar {
	var ev []corev1.EnvVar
	for _, envVar := range EnvVars {
		ev = append(ev, corev1.EnvVar{
			Name:  envVar,
			Value: File,
		})
	}
	return ev
}
