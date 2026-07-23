// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path"
	"runtime"
	"time"

	"github.com/spf13/pflag"
	"knative.dev/pkg/signals"

	"github.com/shipwright-io/build/pkg/ctxlog"
	"github.com/shipwright-io/build/pkg/webhook/conversion"
	"github.com/shipwright-io/build/pkg/webhook/tlsconfig"
	"github.com/shipwright-io/build/version"
)

var (
	versionGiven    = flag.String("version", "devel", "Version of Shipwright webhook running")
	tlsMinVersion   = pflag.String("tls-min-version", "", "Minimum TLS version for the webhook HTTPS server (VersionTLS10, VersionTLS11, VersionTLS12, VersionTLS13). Defaults to VersionTLS12.")
	tlsCipherSuites = pflag.String("tls-cipher-suites", "", "Comma-separated list of TLS 1.2 cipher suites (Go cipher suite names). Only applies when the minimum TLS version is below TLS 1.3. Defaults to Go runtime selection.")
)

func printVersion(ctx context.Context) {
	ctxlog.Info(ctx, fmt.Sprintf("Shipwright Build Webhook Version: %s", version.Version))
	ctxlog.Info(ctx, fmt.Sprintf("Go Version: %s", runtime.Version()))
	ctxlog.Info(ctx, fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}

func main() {
	// Add the zap logger flag set to the CLI. The flag set must
	// be added before calling pflag.Parse().
	pflag.CommandLine.AddGoFlagSet(ctxlog.CustomZapFlagSet())

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	pflag.Parse()

	if err := Execute(); err != nil {
		os.Exit(1)
	}

}

func Execute() error {
	l := ctxlog.NewLogger("shp-build-webhook")

	ctx := ctxlog.NewParentContext(l)

	version.SetVersion(*versionGiven)
	printVersion(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", health)
	ctxlog.Info(ctx, "adding handlefunc() /health")

	// convert endpoint handles ConversionReview API object serialized to JSON
	mux.HandleFunc("/convert", conversion.CRDConvertHandler(ctx))
	ctxlog.Info(ctx, "adding handlefunc() /convert")

	serverTLSConfig, warning, err := tlsconfig.BuildServerTLSConfig(*tlsMinVersion, *tlsCipherSuites)
	if err != nil {
		ctxlog.Error(ctx, err, "invalid TLS configuration")
		return err
	}
	if warning != "" {
		ctxlog.Info(ctx, warning)
	}
	ctxlog.Info(
		ctx,
		fmt.Sprintf(
			"effective TLS configuration: minVersion=%d cipherSuitesConfigured=%t cipherSuitesCount=%d",
			serverTLSConfig.MinVersion,
			serverTLSConfig.CipherSuites != nil,
			len(serverTLSConfig.CipherSuites),
		),
	)

	server := &http.Server{
		Addr:              ":8443",
		Handler:           mux,
		ReadHeaderTimeout: 32 * time.Second,
		TLSConfig:         serverTLSConfig,
	}

	go func() {
		ctxlog.Info(ctx, "starting webhook server")
		// blocking call, returns on error
		if err := server.ListenAndServeTLS(path.Join("/etc/webhook/certs", "tls.crt"), path.Join("/etc/webhook/certs", "tls.key")); err != nil {
			ctxlog.Error(ctx, err, "webhook server failed to start")
		}
	}()

	stopCh := signals.SetupSignalHandler()
	sig := <-stopCh

	l.Info("Shutting down server.", "signal", sig)
	ctxlog.Info(ctx, "shutting down webhook server,", "signal:", sig)
	if err := server.Shutdown(context.Background()); err != nil {
		l.Error(err, "Failed to gracefully shutdown the server.")
		return err
	}
	return nil

}

func health(resp http.ResponseWriter, _ *http.Request) {
	resp.WriteHeader(http.StatusNoContent)
}
