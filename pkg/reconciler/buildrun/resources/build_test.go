// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	crc "sigs.k8s.io/controller-runtime/pkg/client"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/controller/fakes"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
	"github.com/shipwright-io/build/test"
)

var _ = Describe("Build Resource", func() {
	var (
		client *fakes.FakeClient
		ctl    test.Catalog
	)

	Context("Operating on Build resources", func() {
		// init vars
		buildName := "foobuild"
		client = &fakes.FakeClient{}
		buildRun := &build.BuildRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Spec: build.BuildRunSpec{
				BuildRef: &build.BuildRef{
					Name: buildName,
				},
			},
		}

		It("should be able to retrieve a build object if exists", func() {
			buildSample := ctl.DefaultBuild(buildName, "foostrategy", build.ClusterBuildStrategyKind)

			// stub a GET API call with buildSample contents
			getClientStub := func(_ context.Context, nn types.NamespacedName, object crc.Object) error {
				switch object := object.(type) {
				case *build.Build:
					buildSample.DeepCopyInto(object)
					return nil
				}
				return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
			}

			// fake the calls with the above stub definition
			client.GetCalls(getClientStub)

			buildObject := &build.Build{}
			Expect(resources.GetBuildObject(context.TODO(), client, buildRun, buildObject)).To(BeNil())
		})

		It("should not retrieve a missing build object when missing", func() {
			// stub a GET API call that returns "not found"
			client.GetCalls(func(_ context.Context, nn types.NamespacedName, object crc.Object) error {
				return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
			})

			client.StatusCalls(func() crc.StatusWriter {
				return &fakes.FakeStatusWriter{}
			})

			build := &build.Build{}
			Expect(resources.GetBuildObject(context.TODO(), client, buildRun, build)).ToNot(BeNil())
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

	Context("Operating on embedded Build(Spec) resources", func() {
		client = &fakes.FakeClient{}
		buildRun := &build.BuildRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Spec: build.BuildRunSpec{
				BuildSpec: &build.BuildSpec{
					Env: []v1.EnvVar{{Name: "foo", Value: "bar"}},
				},
			},
		}

		It("should be able to retrieve an embedded build object if it exists", func() {
			build := &build.Build{}
			err := resources.GetBuildObject(context.TODO(), client, buildRun, build)

			Expect(err).To(BeNil())
			Expect(build).ToNot(BeNil())
			Expect(build.Spec).ToNot(BeNil())
			Expect(build.Spec.Env).ToNot(BeNil())
			Expect(build.Spec.Env).To(ContainElement(v1.EnvVar{Name: "foo", Value: "bar"}))
		})
	})
})
