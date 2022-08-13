// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/cli/cli/config"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/spf13/pflag"

	"github.com/shipwright-io/build/pkg/bundle"
)

type settings struct {
	help                  bool
	image                 string
	prune                 bool
	target                string
	secretPath            string
	resultFileImageDigest string
}

var flagValues settings

func init() {
	// Explicitly define the help flag so that --help can be invoked and returns status code 0
	pflag.BoolVar(&flagValues.help, "help", false, "Print the help")

	// Main flags of the bundle step
	pflag.StringVar(&flagValues.image, "image", "", "Location of the bundle image (mandatory)")
	pflag.StringVar(&flagValues.target, "target", "/workspace/source", "The target directory to place the code")
	pflag.StringVar(&flagValues.resultFileImageDigest, "result-file-image-digest", "", "A file to write the image digest")

	pflag.StringVar(&flagValues.secretPath, "secret-path", "", "A directory that contains access credentials (optional)")
	pflag.BoolVar(&flagValues.prune, "prune", false, "Delete bundle image from registry after it was pulled")
}

func main() {
	if err := Do(context.Background()); err != nil {
		log.Fatal(err.Error())
	}
}

// Do is the main entry point of the bundle command
func Do(ctx context.Context) error {
	flagValues = settings{}
	pflag.Parse()

	if flagValues.help {
		pflag.Usage()
		return nil
	}

	if flagValues.image == "" {
		return fmt.Errorf("mandatory flag --image is not set")
	}

	ref, err := name.ParseReference(flagValues.image)
	if err != nil {
		return err
	}

	auth, err := resolveAuthBasedOnTarget(ref)
	if err != nil {
		return err
	}

	log.Printf("Pulling image %q", ref)
	img, err := bundle.PullAndUnpack(
		ref,
		flagValues.target,
		remote.WithContext(ctx),
		remote.WithAuth(auth))
	if err != nil {
		return err
	}

	log.Printf("Image content was extracted to %s\n", flagValues.target)

	digest, err := img.Digest()
	if err != nil {
		return fmt.Errorf("failed to retrieve digest from bundle image: %w", err)
	}

	if flagValues.resultFileImageDigest != "" {
		if err = os.WriteFile(flagValues.resultFileImageDigest, []byte(digest.String()), 0644); err != nil {
			return err
		}
	}

	if flagValues.prune {
		// Some container registry implementations, i.e. library/registry:2 will fail to
		// delete the image when there is no image digest given. Use image digest from the
		// image pulling to construct an image name including tag and digest.
		ref, err = name.NewDigest(fmt.Sprintf("%s@%s", ref.Name(), digest.String()))
		if err != nil {
			return err
		}

		log.Printf("Deleting image %q", ref)
		if err := Prune(ctx, ref, auth); err != nil {
			return err
		}
	}

	return nil
}

func resolveAuthBasedOnTarget(ref name.Reference) (authn.Authenticator, error) {
	// In case no secret is mounted, use anonymous
	if flagValues.secretPath == "" {
		log.Printf("No access credentials provided, using anonymous mode")
		return authn.Anonymous, nil
	}

	// Read the registry credentials from the well-known location if it exists
	var mountedSecretDefaultFileName = filepath.Join(flagValues.secretPath, ".dockerconfigjson")
	if _, err := os.Stat(mountedSecretDefaultFileName); err == nil {
		return ResolveAuthBasedOnTargetUsingConfigFile(ref, mountedSecretDefaultFileName)
	}

	// Otherwise, treat secret path as a file (for none Kubernetes setups)
	return ResolveAuthBasedOnTargetUsingConfigFile(ref, flagValues.secretPath)
}

// ResolveAuthBasedOnTargetUsingConfigFile resolves if possible the respective authenticator to be used for
// given image reference (full registry and image name)
func ResolveAuthBasedOnTargetUsingConfigFile(ref name.Reference, dockerConfigFile string) (authn.Authenticator, error) {
	file, err := os.Open(dockerConfigFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cf, err := config.LoadFromReader(file)
	if err != nil {
		return nil, err
	}

	// Look-up the respective registry server inside the credentials
	registryName := ref.Context().RegistryStr()
	if registryName == name.DefaultRegistry {
		registryName = authn.DefaultAuthKey
	}

	authConfig, err := cf.GetAuthConfig(registryName)
	if err != nil {
		return nil, err
	}

	// Return an error in case the credentials do not match the desired
	// registry and list all servers that actually are available
	if authConfig.ServerAddress != registryName {
		var servers []string
		for name := range cf.GetAuthConfigs() {
			servers = append(servers, name)
		}

		var availableConfigs string
		if len(servers) > 0 {
			availableConfigs = strings.Join(servers, ", ")
		} else {
			availableConfigs = "none"
		}

		return nil, fmt.Errorf("failed to find registry credentials for %s, available configurations: %s",
			registryName,
			availableConfigs,
		)
	}

	log.Printf("Using provided access credentials for %s", registryName)
	return authn.FromConfig(authn.AuthConfig{
		Username:      authConfig.Username,
		Password:      authConfig.Password,
		Auth:          authConfig.Auth,
		IdentityToken: authConfig.IdentityToken,
		RegistryToken: authConfig.RegistryToken,
	}), nil
}

// Prune removes the image from the container registry
//
// Deleting a tag, or a whole repo is not as straightforward as initially
// planned as DockerHub seems to restrict deleting a single tag for
// standard users. This might be subject to change, but as of September
// 2021 it is limited to the business tier. However, there is an API call
// to delete the whole repository. In case there is only one tag used in
// a repository, the effect is pretty much the same. For convenience, there
// is a provider switch to deal with images on DockerHub differently.
//
// DockerHub images:
// - In case the repository only has one tag, the repository is deleted.
// - If there are multiple tags, the tag to be deleted is overwritten
//   with an empty image (to remove the content, and save quota).
// - Edge case would be no tags in the repository, which is ignored.
//
// IBM Container Registry images:
// Custom delete API call has to be used, since ICR does not support the
// default registry API for deletions. The credentials need to have an
// IBM API key, which is used to obtain an identity token that needs to
// contains the respective authorization token for requests as well as
// an account identifier to select the IBM account in which the registry
// namespace and image is located.
//
// Other registries:
// Use standard spec delete API request to delete the provided tag.
//
func Prune(ctx context.Context, ref name.Reference, auth authn.Authenticator) error {
	switch {
	case strings.Contains(ref.Context().RegistryStr(), "docker.io"):
		list, err := remote.List(ref.Context(), remote.WithContext(ctx), remote.WithAuth(auth))
		if err != nil {
			return err
		}

		switch len(list) {
		case 0:
			return nil

		case 1:
			var authr *authn.AuthConfig
			authr, err = auth.Authorization()
			if err != nil {
				return err
			}

			var token string
			token, err = dockerHubLogin(authr.Username, authr.Password)
			if err != nil {
				return err
			}

			return dockerHubRepoDelete(token, ref)

		default:
			log.Printf("Removing a specific image tag is not supported on %q, the respective image tag will be overwritten with an empty image.\n", ref.Context().RegistryStr())

			// In case the input argument included a digest, the reference
			// needs to be updated to exclude the digest for the empty image
			// override to succeed.
			switch ref.(type) {
			case name.Digest:
				ref, err = name.NewTag(ref.Context().Name())
				if err != nil {
					return err
				}
			}

			return remote.Write(
				ref,
				empty.Image,
				remote.WithContext(ctx),
				remote.WithAuth(auth),
			)
		}

	case strings.Contains(ref.Context().RegistryStr(), "icr.io"):
		authr, err := auth.Authorization()
		if err != nil {
			return err
		}

		token, accountID, err := icrLogin(ref.Context().RegistryStr(), authr.Username, authr.Password)
		if err != nil {
			return err
		}

		return icrDelete(token, accountID, ref)

	default:
		return remote.Delete(
			ref,
			remote.WithContext(ctx),
			remote.WithAuth(auth),
		)
	}
}

func httpClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
	}
}

func dockerHubLogin(username string, password string) (string, error) {
	type LoginData struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	loginData, err := json.Marshal(LoginData{Username: username, Password: password})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://hub.docker.com/v2/users/login/", bytes.NewReader(loginData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	bodyData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		type LoginToken struct {
			Token string `json:"token"`
		}

		var loginToken LoginToken
		if err := json.Unmarshal(bodyData, &loginToken); err != nil {
			return "", err
		}

		return fmt.Sprintf("JWT %s", loginToken.Token), nil

	default:
		return "", fmt.Errorf(string(bodyData))
	}
}

func dockerHubRepoDelete(token string, ref name.Reference) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("https://hub.docker.com/v2/repositories/%s/", ref.Context().RepositoryStr()), nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", token)

	resp, err := httpClient().Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusAccepted:
		return nil

	default:
		return fmt.Errorf("failed to delete image %q: %s (HTTP status code %d)",
			ref.String(),
			string(respData),
			resp.StatusCode,
		)
	}
}

func icrLogin(registry, username, apikey string) (string, string, error) {
	// IBM Container Registry API calls will only work in case an API key is available
	if username != "iamapikey" {
		return "", "", fmt.Errorf("provided access credentials for %q do not contain an IBM API key", registry)
	}

	iamEndpoint := "https://iam.cloud.ibm.com/identity/token"
	if strings.Contains(registry, "stg.icr.io") {
		iamEndpoint = "https://iam.test.cloud.ibm.com/identity/token"
	}

	data := fmt.Sprintf("grant_type=%s&apikey=%s",
		url.QueryEscape("urn:ibm:params:oauth:grant-type:apikey"),
		apikey,
	)

	req, err := http.NewRequest("POST", iamEndpoint, strings.NewReader(data))
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient().Do(req)
	if err != nil {
		return "", "", err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		type ibmCloudIdentityToken struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			TokenType    string `json:"token_type"`
			Scope        string `json:"scope"`
			ExpiresIn    int64  `json:"expires_in"`
			Expiration   int64  `json:"expiration"`
		}

		var identityToken ibmCloudIdentityToken
		if err := json.Unmarshal(body, &identityToken); err != nil {
			return "", "", err
		}

		var token = fmt.Sprintf("%s %s", identityToken.TokenType, identityToken.AccessToken)

		var accountID string
		_, _ = jwt.Parse(identityToken.AccessToken, func(t *jwt.Token) (interface{}, error) {
			switch obj := t.Claims.(type) {
			case jwt.MapClaims:
				if account, ok := obj["account"]; ok {
					switch accountMap := account.(type) {
					case map[string]interface{}:
						switch tmp := accountMap["bss"].(type) {
						case string:
							accountID = tmp
						}
					}
				}
			}

			return nil, nil
		})

		if accountID == "" {
			return "", "", fmt.Errorf("failed to obtain account ID from identity token")
		}

		return token, accountID, nil

	default:
		var responseMsg map[string]interface{}
		if err := json.Unmarshal(body, &responseMsg); err != nil {
			return "", "", err
		}

		errorCode, errorCodeFound := responseMsg["errorCode"]
		errorMessage, errorMessageFound := responseMsg["errorMessage"]
		if errorCodeFound && errorMessageFound {
			return "", "", fmt.Errorf("failed to obtain identity token from IAM: %v (%v)", errorMessage, errorCode)
		}

		return "", "", fmt.Errorf("failed to obtain identity token from IAM: %s", string(body))
	}
}

func icrDelete(token string, accountID string, ref name.Reference) error {
	deleteURL := fmt.Sprintf("https://%s/api/v1/images/%s",
		ref.Context().RegistryStr(),
		url.QueryEscape(ref.String()),
	)

	req, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Account", accountID)
	req.Header.Set("Authorization", token)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient().Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return nil

	default:
		return fmt.Errorf("failed to delete image %q: %s (HTTP status code %d)",
			ref.String(),
			string(respData),
			resp.StatusCode,
		)
	}
}
