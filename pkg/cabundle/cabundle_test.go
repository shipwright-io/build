// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package cabundle_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"fmt"
	"math/big"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/cabundle"
	"github.com/shipwright-io/build/pkg/controller/fakes"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newCACertDER() []byte {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	Expect(err).ToNot(HaveOccurred())

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	Expect(err).ToNot(HaveOccurred())
	return der
}

func newNonCACertDER() []byte {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	Expect(err).ToNot(HaveOccurred())

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  false,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	Expect(err).ToNot(HaveOccurred())
	return der
}

var _ = Describe("CABundle", func() {

	Describe("Validate", func() {
		var (
			ctx       context.Context
			namespace string
			c         *fakes.FakeClient
		)

		BeforeEach(func() {
			ctx = context.Background()
			namespace = "default"
			c = &fakes.FakeClient{}
		})

		Context("with a Secret reference", func() {
			It("should succeed when the secret exists and contains a valid CA certificate", func() {
				certData := newCACertDER()
				c.GetCalls(func(_ context.Context, key types.NamespacedName, obj client.Object, _ ...client.GetOption) error {
					s, ok := obj.(*corev1.Secret)
					if !ok {
						return fmt.Errorf("unexpected object type")
					}
					s.Name = key.Name
					s.Namespace = key.Namespace
					s.Data = map[string][]byte{
						cabundle.Key: certData,
					}
					return nil
				})

				ca := &buildv1beta1.CABundle{
					Secret: &buildv1beta1.SourceObjectKeySelector{
						Name: "my-ca",
						Key:  cabundle.Key,
					},
				}
				Expect(cabundle.Validate(ctx, c, ca, namespace)).To(Succeed())
			})

			It("should fail when the secret does not exist", func() {
				c.GetReturns(apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, "missing-secret"))

				ca := &buildv1beta1.CABundle{
					Secret: &buildv1beta1.SourceObjectKeySelector{
						Name: "missing-secret",
						Key:  cabundle.Key,
					},
				}
				err := cabundle.Validate(ctx, c, ca, namespace)
				Expect(err).To(HaveOccurred())
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			})

			It("should fail when the secret contains invalid certificate data", func() {
				c.GetCalls(func(_ context.Context, key types.NamespacedName, obj client.Object, _ ...client.GetOption) error {
					s, ok := obj.(*corev1.Secret)
					if !ok {
						return fmt.Errorf("unexpected object type")
					}
					s.Name = key.Name
					s.Namespace = key.Namespace
					s.Data = map[string][]byte{
						cabundle.Key: []byte("not-a-certificate"),
					}
					return nil
				})

				ca := &buildv1beta1.CABundle{
					Secret: &buildv1beta1.SourceObjectKeySelector{
						Name: "bad-cert",
						Key:  cabundle.Key,
					},
				}
				Expect(cabundle.Validate(ctx, c, ca, namespace)).ToNot(Succeed())
			})

			It("should fail when the certificate is not a CA", func() {
				certData := newNonCACertDER()
				c.GetCalls(func(_ context.Context, key types.NamespacedName, obj client.Object, _ ...client.GetOption) error {
					s, ok := obj.(*corev1.Secret)
					if !ok {
						return fmt.Errorf("unexpected object type")
					}
					s.Name = key.Name
					s.Namespace = key.Namespace
					s.Data = map[string][]byte{
						cabundle.Key: certData,
					}
					return nil
				})

				ca := &buildv1beta1.CABundle{
					Secret: &buildv1beta1.SourceObjectKeySelector{
						Name: "non-ca",
						Key:  cabundle.Key,
					},
				}
				err := cabundle.Validate(ctx, c, ca, namespace)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("certificate is not a CA"))
			})
		})

		Context("with a ConfigMap reference", func() {
			It("should succeed when the configmap exists and contains a valid CA certificate", func() {
				certData := newCACertDER()
				c.GetCalls(func(_ context.Context, key types.NamespacedName, obj client.Object, _ ...client.GetOption) error {
					cm, ok := obj.(*corev1.ConfigMap)
					if !ok {
						return fmt.Errorf("unexpected object type")
					}
					cm.Name = key.Name
					cm.Namespace = key.Namespace
					cm.Data = map[string]string{
						cabundle.Key: string(certData),
					}
					return nil
				})

				ca := &buildv1beta1.CABundle{
					ConfigMap: &buildv1beta1.SourceObjectKeySelector{
						Name: "my-ca-cm",
						Key:  cabundle.Key,
					},
				}
				Expect(cabundle.Validate(ctx, c, ca, namespace)).To(Succeed())
			})

			It("should fail when the configmap does not exist", func() {
				c.GetReturns(apierrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, "missing-cm"))

				ca := &buildv1beta1.CABundle{
					ConfigMap: &buildv1beta1.SourceObjectKeySelector{
						Name: "missing-cm",
						Key:  cabundle.Key,
					},
				}
				err := cabundle.Validate(ctx, c, ca, namespace)
				Expect(err).To(HaveOccurred())
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			})

			It("should fail when the configmap contains invalid certificate data", func() {
				c.GetCalls(func(_ context.Context, key types.NamespacedName, obj client.Object, _ ...client.GetOption) error {
					cm, ok := obj.(*corev1.ConfigMap)
					if !ok {
						return fmt.Errorf("unexpected object type")
					}
					cm.Name = key.Name
					cm.Namespace = key.Namespace
					cm.Data = map[string]string{
						cabundle.Key: "not-a-certificate",
					}
					return nil
				})

				ca := &buildv1beta1.CABundle{
					ConfigMap: &buildv1beta1.SourceObjectKeySelector{
						Name: "bad-cert-cm",
						Key:  cabundle.Key,
					},
				}
				Expect(cabundle.Validate(ctx, c, ca, namespace)).ToNot(Succeed())
			})

			It("should fail when the certificate is not a CA", func() {
				certData := newNonCACertDER()
				c.GetCalls(func(_ context.Context, key types.NamespacedName, obj client.Object, _ ...client.GetOption) error {
					cm, ok := obj.(*corev1.ConfigMap)
					if !ok {
						return fmt.Errorf("unexpected object type")
					}
					cm.Name = key.Name
					cm.Namespace = key.Namespace
					cm.Data = map[string]string{
						cabundle.Key: string(certData),
					}
					return nil
				})

				ca := &buildv1beta1.CABundle{
					ConfigMap: &buildv1beta1.SourceObjectKeySelector{
						Name: "non-ca-cm",
						Key:  cabundle.Key,
					},
				}
				err := cabundle.Validate(ctx, c, ca, namespace)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("certificate is not a CA"))
			})
		})
	})

	Describe("NewVolume", func() {
		Context("with a Secret reference", func() {
			It("should create a volume with a SecretVolumeSource", func() {
				ca := &buildv1beta1.CABundle{
					Secret: &buildv1beta1.SourceObjectKeySelector{
						Name: "my-secret",
						Key:  "tls.crt",
					},
				}
				vol := cabundle.NewVolume(ca)

				Expect(vol).ToNot(BeNil())
				Expect(vol.Name).To(ContainSubstring("my-secret"))
				Expect(vol.Secret).ToNot(BeNil())
				Expect(vol.Secret.SecretName).To(Equal("my-secret"))
				Expect(vol.Secret.DefaultMode).ToNot(BeNil())
				Expect(*vol.Secret.DefaultMode).To(Equal(int32(0444)))
				Expect(vol.Secret.Items).To(HaveLen(1))
				Expect(vol.Secret.Items[0].Key).To(Equal("tls.crt"))
				Expect(vol.Secret.Items[0].Path).To(Equal(cabundle.Key))
			})
		})

		Context("with a ConfigMap reference", func() {
			It("should create a volume with a ConfigMapVolumeSource", func() {
				ca := &buildv1beta1.CABundle{
					ConfigMap: &buildv1beta1.SourceObjectKeySelector{
						Name: "my-configmap",
						Key:  "ca.pem",
					},
				}
				vol := cabundle.NewVolume(ca)

				Expect(vol).ToNot(BeNil())
				Expect(vol.ConfigMap).ToNot(BeNil())
				Expect(vol.ConfigMap.Name).To(Equal("my-configmap"))
				Expect(vol.ConfigMap.DefaultMode).ToNot(BeNil())
				Expect(*vol.ConfigMap.DefaultMode).To(Equal(int32(0444)))
				Expect(vol.ConfigMap.Items).To(HaveLen(1))
				Expect(vol.ConfigMap.Items[0].Key).To(Equal("ca.pem"))
				Expect(vol.ConfigMap.Items[0].Path).To(Equal(cabundle.Key))
			})
		})

		It("should produce a deterministic hashed volume name", func() {
			ca := &buildv1beta1.CABundle{
				Secret: &buildv1beta1.SourceObjectKeySelector{
					Name: "my-secret",
					Key:  cabundle.Key,
				},
			}
			vol1 := cabundle.NewVolume(ca)
			vol2 := cabundle.NewVolume(ca)

			Expect(vol1.Name).To(Equal(vol2.Name))
		})
	})

	Describe("NewVolumeMount", func() {
		It("should create a volume mount for each bundle path", func() {
			vol := &corev1.Volume{
				Name: "test-volume",
			}
			mounts := cabundle.NewVolumeMount(vol)

			Expect(mounts).To(HaveLen(len(cabundle.Bundles)))
			for i, mount := range mounts {
				Expect(mount.Name).To(Equal("test-volume"))
				Expect(mount.MountPath).To(Equal(cabundle.Bundles[i]))
				Expect(mount.SubPath).To(Equal(cabundle.Key))
				Expect(mount.ReadOnly).To(BeTrue())
			}
		})
	})

	Describe("NewEnvVar", func() {
		It("should create an env var for each entry in EnvVars", func() {
			envVars := cabundle.NewEnvVar()

			Expect(envVars).To(HaveLen(len(cabundle.EnvVars)))
			for i, ev := range envVars {
				Expect(ev.Name).To(Equal(cabundle.EnvVars[i]))
				Expect(ev.Value).To(Equal(cabundle.File))
			}
		})
	})
})
