package keycloak

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ajg/form"
	"github.com/fabric8-services/fabric8-auth/errors"
	"github.com/fabric8-services/fabric8-auth/log"
	"github.com/fabric8-services/fabric8-auth/rest"
	errs "github.com/pkg/errors"
)

// KeycloakExternalTokenResponse represents standard Keycloak external token response payload
type KeycloakExternalTokenResponse struct {
	AccessToken string `json:"access_token,omitempty" form:"access_token,omniempty"`
	Scope       string `json:"scope,omitempty" form:"scope,omniempty"`
	TokenType   string `json:"token_type,omitempty" form:"token_type,omniempty"`
	ExpiresIn   int64  `json:"expires_in,omitempty" form:"expires_in,omniempty"`
}

// KeycloakExternalTokenService describes what the service can do with respect to external tokens from Keycloak.
type KeycloakExternalTokenService interface {
	Get(ctx context.Context, accessToken string, keycloakExternalTokenURL string) (*KeycloakExternalTokenResponse, error)
}

// KeycloakExternalTokenServiceClient is an implementation of KeycloakExternalTokenService and serves as an interface to the Keycloak token service.
type KeycloakExternalTokenServiceClient struct {
	client *http.Client
}

// NewKeycloakTokenServiceClient creates a new KeycloakTokenServiceClient
func NewKeycloakTokenServiceClient() KeycloakExternalTokenServiceClient {
	return KeycloakExternalTokenServiceClient{
		client: http.DefaultClient,
	}
}

//Get gets the external token information from Keycloak
func (keycloakExternalTokenServiceClient *KeycloakExternalTokenServiceClient) Get(ctx context.Context, accessToken string, keycloakExternalTokenURL string) (*KeycloakExternalTokenResponse, error) {

	log.Info(ctx, map[string]interface{}{
		"keycloak_external_token_url": keycloakExternalTokenURL,
	}, "fetching token..")

	keycloakExternalTokenResponse := KeycloakExternalTokenResponse{}

	req, err := http.NewRequest("GET", keycloakExternalTokenURL, nil)
	if err != nil {
		return nil, errors.NewInternalError(ctx, err)
	}
	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("Accept", "application/json, text/plain, */*")

	resp, err := keycloakExternalTokenServiceClient.client.Do(req)

	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"keycloak_external_token_url": keycloakExternalTokenURL,
			"err": err,
		}, "Unable to fetch external keycloak token")
		return nil, errors.NewInternalError(ctx, err)
	} else if resp != nil {
		defer resp.Body.Close()
	}
	if resp.StatusCode != http.StatusOK {
		log.Error(ctx, map[string]interface{}{
			"response_status":             resp.Status,
			"response_body":               rest.ReadBody(resp.Body),
			"keycloak_external_token_url": keycloakExternalTokenURL,
		}, "Unable to fetch external keycloak token")
		if resp.StatusCode == 400 {
			return nil, errors.NewUnauthorizedError(rest.ReadBody(resp.Body))
		}
		return nil, errors.NewInternalError(ctx, errs.Errorf("received a non-200 response %s while fetching keycloak external token %s", resp.Status, keycloakExternalTokenURL))
	}
	if strings.Contains(keycloakExternalTokenURL, "openshift-v3") {
		err = json.NewDecoder(resp.Body).Decode(&keycloakExternalTokenResponse)
	} else {

		// The format for github response is
		// access_token=f75c6_token_ceea0&scope=admin%3Arepo_hook%2Cgist%2Cread%3Aorg%2Crepo%2Cuser&token_type=bearer
		d := form.NewDecoder(resp.Body)
		if err := d.Decode(&keycloakExternalTokenResponse); err != nil {
			return nil, err
		}
	}
	return &keycloakExternalTokenResponse, err
}
