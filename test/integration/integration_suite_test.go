// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/shipwright-io/build/test/utils"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

// TODO: clean resources in cluster, e.g. mainly cluster-scope ones
// TODO: clean each resource created per spec
var (
	tb  *utils.TestBuild
	err error
)

var _ = BeforeEach(func() {
	tb, err = utils.NewTestBuild()
	if err != nil {
		fmt.Printf("fail to get an instance of TestBuild, error is: %v", err)
	}

	err := tb.CreateNamespace()
	if err != nil {
		fmt.Printf("fail to create namespace: %v, with error: %v", tb.Namespace, err)
	}

	// We store a channel for each Build controller instance we start,
	// so that we can nuke the instance later inside the AfterEach Ginkgo
	// block
	tb.StopBuildControllers, err = tb.StartBuildControllers()
	if err != nil {
		fmt.Println("fail to start the powerful Build controllers", err)
	}
})

var _ = AfterEach(func() {
	// Close the channel, meaning we nuke an instance of the Build
	// operator
	if tb.StopBuildControllers != nil {
		close(tb.StopBuildControllers)
	}

	// Cleanup the namespace
	if err := tb.DeleteNamespace(); err != nil {
		fmt.Printf("failed to delete namespace: %v, with error: %v", tb.Namespace, err)
	}

	if CurrentGinkgoTestDescription().Failed && tb.BuildControllerLogBuffer != nil {
		// print operator logs
		fmt.Println("\nLogs of the operator:")
		fmt.Printf("%v\n", tb.BuildControllerLogBuffer)
	}
})
