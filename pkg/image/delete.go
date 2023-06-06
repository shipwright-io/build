// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// Delete removes the image from the container registry
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
//   - In case the repository only has one tag, the repository is deleted.
//   - If there are multiple tags, the tag to be deleted is overwritten
//     with an empty image (to remove the content, and save quota).
//   - Edge case would be no tags in the repository, which is ignored.
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
func Delete(ref name.Reference, options []remote.Option, auth authn.AuthConfig) error {
	switch {
	case strings.Contains(ref.Context().RegistryStr(), "docker.io"):
		list, err := remote.List(ref.Context(), options...)
		if err != nil {
			return err
		}

		switch len(list) {
		case 0:
			return nil

		case 1:
			var token string
			token, err = dockerHubLogin(auth.Username, auth.Password)
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
				options...,
			)
		}

	case strings.Contains(ref.Context().RegistryStr(), "icr.io"):
		token, accountID, err := icrLogin(ref.Context().RegistryStr(), auth.Username, auth.Password)
		if err != nil {
			return err
		}

		return icrDelete(token, accountID, ref)

	default:
		return remote.Delete(
			ref,
			options...,
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
