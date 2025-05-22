// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package runner

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

var testScheme *runtime.Scheme

func TestResources(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Runner Suite")
}

var _ = BeforeSuite(func() {
	testScheme = scheme.Scheme
	Expect(pipelinev1.AddToScheme(testScheme)).To(Succeed())
	Expect(buildv1beta1.AddToScheme(testScheme)).To(Succeed())
})
