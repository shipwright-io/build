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
// - minVersionFlag accepts: 1.0|1.1|1.2|1.3
// - cipherSuitesFlag is a comma-separated list of Go cipher suite names (TLS 1.2 only).
// - curvePreferencesFlag is a comma-separated list of supported group names.
//
// Returns (cfg, warning, err). A warning is returned when cipher suites are provided
// but ignored due to TLS 1.3 minimum.
func BuildServerTLSConfig(minVersionFlag, cipherSuitesFlag, curvePreferencesFlag string) (*tls.Config, string, error) {
	minVersion, err := parseMinVersionFlag(minVersionFlag)
	if err != nil {
		return nil, "", err
	}

	cfg := &tls.Config{
		MinVersion: minVersion,
	}

	cipherSuitesFlag = strings.TrimSpace(cipherSuitesFlag)
	curvePreferencesFlag = strings.TrimSpace(curvePreferencesFlag)

	var warning string

	// Parse cipher suites
	if cipherSuitesFlag != "" {
		if minVersion >= tls.VersionTLS13 {
			warning = "ignoring --tls-cipher-suites because --tls-min-version is TLS 1.3 or higher"
		} else {
			ciphers, err := parseCipherSuitesFlag(cipherSuitesFlag)
			if err != nil {
				return nil, "", err
			}
			cfg.CipherSuites = ciphers
		}
	}

	// Parse curve preferences
	if curvePreferencesFlag != "" {
		curves, err := parseCurvePreferencesFlag(curvePreferencesFlag)
		if err != nil {
			return nil, "", err
		}
		cfg.CurvePreferences = curves
	}

	return cfg, warning, nil
}

func parseMinVersionFlag(v string) (uint16, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return tls.VersionTLS12, nil
	}

	switch v {
	case "1.0", "VersionTLS10":
		return tls.VersionTLS10, nil
	case "1.1", "VersionTLS11":
		return tls.VersionTLS11, nil
	case "1.2", "VersionTLS12":
		return tls.VersionTLS12, nil
	case "1.3", "VersionTLS13":
		return tls.VersionTLS13, nil
	default:
		return 0, fmt.Errorf("invalid --tls-min-version %q (allowed: 1.0, 1.1, 1.2, 1.3)", v)
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

func parseCurvePreferencesFlag(csv string) ([]tls.CurveID, error) {
	parts := strings.Split(csv, ",")
	var names []string
	for _, p := range parts {
		n := strings.TrimSpace(p)
		if n != "" {
			names = append(names, n)
		}
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("invalid --tls-curve-preferences: no curve preferences provided")
	}

	byName := groupNameMap()
	out := make([]tls.CurveID, 0, len(names))
	for _, name := range names {
		id, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("invalid --tls-curve-preferences entry %q (must be one of: X25519, secp256r1, secp384r1, secp521r1, X25519MLKEM768, SecP256r1MLKEM768, SecP384r1MLKEM1024)", name)
		}
		out = append(out, id)
	}
	return out, nil
}

func groupNameMap() map[string]tls.CurveID {
	return map[string]tls.CurveID{
		"X25519":             tls.X25519,
		"secp256r1":          tls.CurveP256,
		"secp384r1":          tls.CurveP384,
		"secp521r1":          tls.CurveP521,
		"X25519MLKEM768":     tls.X25519MLKEM768,
		"SecP256r1MLKEM768":  tls.SecP256r1MLKEM768,
		"SecP384r1MLKEM1024": tls.SecP384r1MLKEM1024,
	}
}
