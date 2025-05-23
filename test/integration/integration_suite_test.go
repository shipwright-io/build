// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	utils "github.com/shipwright-io/build/test/utils/v1beta1"
)

const (
	BUILD    = "build-"
	BUILDRUN = "buildrun-"
	STRATEGY = "strategy-"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

// TODO: clean resources in cluster, e.g. mainly cluster-scope ones
// TODO: clean each resource created per spec
var (
	tb            *utils.TestBuild
	err           error
	webhookServer *http.Server

	controllerLogs *bytes.Buffer
)

var _ = BeforeSuite(func() {
	webhookServer = utils.StartBuildWebhook()

	var err error
	controllerLogs, err = utils.StartBuildControllers()
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	if webhookServer != nil {
		utils.StopBuildWebhook(webhookServer)
	}
})

var _ = BeforeEach(func() {
	tb, err = utils.NewTestBuild()
	if err != nil {
		fmt.Printf("fail to get an instance of TestBuild, error is: %v", err)
	}

	err := tb.CreateNamespace()
	if err != nil {
		fmt.Printf("fail to create namespace: %v, with error: %v", tb.Namespace, err)
	}

	controllerLogs.Reset()
})

var _ = AfterEach(func() {
	// Cleanup the namespace
	if err := tb.DeleteNamespace(); err != nil {
		fmt.Printf("failed to delete namespace: %v, with error: %v", tb.Namespace, err)
	}

	if CurrentSpecReport().Failed() {
		// print operator logs
		fmt.Println("\nLogs of the operator:")
		fmt.Printf("%v\n", controllerLogs)
	}
})
