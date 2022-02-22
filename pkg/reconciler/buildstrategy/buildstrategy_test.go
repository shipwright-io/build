// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package buildstrategy_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/controller/fakes"
	"github.com/shipwright-io/build/pkg/reconciler/buildstrategy"
)

var _ = Describe("Reconcile BuildStrategy", func() {
	var (
		manager                      *fakes.FakeManager
		reconciler                   reconcile.Reconciler
		request                      reconcile.Request
		namespace, buildStrategyName string
	)

	BeforeEach(func() {
		buildStrategyName = "buildah"
		namespace = "build-examples"

		// Fake the manager and get a reconcile Request
		manager = &fakes.FakeManager{}
		request = reconcile.Request{NamespacedName: types.NamespacedName{Name: buildStrategyName, Namespace: namespace}}
	})

	JustBeforeEach(func() {
		// Reconcile
		reconciler = buildstrategy.NewReconciler(config.NewDefaultConfig(), manager)
	})

	Describe("Reconcile", func() {
		Context("when request a new BuildStrategy", func() {
			It("succeed without any error", func() {
				result, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
			})
		})
	})
})
