// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package git

import (
	"bufio"
	"errors"
	"regexp"
	"strings"
)

type (
	// ErrorClass classifies git stdout error messages in broader categories
	ErrorClass int
	// Prefix is part of an error message output and can be used to determine which participant in the git protocol
	// send parts of the error message
	Prefix int
)

const (
	unknownPrefix Prefix = iota
	remotePrefix
	fatalPrefix
	errorPrefix
)

const (
	// Unknown is the class of choice if no other class fits.
	Unknown ErrorClass = iota
	// AuthInvalidUserOrPass expresses that basic authentication is not possible.
	AuthInvalidUserOrPass
	// AuthExpectedSSH expresses that the ssh protocol is used for git operations but basic auth was provided.
	AuthExpectedSSH
	// AuthUnexpectedSSH expresses that the https protocol is used for git operations but a ssh key was provided.
	AuthUnexpectedSSH
	// AuthBasicIncomplete expresses that either username or password is missing in basic auth credentials
	AuthBasicIncomplete
	// AuthUnexpectedHTTP expresses that basic auth username and password are used in combination with a HTTP endpoint
	AuthUnexpectedHTTP
	// AuthInvalidKey expresses that ssh authentication is not possible
	AuthInvalidKey
	// RevisionNotFound expresses that a remote branch does not exist.
	RevisionNotFound
	// RepositoryNotFound expresses that the remote target for the git operation does not exist. It triggers when an
	// error message is enough to determine that the remote target does not exist and is mostly derived from the
	// Git server's messages e.g. GitLab or GitHub.
	RepositoryNotFound
	// AuthPrompted is caused when a repo is not found, is private and authentication is insufficient
	AuthPrompted
)

type rawToken struct {
	raw string
}

type prefixToken struct {
	scope Prefix
	rawToken
}

type errorClassToken struct {
	class ErrorClass
	rawToken
}

type errorToken struct {
	prefixToken prefixToken
	classToken  errorClassToken
}

// ErrorResult is a representation of a runtime error of a git operation that presents a reason and a message
type ErrorResult struct {
	Message string
	Reason  ErrorClass
}

func (rawToken rawToken) String() string {
	return rawToken.raw
}

func (class ErrorClass) String() string {
	switch class {
	case AuthInvalidUserOrPass:
		return "GitAuthInvalidUserOrPass"
	case AuthInvalidKey:
		return "GitAuthInvalidKey"
	case RevisionNotFound:
		return "GitRevisionNotFound"
	case RepositoryNotFound:
		return "GitRemoteRepositoryNotFound"
	case AuthPrompted:
		return "GitRemoteRepositoryPrivate"
	case AuthBasicIncomplete:
		return "GitBasicAuthIncomplete"
	case AuthUnexpectedSSH:
		return "GitSSHAuthUnexpected"
	case AuthExpectedSSH:
		return "GitSSHAuthExpected"
	case AuthUnexpectedHTTP:
		return "AuthUnexpectedHTTP"
	}

	return "GitError"
}

// ToMessage is a function that transforms an error class to an error message
func (class ErrorClass) ToMessage() string {
	switch class {
	case RepositoryNotFound:
		return "The source repository does not exist, or you have insufficient permission to access it."
	case AuthInvalidUserOrPass:
		return "Basic authentication has failed. Check your username or password. Note: GitHub requires a personal access token instead of your regular password."
	case AuthPrompted:
		return "The source repository does not exist, or you have insufficient permission to access it."
	case RevisionNotFound:
		return "The remote revision does not exist. Check your revision argument."
	case AuthInvalidKey:
		return "The key is invalid for the specified target. Please make sure that the Git repository exists, you have sufficient permissions, and the key is in the right format."
	case AuthUnexpectedSSH:
		return "Credential/URL inconsistency: SSH credentials provided, but URL is not a SSH Git URL."
	case AuthExpectedSSH:
		return "Credential/URL inconsistency: No SSH credentials provided, but URL is a SSH Git URL."
	case AuthBasicIncomplete:
		return "Basic Auth incomplete: Both username and password need to be configured."
	case AuthUnexpectedHTTP:
		return "Refusing to continue with basic authentication (username and password) over insecure HTTP connection"
	}

	return "Git encountered an unknown error."
}

func (token errorToken) String() string {
	return token.prefixToken.String() + ": " + token.classToken.String()
}

func parse(message string) (tokenList []errorToken) {
	reader := strings.NewReader(message)
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		if token, err := parseLine(scanner.Text()); err == nil {
			tokenList = append(tokenList, *token)
		}
	}

	return tokenList
}

var errWrongFormat = errors.New("not in the right format of 'prefix:message'")

func parseLine(line string) (*errorToken, error) {
	var (
		prefixBuilder strings.Builder
		errorMessage  string
	)

	for i, char := range line {
		if char == ':' {
			errorMessage = line[i+1:]

			break
		}

		prefixBuilder.WriteRune(char)
	}

	if len(strings.TrimSpace(errorMessage)) == 0 {
		return nil, errWrongFormat
	}

	return &errorToken{
		classToken:  parseErrorMessage(errorMessage),
		prefixToken: parsePrefix(prefixBuilder.String()),
	}, nil
}

func parsePrefix(raw string) prefixToken {
	prefix := unknownPrefix

	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "fatal":
		prefix = fatalPrefix
	case "remote":
		prefix = remotePrefix
	case "error":
		prefix = errorPrefix
	}

	return prefixToken{prefix, rawToken{raw}}
}

func isAuthInvalidUserOrPass(raw string) bool {
	return strings.Contains(raw, "authentication failed for") ||
		strings.Contains(raw, "invalid username or password")
}

func isAuthPrompted(raw string) bool {
	return strings.Contains(raw, "terminal prompts disabled")
}

func isAuthInvalidKey(raw string) bool {
	return strings.Contains(raw, "could not read from remote")
}

func isRepositoryNotFound(raw string) bool {
	isMatch, _ := regexp.Match("(repository|project) .* found", []byte(raw))

	return isMatch
}

func isBranchNotFound(raw string) bool {
	return strings.Contains(raw, "remote branch") && strings.Contains(raw, "not found")
}

func parseErrorMessage(raw string) errorClassToken {
	errorClass := Unknown
	toCheck := strings.ToLower(strings.TrimSpace(raw))

	switch {
	case isAuthInvalidUserOrPass(toCheck):
		errorClass = AuthInvalidUserOrPass
	case isAuthPrompted(toCheck):
		errorClass = AuthPrompted
	case isAuthInvalidKey(toCheck):
		errorClass = AuthInvalidKey
	case isRepositoryNotFound(toCheck):
		errorClass = RepositoryNotFound
	case isBranchNotFound(toCheck):
		errorClass = RevisionNotFound
	}

	return errorClassToken{errorClass, rawToken{
		raw,
	}}
}

func classifyTokensWithRemotePrefix(tokens []errorToken) ErrorClass {
	for _, remoteToken := range tokens {
		switch remoteToken.classToken.class {
		case AuthInvalidUserOrPass:
			return AuthInvalidUserOrPass
		case RepositoryNotFound:
			return RepositoryNotFound
		}
	}

	return Unknown
}

func classifyTokensWithErrorPrefix(tokens []errorToken) ErrorClass {
	for _, remoteToken := range tokens {
		if remoteToken.classToken.class == RepositoryNotFound {
			return RepositoryNotFound
		}
	}

	return Unknown
}

func classifyTokensWithFatalPrefix(tokens []errorToken) ErrorClass {
	for _, fatalToken := range tokens {
		switch fatalToken.classToken.class {
		case AuthInvalidKey:
			return AuthInvalidKey
		case AuthPrompted:
			return AuthPrompted
		case RevisionNotFound:
			return RevisionNotFound
		case AuthInvalidUserOrPass:
			return AuthInvalidUserOrPass
		}
	}

	return Unknown
}

func classifyErrorFromTokens(tokens []errorToken) ErrorClass {
	classifierMap := map[Prefix][]errorToken{}
	for _, token := range tokens {
		classifierMap[token.prefixToken.scope] = append(classifierMap[token.prefixToken.scope], token)
	}

	if errorClass := classifyTokensWithRemotePrefix(classifierMap[remotePrefix]); errorClass != Unknown {
		return errorClass
	}

	if errorClass := classifyTokensWithErrorPrefix(classifierMap[errorPrefix]); errorClass != Unknown {
		return errorClass
	}

	if errorClass := classifyTokensWithFatalPrefix(classifierMap[fatalPrefix]); errorClass != Unknown {
		return errorClass
	}

	return Unknown
}

func extractResultsFromTokens(tokens []errorToken) *ErrorResult {
	mainErrorClass := classifyErrorFromTokens(tokens)

	if mainErrorClass == Unknown {
		builder := strings.Builder{}
		for _, token := range tokens {
			builder.WriteString(token.String() + "\n")
		}

		return &ErrorResult{Message: builder.String(), Reason: Unknown}
	}

	return &ErrorResult{Message: mainErrorClass.ToMessage(), Reason: mainErrorClass}
}

// NewErrorResultFromMessage parses a message, derives an error result and returns an instance of ErrorResult.
func NewErrorResultFromMessage(message string) *ErrorResult {
	return extractResultsFromTokens(parse(message))
}
