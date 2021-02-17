// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/shipwright-io/build/pkg/controller/fakes"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
	"github.com/shipwright-io/build/test"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Operating service accounts", func() {
	var (
		client                  *fakes.FakeClient
		ctl                     test.Catalog
		buildName, buildRunName string
	)

	Context("Retrieving service accounts", func() {

		// init vars
		buildName = "foobuild"
		buildRunName = "foobuildrun"
		client = &fakes.FakeClient{}
		buildRunSample := ctl.DefaultBuildRun(buildRunName, buildName)

		It("should return a modified one with a secret ref", func() {

			// stub a GET API call for a service account
			getClientStub := func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
				switch object := object.(type) {
				case *corev1.ServiceAccount:
					ctl.DefaultServiceAccount("foobar").DeepCopyInto(object)
					return nil
				}
				return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
			}
			// fake the calls with the above stub
			client.GetCalls(getClientStub)

			sa, err := resources.RetrieveServiceAccount(context.TODO(), client, ctl.BuildWithOutputSecret(buildName, "default", "foosecret"), buildRunSample)
			Expect(err).To(BeNil())
			// assert the build output secret is defined in the default SA
			Expect(len(sa.Secrets)).To(Equal(1))
		})

		It("should provide a generated sa name", func() {
			Expect(resources.GetGeneratedServiceAccountName(buildRunSample)).To(Equal(buildRunSample.Name + "-sa"))
		})
	})
})
