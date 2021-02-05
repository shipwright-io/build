// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	kubemetrics "github.com/operator-framework/operator-sdk/pkg/kube-metrics"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/operator-framework/operator-sdk/pkg/metrics"
	"github.com/operator-framework/operator-sdk/pkg/ready"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/spf13/pflag"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/shipwright-io/build/pkg/apis"
	buildconfig "github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/controller"
	"github.com/shipwright-io/build/pkg/ctxlog"
	buildMetrics "github.com/shipwright-io/build/pkg/metrics"
	"github.com/shipwright-io/build/version"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost               = "0.0.0.0"
	metricsPort         int32 = 8383
	operatorMetricsPort int32 = 8686
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
	pflag.CommandLine.AddFlagSet(zap.FlagSet())

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

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		ctxlog.Error(ctx, err, "Failed to get watch namespace")
		os.Exit(1)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		ctxlog.Error(ctx, err, "")
		os.Exit(1)
	}

	r := ready.NewFileReady()
	err = r.Set()
	if err != nil {
		ctxlog.Error(ctx, err, "Checking for /tmp/operator-sdk-ready failed")
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

	// Add the Metrics Service
	addMetrics(ctx, cfg, namespace)
	buildMetrics.InitPrometheus(buildCfg)

	// Add optionally configured extra handlers to metrics endpoint
	for path, handler := range buildMetrics.MetricsExtraHandlers() {
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

// addMetrics will create the Services and Service Monitors to allow the operator export the metrics by using
// the Prometheus operator
func addMetrics(ctx context.Context, cfg *rest.Config, namespace string) {
	if err := serveCRMetrics(cfg); err != nil {
		if errors.Is(err, k8sutil.ErrRunLocal) {
			ctxlog.Info(ctx, "Skipping CR metrics server creation; not running in a cluster.")
			return
		}
		ctxlog.Info(ctx, "Could not generate and serve custom resource metrics", "error", err.Error())
	}

	// Add to the below struct any other metrics ports you want to expose.
	servicePorts := []v1.ServicePort{
		{Port: metricsPort, Name: metrics.OperatorPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: metricsPort}},
		{Port: operatorMetricsPort, Name: metrics.CRPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: operatorMetricsPort}},
	}

	// Create Service object to expose the metrics port(s).
	service, err := metrics.CreateMetricsService(ctx, cfg, servicePorts)
	if err != nil {
		ctxlog.Info(ctx, "Could not create metrics Service", "error", err.Error())
	}

	// CreateServiceMonitors will automatically create the prometheus-operator ServiceMonitor resources
	// necessary to configure Prometheus to scrape metrics from this operator.
	services := []*v1.Service{service}
	_, err = metrics.CreateServiceMonitors(cfg, namespace, services)
	if err != nil {
		ctxlog.Info(ctx, "Could not create ServiceMonitor object", "error", err.Error())
		// If this operator is deployed to a cluster without the prometheus-operator running, it will return
		// ErrServiceMonitorNotPresent, which can be used to safely skip ServiceMonitor creation.
		if err == metrics.ErrServiceMonitorNotPresent {
			ctxlog.Info(ctx, "Install prometheus-operator in your cluster to create ServiceMonitor objects", "error", err.Error())
		}
	}
}

// serveCRMetrics gets the Operator/CustomResource GVKs and generates metrics based on those types.
// It serves those metrics on "http://metricsHost:operatorMetricsPort".
func serveCRMetrics(cfg *rest.Config) error {
	// Below function returns filtered operator/CustomResource specific GVKs.
	// For more control override the below GVK list with your own custom logic.
	filteredGVK, err := k8sutil.GetGVKsFromAddToScheme(apis.AddToScheme)
	if err != nil {
		return err
	}
	ns := []string{""}
	// Generate and serve custom resource specific metrics.
	err = kubemetrics.GenerateAndServeCRMetrics(cfg, ns, filteredGVK, metricsHost, operatorMetricsPort)
	if err != nil {
		return err
	}
	return nil
}
