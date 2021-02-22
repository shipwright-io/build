// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/spf13/pflag"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	buildconfig "github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/controller"
	"github.com/shipwright-io/build/pkg/controller/ready"
	"github.com/shipwright-io/build/pkg/ctxlog"
	buildMetrics "github.com/shipwright-io/build/pkg/metrics"
	"github.com/shipwright-io/build/version"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost       = "0.0.0.0"
	metricsPort int32 = 8383
)

func printVersion(ctx context.Context) {
	ctxlog.Info(ctx, fmt.Sprintf("Operator Version: %s", version.Version))
	ctxlog.Info(ctx, fmt.Sprintf("Go Version: %s", runtime.Version()))
	ctxlog.Info(ctx, fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	ctxlog.Info(ctx, fmt.Sprintf("Version of operator-sdk: %v", sdkVersion.Version))
}

func main() {
	// Add the zap logger flag set to the CLI. The flag set must
	// be added before calling pflag.Parse().
	pflag.CommandLine.AddGoFlagSet(ctxlog.CustomZapFlagSet())

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	pflag.Parse()

	// Use a zap logr.Logger implementation. If none of the zap
	// flags are configured (or if the zap flag set is not being
	// used), this defaults to a production zap logger.
	//
	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.

	l := ctxlog.NewLogger("build")

	ctx := ctxlog.NewParentContext(l)

	printVersion(ctx)

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		ctxlog.Error(ctx, err, "")
		os.Exit(1)
	}

	r := ready.NewFileReady("/tmp/shipwright-build-ready")
	err = r.Set()
	if err != nil {
		ctxlog.Error(ctx, err, "Checking for /tmp/shipwright-build-ready failed")
		os.Exit(1)
	}
	defer r.Unset()

	buildCfg := buildconfig.NewDefaultConfig()
	if err := buildCfg.SetConfigFromEnv(); err != nil {
		ctxlog.Error(ctx, err, "")
		os.Exit(1)
	}

	mgr, err := controller.NewManager(ctx, buildCfg, cfg, manager.Options{
		LeaderElection:          true,
		LeaderElectionID:        "build-operator-lock",
		LeaderElectionNamespace: buildCfg.ManagerOptions.LeaderElectionNamespace,
		LeaseDuration:           buildCfg.ManagerOptions.LeaseDuration,
		RenewDeadline:           buildCfg.ManagerOptions.RenewDeadline,
		RetryPeriod:             buildCfg.ManagerOptions.RetryPeriod,
		Namespace:               "",
		MetricsBindAddress:      fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	})
	if err != nil {
		ctxlog.Error(ctx, err, "")
		os.Exit(1)
	}

	buildMetrics.InitPrometheus(buildCfg)

	// Add optionally configured extra handlers to metrics endpoint
	for path, handler := range buildMetrics.ExtraHandlers() {
		ctxlog.Info(ctx, "Adding metrics extra handler path", "path", path)
		if err := mgr.AddMetricsExtraHandler(path, handler); err != nil {
			ctxlog.Error(ctx, err, "")
			os.Exit(2)
		}
	}

	// Start the Cmd
	ctxlog.Info(ctx, "Starting the Cmd.")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		ctxlog.Error(ctx, err, "Manager exited non-zero")
		os.Exit(1)
	}
}
