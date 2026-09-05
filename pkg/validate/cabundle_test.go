// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/cabundle"
	"github.com/shipwright-io/build/pkg/controller/fakes"
	"github.com/shipwright-io/build/pkg/validate"
)

func newCACertPEM() []byte {
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

	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: der,
	})
}

var _ = Describe("CABundle", func() {
	Context("ValidatePath", func() {
		var (
			ctx    context.Context
			c      *fakes.FakeClient
			build  *buildapi.Build
			bundle *validate.CABundle
		)

		BeforeEach(func() {
			ctx = context.Background()
			c = &fakes.FakeClient{}
			build = &buildapi.Build{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-build",
					Namespace: "default",
				},
			}
		})

		It("should successfully validate when no CA bundle is specified", func() {
			bundle = validate.NewCABundle(c, build)
			Expect(bundle.ValidatePath(ctx)).To(BeNil())
			Expect(build.Status.Reason).To(BeNil())
			Expect(build.Status.Message).To(BeNil())
		})

		It("should successfully validate a valid CA bundle from Secret", func() {
			certData := newCACertPEM()
			build.Spec.CABundle = &buildapi.CABundle{
				Secret: &buildapi.SourceObjectKeySelector{
					Name: "my-ca",
					Key:  cabundle.VolumePath,
				},
			}

			c.GetCalls(func(_ context.Context, key types.NamespacedName, obj client.Object, _ ...client.GetOption) error {
				s, ok := obj.(*corev1.Secret)
				if !ok {
					return fmt.Errorf("unexpected object type")
				}
				s.Name = key.Name
				s.Namespace = key.Namespace
				s.Data = map[string][]byte{
					cabundle.VolumePath: certData,
				}
				return nil
			})

			bundle = validate.NewCABundle(c, build)
			Expect(bundle.ValidatePath(ctx)).To(BeNil())
			Expect(build.Status.Reason).To(BeNil())
			Expect(build.Status.Message).To(BeNil())
		})

		It("should set status when CA bundle Secret is not found", func() {
			build.Spec.CABundle = &buildapi.CABundle{
				Secret: &buildapi.SourceObjectKeySelector{
					Name: "missing-ca",
					Key:  cabundle.VolumePath,
				},
			}

			c.GetReturns(apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, "missing-ca"))

			bundle = validate.NewCABundle(c, build)
			Expect(bundle.ValidatePath(ctx)).To(BeNil())
			Expect(build.Status.Reason).ToNot(BeNil())
			Expect(*build.Status.Reason).To(Equal(buildapi.CABundleNotFound))
			Expect(build.Status.Message).ToNot(BeNil())
		})

		It("should set status when CA bundle contains invalid certificate data", func() {
			build.Spec.CABundle = &buildapi.CABundle{
				Secret: &buildapi.SourceObjectKeySelector{
					Name: "bad-ca",
					Key:  cabundle.VolumePath,
				},
			}

			c.GetCalls(func(_ context.Context, key types.NamespacedName, obj client.Object, _ ...client.GetOption) error {
				s, ok := obj.(*corev1.Secret)
				if !ok {
					return fmt.Errorf("unexpected object type")
				}
				s.Name = key.Name
				s.Namespace = key.Namespace
				s.Data = map[string][]byte{
					cabundle.VolumePath: []byte("not-a-certificate"),
				}
				return nil
			})

			bundle = validate.NewCABundle(c, build)
			Expect(bundle.ValidatePath(ctx)).To(BeNil())
			Expect(build.Status.Reason).ToNot(BeNil())
			Expect(*build.Status.Reason).To(Equal(buildapi.CABundleNotValid))
			Expect(build.Status.Message).ToNot(BeNil())
		})

		It("should successfully validate a valid CA bundle from ConfigMap", func() {
			certData := newCACertPEM()
			build.Spec.CABundle = &buildapi.CABundle{
				ConfigMap: &buildapi.SourceObjectKeySelector{
					Name: "my-ca-cm",
					Key:  cabundle.VolumePath,
				},
			}

			c.GetCalls(func(_ context.Context, key types.NamespacedName, obj client.Object, _ ...client.GetOption) error {
				cm, ok := obj.(*corev1.ConfigMap)
				if !ok {
					return fmt.Errorf("unexpected object type")
				}
				cm.Name = key.Name
				cm.Namespace = key.Namespace
				cm.Data = map[string]string{
					cabundle.VolumePath: string(certData),
				}
				return nil
			})

			bundle = validate.NewCABundle(c, build)
			Expect(bundle.ValidatePath(ctx)).To(BeNil())
			Expect(build.Status.Reason).To(BeNil())
			Expect(build.Status.Message).To(BeNil())
		})
	})
})
