// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	buildconfig "github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/controller"
	"github.com/shipwright-io/build/test/integration/utils"
)

// Logf logs data
func Logf(format string, args ...interface{}) {
	currentTime := time.Now().UTC().Format(time.RFC3339)

	fmt.Fprintf(
		GinkgoWriter,
		fmt.Sprintf("%s %d %s\n", currentTime, getGinkgoNode(), format),
		args...,
	)
}

func getArg(argName string) (bool, string) {
	for i, arg := range os.Args {
		if arg == argName {
			return true, os.Args[i+1]
		} else if strings.HasPrefix(arg, argName+"=") {
			argAndValue := strings.SplitN(arg, "=", 2)
			return true, argAndValue[1]
		}
	}
	return false, ""
}

func getGinkgoNode() int {
	defined, ginkgoNodeString := getArg("--ginkgo.parallel.node")
	if !defined {
		return 1
	}
	ginkgoNode, err := strconv.Atoi(ginkgoNodeString)
	if err != nil {
		fmt.Printf("Error: %s", err.Error())
		return 0
	}
	return ginkgoNode
}

func startLocalOperator(testBuild *utils.TestBuild, stop chan struct{}) {
	buildCfg := buildconfig.NewDefaultConfig()
	err := buildCfg.SetConfigFromEnv()
	Expect(err).ToNot(HaveOccurred())

	mgr, err := controller.NewManager(testBuild.Context, buildCfg, testBuild.KubeConfig, manager.Options{
		LeaderElection:          true,
		LeaderElectionID:        "shipwright-build-controller-lock",
		LeaderElectionNamespace: testBuild.Namespace,
		LeaseDuration:           buildCfg.ManagerOptions.LeaseDuration,
		RenewDeadline:           buildCfg.ManagerOptions.RenewDeadline,
		RetryPeriod:             buildCfg.ManagerOptions.RetryPeriod,
		Namespace:               "",
	})
	Expect(err).ToNot(HaveOccurred())

	go func() {
		err := mgr.Start(stop)
		Expect(err).ToNot(HaveOccurred(), "Failed to start local operator")
	}()
}
