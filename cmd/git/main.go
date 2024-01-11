// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	shpgit "github.com/shipwright-io/build/pkg/git"
	"github.com/spf13/pflag"
)

type credentialType int

const (
	typeUndef credentialType = iota
	typePrivateKey
	typeUsernamePassword
)

var useNoTagsFlag = false
var useDepthForSubmodule = false

var displayURL string

// ExitError is an error which has an exit code to be used in os.Exit() to
// return both an exit code and an error message
type ExitError struct {
	Code    int
	Message string
	Cause   error
	Reason  shpgit.ErrorClass
}

func (e ExitError) Error() string {
	return fmt.Sprintf("%s (exit code %d)", e.Message, e.Code)
}

type settings struct {
	help                      bool
	url                       string
	revision                  string
	depth                     uint
	target                    string
	resultFileCommitSha       string
	resultFileCommitAuthor    string
	resultFileBranchName      string
	resultFileSourceTimestamp string
	secretPath                string
	skipValidation            bool
	gitURLRewrite             bool
	resultFileErrorMessage    string
	resultFileErrorReason     string
	verbose                   bool
}

var flagValues settings

var (
	sshGitURLRegEx = regexp.MustCompile(`^(git@|ssh:\/\/).+$`)
	commitShaRegEx = regexp.MustCompile(`^[0-9a-f]{7,40}$`)
)

func init() {
	// Explicitly define the help flag so that --help can be invoked and returns status code 0
	pflag.BoolVar(&flagValues.help, "help", false, "Print the help")

	// Main flags for the Git step to define the configuration, for example
	// the flags for `url`, and `target` will always be used, but `revision`
	// depends on the respective use case.
	pflag.StringVar(&flagValues.url, "url", "", "The URL of the Git repository")
	pflag.StringVar(&flagValues.revision, "revision", "", "The revision of the Git repository to be cloned. Optional, defaults to the default branch.")
	pflag.StringVar(&flagValues.target, "target", "", "The target directory of the clone operation")
	pflag.StringVar(&flagValues.resultFileCommitSha, "result-file-commit-sha", "", "A file to write the commit sha to.")
	pflag.StringVar(&flagValues.resultFileCommitAuthor, "result-file-commit-author", "", "A file to write the commit author to.")
	pflag.StringVar(&flagValues.resultFileSourceTimestamp, "result-file-source-timestamp", "", "A file to write the source timestamp to.")
	pflag.StringVar(&flagValues.resultFileBranchName, "result-file-branch-name", "", "A file to write the branch name to.")
	pflag.StringVar(&flagValues.secretPath, "secret-path", "", "A directory that contains a secret. Either username and password for basic authentication. Or a SSH private key and optionally a known hosts file. Optional.")

	// Flags with paths for writing error related information
	pflag.StringVar(&flagValues.resultFileErrorMessage, "result-file-error-message", "", "A file to write the error message to.")
	pflag.StringVar(&flagValues.resultFileErrorReason, "result-file-error-reason", "", "A file to write the error reason to.")

	// Optional flag to be able to override the default shallow clone depth,
	// which should be fine for almost all use cases we use the Git source step
	// for (in the context of Shipwright build).
	pflag.UintVar(&flagValues.depth, "depth", 1, "Create a shallow clone based on the given depth")

	// Mostly internal flag
	pflag.BoolVar(&flagValues.skipValidation, "skip-validation", false, "skip pre-requisite validation")
	pflag.BoolVar(&flagValues.gitURLRewrite, "git-url-rewrite", false, "set Git config to use url-insteadOf setting based on Git repository URL")
	pflag.BoolVar(&flagValues.verbose, "verbose", false, "Verbose logging")
}

func main() {
	if err := Execute(context.Background()); err != nil {
		var exitcode = 1
		switch err := err.(type) {
		case *ExitError:
			exitcode = err.Code
		}

		if writeErr := writeErrorResults(shpgit.NewErrorResultFromMessage(err.Error())); writeErr != nil {
			log.Printf("Could not write error results: %s", writeErr.Error())
		}

		log.Print(err.Error())
		os.Exit(exitcode)
	}
}

// Execute performs flag parsing, input validation and the Git clone
func Execute(ctx context.Context) error {
	flagValues = settings{depth: 1}
	pflag.Parse()

	if flagValues.help {
		pflag.Usage()
		return nil
	}

	// pre-req checks
	if err := checkEnvironment(ctx); err != nil {
		return err
	}

	// Check if Git CLI supports --no-tags for clone
	out, _ := git(ctx, "clone", "-h")
	useNoTagsFlag = strings.Contains(out, "--no-tags")

	// Check if Git CLI support --single-branch and therefore shallow clones using --depth
	out, _ = git(ctx, "submodule", "-h")
	useDepthForSubmodule = strings.Contains(out, "single-branch")

	// Create clean version of the URL that should be safe to be displayed in logs
	displayURL = cleanURL()

	return runGitClone(ctx)
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

		if err := os.WriteFile(flagValues.resultFileCommitSha, []byte(output), 0644); err != nil {
			return err
		}
	}

	if flagValues.resultFileCommitAuthor != "" {
		output, err := git(ctx, "-C", flagValues.target, "log", "-1", "--pretty=format:%an")
		if err != nil {
			return err
		}

		if err = os.WriteFile(flagValues.resultFileCommitAuthor, []byte(output), 0644); err != nil {
			return err
		}
	}

	if flagValues.resultFileSourceTimestamp != "" {
		output, err := git(ctx, "-C", flagValues.target, "show", "--no-patch", "--format=%ct")
		if err != nil {
			return err
		}

		if err = os.WriteFile(flagValues.resultFileSourceTimestamp, []byte(output), 0644); err != nil {
			return err
		}
	}

	if strings.TrimSpace(flagValues.revision) == "" && strings.TrimSpace(flagValues.resultFileBranchName) != "" {
		output, err := git(ctx, "-C", flagValues.target, "rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			return err
		}

		if err := os.WriteFile(flagValues.resultFileBranchName, []byte(output), 0644); err != nil {
			return err
		}
	}

	return nil
}

func checkEnvironment(ctx context.Context) error {
	if flagValues.skipValidation {
		return nil
	}

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

		if flagValues.verbose {
			log.Printf("Debug: %s %s\n", path, check.versionArg)
		}
		out, err := exec.CommandContext(ctx, path, check.versionArg).CombinedOutput()
		if err != nil {
			log.Printf("Error: %s: %s\n", check.toolName, strings.TrimRight(string(out), "\n"))
			return err
		}

		log.Printf("Info: %s (%s): %s\n",
			check.toolName,
			path,
			strings.TrimRight(string(out), "\n"),
		)
	}

	return nil
}

func clone(ctx context.Context) error {
	cloneArgs := []string{
		"clone",
		"--quiet",
	}

	if useNoTagsFlag {
		cloneArgs = append(cloneArgs, "--no-tags")
	}

	var commitSha string
	switch {
	case commitShaRegEx.MatchString(flagValues.revision):
		commitSha = flagValues.revision
		cloneArgs = append(cloneArgs, "--no-checkout")

	default:
		cloneArgs = append(cloneArgs, "--single-branch")

		if flagValues.revision != "" {
			cloneArgs = append(cloneArgs, "--branch", flagValues.revision)
		}

		if flagValues.depth > 0 {
			cloneArgs = append(cloneArgs, "--depth", fmt.Sprintf("%d", flagValues.depth))
		}
	}

	var addtlGitArgs []string
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
			data, err := os.ReadFile(filepath.Join(flagValues.secretPath, "ssh-privatekey"))
			if err != nil {
				return err
			}

			sshPrivateKeyFile, err := os.CreateTemp(os.TempDir(), "ssh-private-key")
			if err != nil {
				return err
			}

			defer os.Remove(sshPrivateKeyFile.Name())

			if err := os.WriteFile(sshPrivateKeyFile.Name(), data, 0400); err != nil {
				return err
			}

			var sshCmd = []string{"ssh",
				"-o", "LogLevel=ERROR",
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

			addtlGitArgs = append(addtlGitArgs,
				"-c",
				fmt.Sprintf(`core.sshCommand=%s`, strings.Join(sshCmd, " ")),
			)

			// When the Git URL rewrite is enabled, additional Git config
			// options are required to introduce a rewrite rule so that
			// HTTPS URLs are rewritten into Git+SSH URLs on the fly for
			// the main clone as well as the submodule operations. This
			// only makes sense in case a private key is configured.
			if flagValues.gitURLRewrite {
				var hostname string
				switch {
				case strings.HasPrefix(flagValues.url, "git@"):
					trimmed := strings.TrimPrefix(flagValues.url, "git@")
					splitted := strings.SplitN(trimmed, ":", 2)
					hostname = splitted[0]

				case strings.HasPrefix(flagValues.url, "http"):
					repoURL, err := url.Parse(flagValues.url)
					if err != nil {
						return err
					}
					hostname = repoURL.Host

				default:
					log.Printf("Failed to setup Git URL rewrite, unknown/unsupported URL type: %q\n", flagValues.url)
				}

				if hostname != "" {
					addtlGitArgs = append(addtlGitArgs,
						"-c",
						fmt.Sprintf("url.ssh://git@%s/.insteadOf=https://%s/", hostname, hostname),
					)
				}
			}

		case typeUsernamePassword:
			repoURL, err := url.Parse(flagValues.url)
			if err != nil {
				return err
			}

			username, err := os.ReadFile(filepath.Join(flagValues.secretPath, "username"))
			if err != nil {
				return err
			}

			password, err := os.ReadFile(filepath.Join(flagValues.secretPath, "password"))
			if err != nil {
				return err
			}

			repoURL.User = url.UserPassword(string(username), string(password))

			credHelperFile, err := os.CreateTemp(os.TempDir(), "cred-helper-file")
			if err != nil {
				return err
			}

			defer os.Remove(credHelperFile.Name())

			if err := os.WriteFile(credHelperFile.Name(), []byte(repoURL.String()), 0400); err != nil {
				return err
			}

			addtlGitArgs = append(addtlGitArgs,
				"-c",
				fmt.Sprintf("credential.helper=%s", fmt.Sprintf("store --file %s", credHelperFile.Name())),
			)
		}
	}

	cloneArgs = append(cloneArgs, addtlGitArgs...)
	cloneArgs = append(cloneArgs, "--", flagValues.url, flagValues.target)
	if _, err := git(ctx, cloneArgs...); err != nil {
		return err
	}

	if commitSha != "" {
		if _, err := git(ctx, "-C", flagValues.target, "checkout", commitSha); err != nil {
			return err
		}
	}

	submoduleArgs := []string{"-C", flagValues.target}
	submoduleArgs = append(submoduleArgs, addtlGitArgs...)
	submoduleArgs = append(submoduleArgs, "submodule", "update", "--init", "--recursive")
	if useDepthForSubmodule && flagValues.depth > 0 {
		submoduleArgs = append(submoduleArgs, "--depth", fmt.Sprintf("%d", flagValues.depth))
	}

	if _, err := git(ctx, submoduleArgs...); err != nil {
		return err
	}

	revision := flagValues.revision
	if revision == "" {
		// user requested to clone the default branch, determine the branch name
		refParse, err := git(ctx, "-C", flagValues.target, "rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			return err
		}

		revision = strings.TrimRight(refParse, "\n")
	}

	log.Printf("Successfully loaded %s (%s) into %s\n",
		displayURL,
		revision,
		flagValues.target,
	)

	return nil
}

func git(ctx context.Context, args ...string) (string, error) {
	fullArgs := []string{
		"-c",
		fmt.Sprintf("safe.directory=%s", flagValues.target),
	}
	fullArgs = append(fullArgs, args...)
	cmd := exec.CommandContext(ctx, "git", fullArgs...)

	// Print the command to be executed, but replace the URL with a safe version
	log.Print(strings.ReplaceAll(cmd.String(), flagValues.url, displayURL))

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
	isGitURLRewriteSet := flagValues.gitURLRewrite
	switch {
	case hasPrivateKey && isSSHGitURL:
		return typePrivateKey, nil

	case hasPrivateKey && !isSSHGitURL && isGitURLRewriteSet:
		return typePrivateKey, nil

	case hasPrivateKey && !isSSHGitURL:
		return typeUndef, &ExitError{
			Code:    110,
			Message: shpgit.AuthUnexpectedSSH.ToMessage(),
			Reason:  shpgit.AuthUnexpectedSSH,
		}

	case !hasPrivateKey && isSSHGitURL:
		return typeUndef, &ExitError{
			Code:    110,
			Message: shpgit.AuthExpectedSSH.ToMessage(),
			Reason:  shpgit.AuthExpectedSSH,
		}
	}

	// Checking whether mounted secret is of type `kubernetes.io/basic-auth`
	// in which case there need to be the files username and password
	hasUsername := hasFile(flagValues.secretPath, "username")
	hasPassword := hasFile(flagValues.secretPath, "password")
	switch {
	case hasUsername && hasPassword && strings.HasPrefix(flagValues.url, "https://"):
		return typeUsernamePassword, nil

	case hasUsername && hasPassword && strings.HasPrefix(flagValues.url, "http://"):
		return typeUndef, &ExitError{
			Code:    110,
			Message: shpgit.AuthUnexpectedHTTP.ToMessage(),
			Reason:  shpgit.AuthUnexpectedHTTP,
		}

	case hasUsername && !hasPassword || !hasUsername && hasPassword:
		return typeUndef, &ExitError{
			Code:    110,
			Message: shpgit.AuthBasicIncomplete.ToMessage(),
			Reason:  shpgit.AuthBasicIncomplete,
		}
	}

	return typeUndef, &ExitError{
		Code:    110,
		Message: "Unsupported type of credentials provided, either SSH private key or username/password is supported",
		Reason:  shpgit.Unknown,
	}
}

func writeErrorResults(failure *shpgit.ErrorResult) (err error) {
	if flagValues.resultFileErrorReason == "" || flagValues.resultFileErrorMessage == "" {
		return nil
	}

	messageToWrite := failure.Message
	messageLengthThreshold := 300

	if len(messageToWrite) > messageLengthThreshold {
		messageToWrite = messageToWrite[:messageLengthThreshold-3] + "..."
	}

	if err = os.WriteFile(flagValues.resultFileErrorMessage, []byte(strings.TrimSpace(messageToWrite)), 0666); err != nil {
		return err
	}

	return os.WriteFile(flagValues.resultFileErrorReason, []byte(failure.Reason.String()), 0666)
}

func cleanURL() string {
	// non HTTP/HTTPS URLs are returned as-is (i.e. Git+SSH URLs)
	if !strings.HasPrefix(flagValues.url, "http") {
		return flagValues.url
	}

	// return redacted version of the URL if it is a parsable URL
	if repoURL, err := url.Parse(flagValues.url); err == nil {
		if repoURL.User != nil {
			log.Println("URL has inline credentials, which need to be redacted for log out. If possible, use an alternative approach.")
		}

		return repoURL.Redacted()
	}

	// in any case, as a fallback, return it as-is
	return flagValues.url
}
