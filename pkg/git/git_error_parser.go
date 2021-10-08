package git

import (
	"bufio"
	"errors"
	"regexp"
	"strings"
)

type ErrorClass int
type Prefix int

const (
	General Prefix = iota
	Remote
	Warning
	Fatal
	Error
)

const (
	Unknown ErrorClass = iota
	AuthInvalidUserOrPass
	AuthInvalidKey
	BranchNotFound
	RepositoryNotFound
	AuthPrompted // caused when repo not found or private repo access with wrong/missing auth
)

var mapClassToMessage = map[ErrorClass]string{
	RepositoryNotFound:    "The source repository does not exist or you have insufficient rights.",
	AuthInvalidUserOrPass: "Basic authentication has failed. Check your username or password. Note: Github requires a personal access token instead of your regular password.",
	AuthPrompted:          "The source repository does not exist or you have insufficient right. Provide authentication for more details.",
	BranchNotFound:        "The remote branch does not exist. Check your revision argument",
	AuthInvalidKey:        "The key is invalid for the specified target. Please make sure that the remote source exists, you have sufficient rights and the key is in the right format.",
}

type RawToken struct {
	raw string
}

type PrefixToken struct {
	scope Prefix
	RawToken
}

type ErrorClassToken struct {
	class ErrorClass
	RawToken
}

type ErrorToken struct {
	Prefix PrefixToken
	Error  ErrorClassToken
}

type ErrorWithCode struct {
	ErrorCode    int
	ErrorMessage string
	Error        error
}

type ErrorResult struct {
	Message string
	Reason  ErrorClass
}

func (rawToken RawToken) String() string {
	return rawToken.raw
}

func (class ErrorClass) String() string {
	switch class {
	case AuthInvalidUserOrPass:
		return "git-auth-basic"
	// either insufficient access rights or invalid key format
	case AuthInvalidKey:
		return "git-auth-ssh"
	case BranchNotFound:
		return "git-remote-revision"
	case RepositoryNotFound:
		return "git-remote-repository"
	// https based git operations against non-existing or private repo without authentication
	case AuthPrompted:
		return "git-remote-private"
	default:
		return "git-error"
	}
}

func (token ErrorToken) String() string {
	return token.Prefix.String() + ": " + token.Error.String()
}

func parse(message string) (tokenList []ErrorToken) {
	reader := strings.NewReader(message)
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		if token, err := parseLine(scanner.Text()); err == nil {
			tokenList = append(tokenList, *token)
		}
	}

	return tokenList
}

func parseLine(line string) (*ErrorToken, error) {
	var prefixBuilder strings.Builder
	var errorMessage string

	for i, char := range line {
		if char == ':' {
			errorMessage = line[i+1:]
			break
		}
		prefixBuilder.WriteRune(char)
	}

	if len(strings.TrimSpace(errorMessage)) == 0 {
		return nil, errors.New("not in the right format of 'prefix: message'")
	}

	prefixToken := parsePrefix(prefixBuilder.String())
	errorClassToken := parseErrorMessage(errorMessage)

	return &ErrorToken{Error: errorClassToken, Prefix: prefixToken}, nil
}

func parsePrefix(raw string) PrefixToken {
	var prefix = General

	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "fatal":
		prefix = Fatal
	case "warning":
		prefix = Warning
	case "remote":
		prefix = Remote
	case "error":
		prefix = Error
	}

	return PrefixToken{prefix, RawToken{raw}}
}

func parseErrorMessage(raw string) ErrorClassToken {
	var errorClass = Unknown
	toCheck := strings.ToLower(strings.TrimSpace(raw))

	// basic auth failed for given creds
	if strings.Contains(toCheck, "authentication failed for") || strings.Contains(toCheck, "invalid username or password") {
		errorClass = AuthInvalidUserOrPass
	} else if strings.Contains(toCheck, "terminal prompts disabled") {
		errorClass = AuthPrompted
	} else if strings.Contains(toCheck, "could not read from remote") {
		errorClass = AuthInvalidKey
	} else if isMatch, _ := regexp.Match("(repository|project) .* found", []byte(toCheck)); isMatch {
		errorClass = RepositoryNotFound
	} else if strings.Contains(toCheck, "remote branch") && strings.Contains(toCheck, "not found") {
		errorClass = BranchNotFound
	}
	return ErrorClassToken{errorClass, RawToken{
		raw,
	}}
}

func classifyErrorFromTokens(tokens []ErrorToken) ErrorClass {
	classifierMap := map[Prefix][]ErrorToken{}
	for _, token := range tokens {
		classifierMap[token.Prefix.scope] = append(classifierMap[token.Prefix.scope], token)
	}

	for _, remoteToken := range classifierMap[Remote] {
		switch remoteToken.Error.class {
		case AuthInvalidUserOrPass:
			return AuthInvalidUserOrPass
		case RepositoryNotFound:
			return RepositoryNotFound
		}
	}

	for _, remoteToken := range classifierMap[Error] {
		switch remoteToken.Error.class {
		case RepositoryNotFound:
			return RepositoryNotFound
		}
	}

	for _, fatalToken := range classifierMap[Fatal] {
		switch fatalToken.Error.class {
		case AuthInvalidKey:
			// either repo no exists or wrong key used
			return AuthInvalidKey
		case AuthPrompted:
			// similar to invalid key, no rights for repo or non-existing repo
			return AuthPrompted
		case BranchNotFound:
			return BranchNotFound
		case AuthInvalidUserOrPass:
			return AuthInvalidUserOrPass
		}
	}

	return Unknown
}

func extractResultsFromTokens(tokens []ErrorToken) *ErrorResult {
	mainErrorClass := classifyErrorFromTokens(tokens)

	if mainErrorClass == Unknown {
		builder := strings.Builder{}
		for _, token := range tokens {
			builder.WriteString(token.String() + "\n")
		}
		return &ErrorResult{Message: builder.String(), Reason: Unknown}
	}

	return &ErrorResult{Message: mapClassToMessage[mainErrorClass], Reason: mainErrorClass}
}

// NewErrorResultFromMessage parses a message, derives an error result and returns an instance of ErrorResult
func NewErrorResultFromMessage(message string) *ErrorResult {
	return extractResultsFromTokens(parse(message))
}
