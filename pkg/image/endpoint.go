// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
)

// ExtractHostnamePort tries to extract the hostname and port of the provided image URL
func ExtractHostnamePort(url string) (string, int, error) {
	ref, err := name.ParseReference(url)
	if err != nil {
		return "", 0, err
	}

	registry := ref.Context().Registry
	host := registry.RegistryStr()
	hostname := host
	port := 0

	parts := strings.SplitN(host, ":", 2)
	if len(parts) == 2 {
		hostname = parts[0]
		if port, err = strconv.Atoi(parts[1]); err != nil {
			return "", 0, err
		}
	} else {
		scheme := registry.Scheme()

		switch scheme {
		case "http":
			port = 80
		case "https":
			port = 443

		default:
			return "", 0, fmt.Errorf("Unknown protocol: %s", scheme)
		}
	}

	return hostname, port, nil
}
