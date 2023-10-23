// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/shipwright-io/build/pkg/webhook/conversion"
	"github.com/shipwright-io/build/test/utils"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

func StartBuildWebhook() *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/convert", conversion.CRDConvertHandler(context.Background()))
	mux.HandleFunc("/health", health)

	webhookServer := &http.Server{
		Addr:              ":30443",
		Handler:           mux,
		ReadHeaderTimeout: 32 * time.Second,
		IdleTimeout:       time.Second,
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

	// start server
	go func() {
		defer ginkgo.GinkgoRecover()

		if err := webhookServer.ListenAndServeTLS("/tmp/server-cert.pem", "/tmp/server-key.pem"); err != nil {
			if err != http.ErrServerClosed {
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
			}
		}
	}()

	gomega.Eventually(func() int {
		r, err := utils.TestClient().Get("https://localhost:30443/health")
		if err != nil {
			return 0
		}
		if r != nil {
			return r.StatusCode
		}
		return 0
	}).WithTimeout(10 * time.Second).Should(gomega.Equal(http.StatusNoContent))

	return webhookServer
}

func StopBuildWebhook(webhookServer *http.Server) {
	err := webhookServer.Close()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	gomega.Eventually(func() int {
		r, err := utils.TestClient().Get("https://localhost:30443/health")
		if err != nil {
			return 0
		}
		if r != nil {
			return r.StatusCode
		}
		return 0
	}).WithTimeout(10 * time.Second).Should(gomega.Equal(0))
}

func health(resp http.ResponseWriter, _ *http.Request) {
	resp.WriteHeader(http.StatusNoContent)
}
