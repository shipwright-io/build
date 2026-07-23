// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package tlsconfig

import (
	"crypto/tls"
	"fmt"
	"strings"
)

// BuildServerTLSConfig builds a tls.Config for serving HTTPS.
//
// Defaults:
//   - Minimum TLS version is TLS 1.2.
//   - Cipher suites and curve preferences are left unset to allow Go defaults
//     (including TLS 1.3 cipher negotiation and updated curve defaults).
//
// Flags:
// - minVersionFlag accepts: VersionTLS10|VersionTLS11|VersionTLS12|VersionTLS13
// - cipherSuitesFlag is a comma-separated list of Go cipher suite names (TLS 1.2 only).
//
// Returns (cfg, warning, err). A warning is returned when cipher suites are provided
// but ignored due to TLS 1.3 minimum.
func BuildServerTLSConfig(minVersionFlag, cipherSuitesFlag string) (*tls.Config, string, error) {
	minVersion, err := parseMinVersionFlag(minVersionFlag)
	if err != nil {
		return nil, "", err
	}

	cfg := &tls.Config{
		MinVersion: minVersion,
	}

	cipherSuitesFlag = strings.TrimSpace(cipherSuitesFlag)
	if cipherSuitesFlag == "" {
		return cfg, "", nil
	}

	if minVersion >= tls.VersionTLS13 {
		return cfg, "ignoring --tls-cipher-suites because --tls-min-version is TLS 1.3 or higher", nil
	}

	ciphers, err := parseCipherSuitesFlag(cipherSuitesFlag)
	if err != nil {
		return nil, "", err
	}
	cfg.CipherSuites = ciphers

	return cfg, "", nil
}

func parseMinVersionFlag(v string) (uint16, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return tls.VersionTLS12, nil
	}

	switch v {
	case "VersionTLS10":
		return tls.VersionTLS10, nil
	case "VersionTLS11":
		return tls.VersionTLS11, nil
	case "VersionTLS12":
		return tls.VersionTLS12, nil
	case "VersionTLS13":
		return tls.VersionTLS13, nil
	default:
		return 0, fmt.Errorf("invalid --tls-min-version %q (allowed: VersionTLS10, VersionTLS11, VersionTLS12, VersionTLS13)", v)
	}
}

func parseCipherSuitesFlag(csv string) ([]uint16, error) {
	parts := strings.Split(csv, ",")
	var names []string
	for _, p := range parts {
		n := strings.TrimSpace(p)
		if n != "" {
			names = append(names, n)
		}
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("invalid --tls-cipher-suites: no cipher suites provided")
	}

	byName := cipherSuiteNameMap()
	out := make([]uint16, 0, len(names))
	for _, name := range names {
		id, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("invalid --tls-cipher-suites entry %q (must be a Go cipher suite name, e.g. TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256)", name)
		}
		out = append(out, id)
	}
	return out, nil
}

func cipherSuiteNameMap() map[string]uint16 {
	m := make(map[string]uint16, 64)
	for _, cs := range tls.CipherSuites() {
		m[cs.Name] = cs.ID
	}
	for _, cs := range tls.InsecureCipherSuites() {
		m[cs.Name] = cs.ID
	}
	return m
}
