// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package tlsconfig

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"
)

func TestBuildServerTLSConfig_MinVersionDefaultsToTLS12(t *testing.T) {
	cfg, warning, err := BuildServerTLSConfig("", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if warning != "" {
		t.Fatalf("expected no warning, got %q", warning)
	}
	if cfg.MinVersion != tls.VersionTLS12 {
		t.Fatalf("expected MinVersion TLS1.2, got %d", cfg.MinVersion)
	}
	if cfg.CipherSuites != nil {
		t.Fatalf("expected CipherSuites to be unset by default")
	}
	if cfg.CurvePreferences != nil {
		t.Fatalf("expected CurvePreferences to be unset by default")
	}
}

func TestBuildServerTLSConfig_MinVersionParsing(t *testing.T) {
	cases := []struct {
		in   string
		want uint16
	}{
		{"VersionTLS10", tls.VersionTLS10},
		{"VersionTLS11", tls.VersionTLS11},
		{"VersionTLS12", tls.VersionTLS12},
		{"VersionTLS13", tls.VersionTLS13},
		{"  VersionTLS12  ", tls.VersionTLS12},
	}

	for _, tc := range cases {
		cfg, warning, err := BuildServerTLSConfig(tc.in, "")
		if err != nil {
			t.Fatalf("minVersion %q: expected no error, got %v", tc.in, err)
		}
		if warning != "" {
			t.Fatalf("minVersion %q: expected no warning, got %q", tc.in, warning)
		}
		if cfg.MinVersion != tc.want {
			t.Fatalf("minVersion %q: expected %d, got %d", tc.in, tc.want, cfg.MinVersion)
		}
	}

	if _, _, err := BuildServerTLSConfig("VersionTLS14", ""); err == nil {
		t.Fatalf("expected error for invalid min version")
	}
}

func TestBuildServerTLSConfig_CipherSuiteParsingAndOrderPreserved(t *testing.T) {
	a := "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
	b := "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"

	cfg, warning, err := BuildServerTLSConfig("VersionTLS12", a+","+b)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if warning != "" {
		t.Fatalf("expected no warning, got %q", warning)
	}
	if cfg.CipherSuites == nil || len(cfg.CipherSuites) != 2 {
		t.Fatalf("expected 2 cipher suites, got %#v", cfg.CipherSuites)
	}

	m := cipherSuiteNameMap()
	if cfg.CipherSuites[0] != m[a] || cfg.CipherSuites[1] != m[b] {
		t.Fatalf("expected order [%d,%d], got [%d,%d]", m[a], m[b], cfg.CipherSuites[0], cfg.CipherSuites[1])
	}
}

func TestBuildServerTLSConfig_TLS13IgnoresCipherSuitesWithWarning(t *testing.T) {
	cfg, warning, err := BuildServerTLSConfig("VersionTLS13", "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.MinVersion != tls.VersionTLS13 {
		t.Fatalf("expected MinVersion TLS1.3, got %d", cfg.MinVersion)
	}
	if warning == "" || !strings.Contains(warning, "ignoring --tls-cipher-suites") {
		t.Fatalf("expected ignore warning, got %q", warning)
	}
	if cfg.CipherSuites != nil {
		t.Fatalf("expected CipherSuites to be unset for TLS 1.3 minimum")
	}
}

func TestBuildServerTLSConfig_InvalidCipherSuitesFailFast(t *testing.T) {
	if _, _, err := BuildServerTLSConfig("VersionTLS12", ""); err != nil {
		t.Fatalf("empty flag should be treated as unset, got %v", err)
	}

	if _, _, err := BuildServerTLSConfig("VersionTLS12", " , , "); err == nil {
		t.Fatalf("expected error for empty cipher suite list")
	}

	if _, _, err := BuildServerTLSConfig("VersionTLS12", "TLS_NOT_A_REAL_CIPHER"); err == nil {
		t.Fatalf("expected error for invalid cipher suite name")
	}
}

func testTLSClientConfig(t *testing.T, minVersion, maxVersion uint16, cipherSuites []uint16) *tls.Config {
	t.Helper()

	cfg := &tls.Config{
		MinVersion: minVersion,
		MaxVersion: maxVersion,
	}
	if len(cipherSuites) > 0 {
		cfg.CipherSuites = cipherSuites
	}
	// #nosec:G402 test code uses ephemeral self-signed certificates
	cfg.InsecureSkipVerify = true
	return cfg
}

func TestHandshake_DefaultConfigAcceptsTLS12Client(t *testing.T) {
	serverCfg, _, err := BuildServerTLSConfig("", "")
	if err != nil {
		t.Fatalf("expected no error building server config, got %v", err)
	}

	addr, stop := startHandshakeTestServer(t, serverCfg)
	defer stop()

	clientCfg := testTLSClientConfig(t, tls.VersionTLS12, tls.VersionTLS12, nil)

	if err := dialTLS(addr, clientCfg); err != nil {
		t.Fatalf("expected TLS 1.2 handshake to succeed with defaults, got %v", err)
	}
}

func TestHandshake_TLS13MinimumRejectsTLS12Client(t *testing.T) {
	serverCfg, _, err := BuildServerTLSConfig("VersionTLS13", "")
	if err != nil {
		t.Fatalf("expected no error building server config, got %v", err)
	}

	addr, stop := startHandshakeTestServer(t, serverCfg)
	defer stop()

	clientCfg := testTLSClientConfig(t, tls.VersionTLS12, tls.VersionTLS12, nil)

	if err := dialTLS(addr, clientCfg); err == nil {
		t.Fatal("expected TLS 1.2 client to be rejected when server minimum is TLS 1.3")
	}
}

func TestHandshake_TLS13MinimumAcceptsTLS13Client(t *testing.T) {
	serverCfg, _, err := BuildServerTLSConfig("VersionTLS13", "")
	if err != nil {
		t.Fatalf("expected no error building server config, got %v", err)
	}

	addr, stop := startHandshakeTestServer(t, serverCfg)
	defer stop()

	clientCfg := testTLSClientConfig(t, tls.VersionTLS13, tls.VersionTLS13, nil)

	if err := dialTLS(addr, clientCfg); err != nil {
		t.Fatalf("expected TLS 1.3 handshake to succeed, got %v", err)
	}
}

func TestHandshake_CipherAllowlistRestrictsTLS12Negotiation(t *testing.T) {
	const allowedCipher = "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256"
	const disallowedCipher = "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384"

	serverCfg, _, err := BuildServerTLSConfig("VersionTLS12", allowedCipher)
	if err != nil {
		t.Fatalf("expected no error building server config, got %v", err)
	}

	addr, stop := startHandshakeTestServer(t, serverCfg)
	defer stop()

	suiteIDs := cipherSuiteNameMap()
	allowedID, ok := suiteIDs[allowedCipher]
	if !ok {
		t.Fatalf("expected cipher suite %q to exist", allowedCipher)
	}
	disallowedID, ok := suiteIDs[disallowedCipher]
	if !ok {
		t.Fatalf("expected cipher suite %q to exist", disallowedCipher)
	}

	allowedClient := testTLSClientConfig(t, tls.VersionTLS12, tls.VersionTLS12, []uint16{allowedID})
	if err := dialTLS(addr, allowedClient); err != nil {
		t.Fatalf("expected allowed cipher suite to handshake successfully, got %v", err)
	}

	disallowedClient := testTLSClientConfig(t, tls.VersionTLS12, tls.VersionTLS12, []uint16{disallowedID})
	if err := dialTLS(addr, disallowedClient); err == nil {
		t.Fatal("expected disallowed cipher suite to be rejected by server allowlist")
	}
}

func generateTestCertificate(t *testing.T) tls.Certificate {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "shipwright-webhook-test",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	return tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}
}

func startHandshakeTestServer(t *testing.T, serverCfg *tls.Config) (string, func()) {
	t.Helper()

	serverCfg = serverCfg.Clone()
	serverCfg.Certificates = []tls.Certificate{generateTestCertificate(t)}

	ln, err := tls.Listen("tcp", "127.0.0.1:0", serverCfg)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				tlsConn, ok := c.(*tls.Conn)
				if !ok {
					return
				}
				_ = tlsConn.Handshake()
			}(conn)
		}
	}()

	return ln.Addr().String(), func() {
		_ = ln.Close()
		<-done
	}
}

func dialTLS(addr string, cfg *tls.Config) error {
	conn, err := tls.Dial("tcp", addr, cfg)
	if err != nil {
		return err
	}
	return conn.Close()
}
