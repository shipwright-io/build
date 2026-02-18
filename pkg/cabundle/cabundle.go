// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package cabundle

import (
	"context"
	"crypto/x509"
	"fmt"
	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"hash/fnv"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const Key = "ca.crt"
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
	var o client.Object
	switch {
	case ca.Secret != nil:
		o = &corev1.Secret{}
		o.SetName(ca.Secret.Name)
	case ca.ConfigMap != nil:
		o = &corev1.ConfigMap{}
		o.SetName(ca.ConfigMap.Name)
	}
	o.SetNamespace(namespace)

	// Validations
	if err := checkResourceExists(ctx, c, o); err != nil {
		return err
	}
	if err := checkValidCABundle(o); err != nil {
		return err
	}

	return nil
}

func checkResourceExists(ctx context.Context, client client.Client, object client.Object) error {
	return client.Get(ctx, types.NamespacedName{
		Name:      object.GetName(),
		Namespace: object.GetNamespace(),
	}, object)
}

func checkValidCABundle(object client.Object) (err error) {
	var data []byte
	switch o := object.(type) {
	case *corev1.Secret:
		data = o.Data[Key]
	case *corev1.ConfigMap:
		data = []byte(o.Data[Key])
	}
	c, err := x509.ParseCertificate(data)
	if err != nil {
		return
	}
	if !c.IsCA {
		return fmt.Errorf("certificate is not a CA")
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
				{Key: ca.Secret.Key, Path: Key},
			},
		}
	case ca.ConfigMap != nil:
		v.ConfigMap = &corev1.ConfigMapVolumeSource{
			LocalObjectReference: corev1.LocalObjectReference{Name: ca.ConfigMap.Name},
			DefaultMode:          ptr.To[int32](0444),
			Items: []corev1.KeyToPath{
				{Key: ca.ConfigMap.Key, Path: Key},
			},
		}
	}
	v.Name = getHashedName(name)
	return v
}

func getHashedName(name string) string {
	hasher := fnv.New32a()
	hasher.Write([]byte(name))
	return fmt.Sprintf("%s-%s", name, rand.SafeEncodeString(fmt.Sprint(hasher.Sum32())))
}

func NewVolumeMount(volume *corev1.Volume) []corev1.VolumeMount {
	var vm []corev1.VolumeMount
	for _, bundle := range Bundles {
		vm = append(vm, corev1.VolumeMount{
			Name:      volume.Name,
			MountPath: bundle,
			SubPath:   Key,
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
