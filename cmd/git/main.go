// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/shipwright-io/build/pkg/ctxlog"
	"github.com/spf13/pflag"
)

type credentialType int

const (
	typeUndef credentialType = iota
	typePrivateKey
	typeUsernamePassword
)

// ExitError is an error which has an exit code to be used in os.Exit() to
// return both an exit code and an error message
type ExitError struct {
	Code    int
	Message string
	Cause   error
}

func (e ExitError) Error() string {
	return fmt.Sprintf("%s (exit code %d)", e.Message, e.Code)
}

type settings struct {
	url                 string
	revision            string
	target              string
	resultFileCommitSha string
	secretPath          string
}

var flagValues settings

var (
	sshGitURLRegEx = regexp.MustCompile(`^(git@|ssh:\/\/).+$`)
	commitShaRegEx = regexp.MustCompile(`^[0-9a-f]{7,40}$`)
)

func init() {
	// Add the zap logger flag set to the CLI. The flag set must
	// be added before calling pflag.Parse().
	pflag.CommandLine.AddGoFlagSet(ctxlog.CustomZapFlagSet())

	pflag.StringVar(&flagValues.url, "url", "", "The URL of the Git repository")
	pflag.StringVar(&flagValues.revision, "revision", "", "The revision of the Git repository to be cloned. Optional, defaults to the default branch.")
	pflag.StringVar(&flagValues.target, "target", "", "The target directory of the clone operation")
	pflag.StringVar(&flagValues.resultFileCommitSha, "result-file-commit-sha", "", "A file to write the commit sha to.")
	pflag.StringVar(&flagValues.secretPath, "secret-path", "", "A directory that contains a secret. Either username and password for basic authentication. Or a SSH private key and optionally a known hosts file. Optional.")
}

func main() {
	if err := checkAndRun(); err != nil {
		var exitcode = 1
		switch err := err.(type) {
		case *ExitError:
			exitcode = err.Code
		}

		os.Exit(exitcode)
	}
}

func checkAndRun() error {
	// create logger and context
	l := ctxlog.NewLogger("git")
	ctx := ctxlog.NewParentContext(l)

	if err := checkEnvironment(ctx); err != nil {
		return err
	}

	return Execute(ctx)
}

// Execute performs flag parsing, input validation and the Git clone
func Execute(ctx context.Context) error {
	flagValues = settings{}
	pflag.Parse()

	err := runGitClone(ctx)
	if err != nil {
		ctxlog.Error(ctx, err, "program failed with an error")
	}

	return err
}

func runGitClone(ctx context.Context) error {
	if flagValues.url == "" {
		return &ExitError{Code: 100, Message: "the 'url' argument must not be empty"}
	}

	if flagValues.target == "" {
		return &ExitError{Code: 101, Message: "the 'target' argument must not be empty"}
	}

	if err := clone(ctx); err != nil {
		return err
	}

	if flagValues.resultFileCommitSha != "" {
		output, err := git(ctx, "-C", flagValues.target, "rev-parse", "--verify", "HEAD")
		if err != nil {
			return err
		}

		if err := ioutil.WriteFile(flagValues.resultFileCommitSha, []byte(output), 0644); err != nil {
			return err
		}
	}

	return nil
}

func checkEnvironment(ctx context.Context) error {
	var checks = []struct{ toolName, versionArg string }{
		{toolName: "ssh", versionArg: "-V"},
		{toolName: "git", versionArg: "version"},
		{toolName: "git-lfs", versionArg: "version"},
	}

	for _, check := range checks {
		path, err := exec.LookPath(check.toolName)
		if err != nil {
			return &ExitError{Code: 120, Message: err.Error(), Cause: err}
		}

		out, err := exec.CommandContext(ctx, path, check.versionArg).CombinedOutput()
		if err != nil {
			return err
		}

		ctxlog.Info(ctx, check.toolName,
			"path", path,
			"version", strings.TrimRight(string(out), "\n"),
		)
	}

	return nil
}

func clone(ctx context.Context) error {
	args := []string{
		"clone",
		"--quiet",
		"--no-tags",
	}

	var commitSha string
	switch {
	case commitShaRegEx.MatchString(flagValues.revision):
		commitSha = flagValues.revision
		args = append(args,
			"--no-checkout",
		)

	case flagValues.revision != "":
		args = append(args,
			"--branch", flagValues.revision,
			"--depth", "1",
			"--single-branch",
		)

	default:
		args = append(args,
			"--depth", "1",
			"--single-branch",
		)
	}

	if flagValues.secretPath != "" {
		credType, err := checkCredentials()
		if err != nil {
			return err
		}

		switch credType {
		case typePrivateKey:
			// Since the key provided via a secret can have undesirable file
			// permissions, it will end up failing due to SSH sanity checks.
			// Therefore, create a temporary replacement with the right
			// file permissions.
			data, err := ioutil.ReadFile(filepath.Join(flagValues.secretPath, "ssh-privatekey"))
			if err != nil {
				return err
			}

			sshPrivateKeyFile, err := ioutil.TempFile(os.TempDir(), "ssh-private-key")
			if err != nil {
				return err
			}

			defer os.Remove(sshPrivateKeyFile.Name())

			if err := ioutil.WriteFile(sshPrivateKeyFile.Name(), data, 0400); err != nil {
				return err
			}

			var sshCmd = []string{"ssh",
				"-o", "BatchMode=yes",
				"-i", sshPrivateKeyFile.Name(),
			}

			var knownHostsFile = filepath.Join(flagValues.secretPath, "known_hosts")
			if hasFile(knownHostsFile) {
				sshCmd = append(sshCmd,
					"-o", "GlobalKnownHostsFile=/dev/null",
					"-o", fmt.Sprintf("UserKnownHostsFile=%s", knownHostsFile),
				)
			} else {
				sshCmd = append(sshCmd,
					"-o", "StrictHostKeyChecking=accept-new",
				)
			}

			args = append(args,
				"--config",
				fmt.Sprintf(`core.sshCommand=%s`, strings.Join(sshCmd, " ")),
			)

		case typeUsernamePassword:
			repoURL, err := url.Parse(flagValues.url)
			if err != nil {
				return err
			}

			username, err := ioutil.ReadFile(filepath.Join(flagValues.secretPath, "username"))
			if err != nil {
				return err
			}

			password, err := ioutil.ReadFile(filepath.Join(flagValues.secretPath, "password"))
			if err != nil {
				return err
			}

			repoURL.User = url.UserPassword(string(username), string(password))

			credHelperFile, err := ioutil.TempFile(os.TempDir(), "cred-helper-file")
			if err != nil {
				return err
			}

			defer os.Remove(credHelperFile.Name())

			if err := ioutil.WriteFile(credHelperFile.Name(), []byte(repoURL.String()), 0400); err != nil {
				return err
			}

			if _, err := git(ctx, "config", "--global", "credential.helper", fmt.Sprintf("store --file %s", credHelperFile.Name())); err != nil {
				return err
			}
		}
	}

	if _, err := git(ctx, append(args, "--", flagValues.url, flagValues.target)...); err != nil {
		return err
	}

	if commitSha != "" {
		_, err := git(ctx, "-C", flagValues.target, "checkout", commitSha)
		if err != nil {
			return err
		}
	}

	_, err := git(ctx, "-C", flagValues.target, "submodule", "update", "--init", "--recursive")
	return err
}

func git(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	ctxlog.Debug(ctx, cmd.String())

	// Make sure that the spawned process does not try to prompt for infos
	os.Setenv("GIT_TERMINAL_PROMPT", "0")
	cmd.Stdin = nil

	out, err := cmd.CombinedOutput()

	var output string
	if out != nil {
		output = strings.TrimRight(string(out), "\n")
	}

	if err != nil {
		// In case the command fails, it is very likely to be an ExitCode error
		// which contains the exit code of the command. Create a custom exit
		// error where the command output is placed into the error to be more
		// readable to the end-user.
		switch terr := err.(type) {
		case *exec.ExitError:
			err = &ExitError{
				Code:    terr.ExitCode(),
				Message: output,
				Cause:   err,
			}
		}

		ctxlog.Error(ctx, err, "git command failed",
			"command", cmd.String(),
			"output", output,
		)
	}

	return output, err
}

func hasFile(elem ...string) bool {
	_, err := os.Stat(filepath.Join(elem...))
	return !os.IsNotExist(err)
}

func checkCredentials() (credentialType, error) {
	// Checking whether mounted secret is of type `kubernetes.io/ssh-auth`
	// in which case there is a file called ssh-privatekey
	hasPrivateKey := hasFile(flagValues.secretPath, "ssh-privatekey")
	isSSHGitURL := sshGitURLRegEx.MatchString(flagValues.url)
	switch {
	case hasPrivateKey && isSSHGitURL:
		return typePrivateKey, nil

	case hasPrivateKey && !isSSHGitURL:
		return typeUndef, &ExitError{Code: 110, Message: "Credential/URL inconsistency: SSH credentials provided, but URL is not a SSH Git URL"}

	case !hasPrivateKey && isSSHGitURL:
		return typeUndef, &ExitError{Code: 110, Message: "Credential/URL inconsistency: No SSH credentials provided, but URL is a SSH Git URL"}
	}

	// Checking whether mounted secret is of type `kubernetes.io/basic-auth`
	// in which case there need to be the files username and password
	hasUsername := hasFile(flagValues.secretPath, "username")
	hasPassword := hasFile(flagValues.secretPath, "password")
	isHTTPSURL := strings.HasPrefix(flagValues.url, "https")
	switch {
	case hasUsername && hasPassword && isHTTPSURL:
		return typeUsernamePassword, nil

	case hasUsername && !hasPassword || !hasUsername && hasPassword:
		return typeUndef, &ExitError{Code: 110, Message: "Basic Auth incomplete: Both username and password need to be configured"}

	}

	return typeUndef, &ExitError{Code: 110, Message: "Unsupported type of credentials provided, either SSH private key or username/password is supported"}
}
