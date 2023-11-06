// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// optionsRoundTripper ensures that the insecure flag is honored, and sets headers on outgoing requests
type optionsRoundTripper struct {
	inner       http.RoundTripper
	httpHeaders map[string]string
	insecure    bool
	userAgent   string
}

// RoundTrip implements http.RoundTripper
func (ort *optionsRoundTripper) RoundTrip(in *http.Request) (*http.Response, error) {
	// Make sure that HTTP traffic is only used if insecure is set to true
	if !ort.insecure && in.URL != nil && in.URL.Scheme == "http" {
		return nil, errors.New("the http protocol is not allowed")
	}

	for k, v := range ort.httpHeaders {
		in.Header.Set(k, v)
	}

	in.Header.Set("User-Agent", ort.userAgent)

	return ort.inner.RoundTrip(in)
}

// GetOptions constructs go-containerregistry options to access the remote registry, in addition, it returns the authentication separately which can be an empty object
func GetOptions(ctx context.Context, imageName name.Reference, insecure bool, dockerConfigJSONPath string, userAgent string) ([]remote.Option, *authn.AuthConfig, error) {
	var options []remote.Option

	options = append(options, remote.WithContext(ctx))

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: false,
	}

	if insecure {
		// #nosec:G402 insecure is explicitly requested by user, make sure to skip verification and reset empty defaults
		transport.TLSClientConfig.InsecureSkipVerify = insecure
		transport.TLSClientConfig.MinVersion = 0
	}

	// find a Docker config.json
	if dockerConfigJSONPath != "" {
		// if we have a value provided already, then we support a directory that contains a .dockerconfigjson file
		mountedSecretDefaultFileName := filepath.Join(dockerConfigJSONPath, ".dockerconfigjson")
		if fileInfo, err := os.Stat(mountedSecretDefaultFileName); err == nil && !fileInfo.IsDir() {
			dockerConfigJSONPath = mountedSecretDefaultFileName
		}
	}

	var dockerconfig *configfile.ConfigFile
	if dockerConfigJSONPath != "" {
		file, err := os.Open(dockerConfigJSONPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open the config json: %w", err)
		}
		defer file.Close()

		if dockerconfig, err = config.LoadFromReader(file); err != nil {
			return nil, nil, err
		}
	}

	// Add a RoundTripper
	rt := &optionsRoundTripper{
		inner:     transport,
		insecure:  insecure,
		userAgent: userAgent,
	}
	if dockerconfig != nil && len(dockerconfig.HTTPHeaders) > 0 {
		rt.httpHeaders = dockerconfig.HTTPHeaders
	}
	options = append(options, remote.WithTransport(rt))

	// Authentication
	auth := authn.AuthConfig{}
	if dockerconfig != nil {
		registryName := imageName.Context().RegistryStr()
		if registryName == name.DefaultRegistry {
			registryName = authn.DefaultAuthKey
		}

		authConfig, err := dockerconfig.GetAuthConfig(registryName)
		if err != nil {
			return nil, nil, err
		}

		// Return an error in case the credentials do not match the desired
		// registry and list all servers that actually are available
		if authConfig.ServerAddress != registryName {
			var servers []string
			for name := range dockerconfig.GetAuthConfigs() {
				servers = append(servers, name)
			}

			var availableConfigs string
			if len(servers) > 0 {
				availableConfigs = strings.Join(servers, ", ")
			} else {
				availableConfigs = "none"
			}

			return nil, nil, fmt.Errorf("failed to find registry credentials for %s, available configurations: %s",
				registryName,
				availableConfigs,
			)
		}

		auth.Username = authConfig.Username
		auth.Password = authConfig.Password
		auth.Auth = authConfig.Auth
		auth.IdentityToken = authConfig.IdentityToken
		auth.RegistryToken = authConfig.RegistryToken
	}

	options = append(options, remote.WithAuth(authn.FromConfig(auth)))

	return options, &auth, nil
}
