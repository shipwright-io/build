// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/controller/fakes"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
	"github.com/shipwright-io/build/test"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Build Resource", func() {

	var (
		client    *fakes.FakeClient
		ctl       test.Catalog
		buildName string
	)

	Context("Operating on Build resources", func() {
		// init vars
		buildName = "foobuild"
		client = &fakes.FakeClient{}

		It("should be able to retrieve a build object if exists", func() {
			buildSample := ctl.DefaultBuild(buildName, "foostrategy", build.ClusterBuildStrategyKind)

			// stub a GET API call with buildSample contents
			getClientStub := func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
				switch object := object.(type) {
				case *build.Build:
					buildSample.DeepCopyInto(object)
					return nil
				}
				return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
			}

			// fake the calls with the above stub definition
			client.GetCalls(getClientStub)

			build := &build.Build{}
			Expect(resources.GetBuildObject(context.TODO(), client, buildName, "default", build)).To(BeNil())
		})
		It("should not retrieve a missing build object when missing", func() {
			// stub a GET API call with buildSample contents that returns "not found"
			getClientStub := func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
				switch object.(type) {
				case *build.Build:
					return errors.New("not found")
				}
				return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
			}
			// fake the calls with the above stub
			client.GetCalls(getClientStub)

			build := &build.Build{}
			Expect(resources.GetBuildObject(context.TODO(), client, buildName, "default", build)).ToNot(BeNil())
		})
		It("should be able to verify valid ownerships", func() {
			managingController := true

			buildSample := &build.Build{
				TypeMeta: metav1.TypeMeta{
					Kind: "fakekind",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: buildName,
				},
			}
			// fake an instance of OwnerReference with a
			// well known fake Kind and Name
			fakeOwnerRef := []metav1.OwnerReference{
				{
					Kind:       "fakekind",
					Name:       buildName,
					Controller: &managingController,
				},
			}
			// Assert that our Build is owned by an owner
			Expect(resources.IsOwnedByBuild(buildSample, fakeOwnerRef)).To(BeTrue())
		})
		It("should be able to verify invalid ownerships", func() {
			managingController := true

			buildSample := &build.Build{
				TypeMeta: metav1.TypeMeta{
					Kind: "notthatkind",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: buildName,
				},
			}
			// fake an instance of OwnerReference with a
			// well known fake Kind and Name
			fakeOwnerRef := []metav1.OwnerReference{
				{
					Kind:       "fakekind",
					Name:       buildName,
					Controller: &managingController,
				},
			}
			// Assert that our Build is not owned by an owner
			Expect(resources.IsOwnedByBuild(buildSample, fakeOwnerRef)).To(BeFalse())
		})
	})
})
