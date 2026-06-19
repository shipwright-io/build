// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/shipwright-io/build/pkg/webhook/conversion"
	"github.com/shipwright-io/build/pkg/webhook/tlsconfig"
)

func TestClient() *http.Client {
	transport := &http.Transport{
		IdleConnTimeout:       5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	// #nosec:G402 test code
	transport.TLSClientConfig.InsecureSkipVerify = true

	return &http.Client{
		Transport: transport,
	}
}

func StartBuildWebhook() *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/convert", conversion.CRDConvertHandler(context.Background()))
	mux.HandleFunc("/health", health)

	serverTLSConfig, _, err := tlsconfig.BuildServerTLSConfig("", "")
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	webhookServer := &http.Server{
		Addr:              ":30443",
		Handler:           mux,
		ReadHeaderTimeout: 32 * time.Second,
		IdleTimeout:       time.Second,
		TLSConfig:         serverTLSConfig,
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
		r, err := TestClient().Get("https://localhost:30443/health")
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
		r, err := TestClient().Get("https://localhost:30443/health")
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
