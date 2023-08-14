// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path"
	"runtime"
	"time"

	"github.com/shipwright-io/build/pkg/ctxlog"
	"github.com/shipwright-io/build/pkg/webhook/conversion"
	"github.com/shipwright-io/build/version"
	"github.com/spf13/pflag"
	"knative.dev/pkg/signals"
)

var (
	versionGiven = flag.String("version", "devel", "Version of Shipwright webhook running")
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

	server := &http.Server{
		Addr:              ":8443",
		Handler:           mux,
		ReadHeaderTimeout: 32 * time.Second,
		TLSConfig: &tls.Config{
			MinVersion:       tls.VersionTLS12,
			CurvePreferences: []tls.CurveID{tls.CurveP256, tls.CurveP384, tls.X25519},
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			},
		},
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
