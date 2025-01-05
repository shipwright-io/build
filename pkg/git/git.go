// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package git

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/storage/memory"

	gogitv5 "github.com/go-git/go-git/v5"
)

const (
	defaultRemote = "origin"
	httpsProtocol = "https"
	httpProtocol  = "http"
	fileProtocol  = "file"
	gitProtocol   = "ssh"
)

// ExtractHostnamePort extracts the hostname and port of the provided Git URL
func ExtractHostnamePort(url string) (string, int, error) {
	endpoint, err := transport.NewEndpoint(url)
	if err != nil {
		return "", 0, err
	}

	port := endpoint.Port

	if port == 0 {
		switch endpoint.Protocol {
		case httpProtocol:
			port = 80
		case httpsProtocol:
			port = 443
		case gitProtocol:
			port = 22

		default:
			return "", 0, fmt.Errorf("unknown protocol: %s", endpoint.Protocol)
		}
	}

	return endpoint.Host, port, nil
}

// ValidateGitURLExists validate if a source URL exists or not
// Note: We have an upcoming PR for the Build Status, where we
// intend to define a single Status.Reason in the form of 'remoteRepositoryUnreachable',
// where the Status.Message will contain the longer text, like 'invalid source url
func ValidateGitURLExists(ctx context.Context, urlPath string) error {
	endpoint, err := transport.NewEndpoint(urlPath)
	if err != nil {
		return err
	}

	switch endpoint.Protocol {
	case httpsProtocol, httpProtocol:
		repo := gogitv5.NewRemote(memory.NewStorage(), &config.RemoteConfig{
			Name: defaultRemote,
			URLs: []string{urlPath},
		})

		if _, err := repo.ListContext(ctx, &gogitv5.ListOptions{}); err != nil {
			// Note: When the urlPath is an valid public path, however, this
			// path doesn't exist, func will return `authentication required`,
			// this is maybe misleading. So convert this error message to:
			// `remote repository unreachable`
			if errors.Is(err, transport.ErrAuthenticationRequired) {
				return fmt.Errorf("remote repository unreachable")
			}

			return err
		}

	case fileProtocol:
		return fmt.Errorf("invalid source url")

	case gitProtocol:
		return fmt.Errorf("the source url requires authentication")

	}

	return nil
}
