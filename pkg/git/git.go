// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package git

import (
	"fmt"

	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/storage/memory"

	gogitv5 "github.com/go-git/go-git/v5"
)

const (
	defaultRemote = "origin"
	httpsProtocol = "https"
	fileProtocol  = "file"
	gitProtocol   = "ssh"
)

// ValidateGitURLExists validate if a source URL exists or not
// Note: We have an upcoming PR for the Build Status, where we
// intend to define a single Status.Reason in the form of 'remoteRepositoryUnreachable',
// where the Status.Message will contain the longer text, like 'invalid source url
func ValidateGitURLExists(urlPath string) error {

	endpoint, err := transport.NewEndpoint(urlPath)
	if err != nil {
		return err
	}

	switch endpoint.Protocol {
	case httpsProtocol:
		repo := gogitv5.NewRemote(memory.NewStorage(), &config.RemoteConfig{
			Name: defaultRemote,
			URLs: []string{urlPath},
		})
		// Note: When the urlPath is an valid public path, however, this path doesn't exist, func will return `authentication required`, this maybe mislead
		// So convert this error message to `repository not found`
		_, err := repo.List(&gogitv5.ListOptions{})
		if err != nil && err == transport.ErrAuthenticationRequired {
			return fmt.Errorf("remote repository unreachable")
		} else if err != nil {
			return err
		}
	case fileProtocol:
		return fmt.Errorf("invalid source url")
	case gitProtocol:
		return fmt.Errorf("the source url requires authentication")
	default:
		return nil
	}
	return nil
}
