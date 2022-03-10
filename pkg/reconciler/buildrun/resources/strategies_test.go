// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/controller/fakes"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
	"github.com/shipwright-io/build/test"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	crc "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Operating Build strategies", func() {
	var (
		client *fakes.FakeClient
		ctl    test.Catalog
	)

	Context("Retrieving build strategies", func() {
		client = &fakes.FakeClient{}
		buildSample := ctl.BuildWithBuildStrategy("foobuild", "foostrategy", "foostrategy")

		It("should return a cluster buildstrategy", func() {			
			// stub a GET API call with a cluster strategy
			getClientStub := func(_ context.Context, nn types.NamespacedName, object crc.Object) error {
				switch object := object.(type) {
				case *buildv1alpha1.ClusterBuildStrategy:
					ctl.DefaultClusterBuildStrategy().DeepCopyInto(object)
					return nil
				}
				return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
			}
			// fake the calls with the above stub
			client.GetCalls(getClientStub)

			cbs, err := resources.RetrieveClusterBuildStrategy(context.TODO(), client, buildSample)
			Expect(err).To(BeNil())
			Expect(cbs.Name).To(Equal("foobar"))
		})

		It("should return a namespaced buildstrategy", func() {			
			// stub a GET API call with a namespace strategy
			getClientStub := func(_ context.Context, nn types.NamespacedName, object crc.Object) error {
				switch object := object.(type) {
				case *buildv1alpha1.BuildStrategy:
					ctl.DefaultNamespacedBuildStrategy().DeepCopyInto(object)
					return nil
				}
				return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
			}
			// fake the calls with the above stub
			client.GetCalls(getClientStub)

			cbs, err := resources.RetrieveBuildStrategy(context.TODO(), client, buildSample)
			Expect(err).To(BeNil())
			Expect(cbs.Name).To(Equal("foobar"))
		})

	})
})
