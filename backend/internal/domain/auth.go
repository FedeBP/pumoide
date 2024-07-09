package domain

import (
	"fmt"
	"net/http"

	"github.com/FedeBP/pumoide/backend/pkg/constants"
	"github.com/FedeBP/pumoide/backend/pkg/errors"
)

type Auth struct {
	Type   AuthType          `json:"type"`
	Params map[string]string `json:"params,omitempty"`
	OAuth2 *OAuth2Config     `json:"oauth2,omitempty"`
}

type AuthType string

const (
	AuthNone     AuthType = "none"
	AuthBasic    AuthType = "basic"
	AuthBearer   AuthType = "bearer"
	AuthAPIKey   AuthType = "apiKey"
	AuthOAuth2   AuthType = "oauth2"
	AuthAWSSigV4 AuthType = "awsSigV4"
	AuthDigest   AuthType = "digest"
)

func (a *Auth) Validate() error {
	switch a.Type {
	case AuthNone:
		return nil

	case AuthBasic:
		if _, ok := a.Params[constants.Username]; !ok {
			return errors.NewAppError(http.StatusBadRequest, constants.ErrBasicAuthUser, nil)
		}
		if _, ok := a.Params[constants.Password]; !ok {
			return errors.NewAppError(http.StatusBadRequest, constants.ErrBasicAuthPass, nil)
		}

	case AuthBearer:
		if _, ok := a.Params[constants.Token]; !ok {
			return errors.NewAppError(http.StatusBadRequest, constants.ErrBearerAuth, nil)
		}

	case AuthAPIKey:
		if _, ok := a.Params[constants.Key]; !ok {
			return errors.NewAppError(http.StatusBadRequest, constants.ErrAPIKey, nil)
		}
		if _, ok := a.Params[constants.Value]; !ok {
			return errors.NewAppError(http.StatusBadRequest, constants.ErrAPIKeyValue, nil)
		}
		if in, ok := a.Params[constants.In]; !ok || (in != constants.Header && in != constants.Query) {
			return errors.NewAppError(http.StatusBadRequest, constants.ErrAPIKeyIn, nil)
		}

	case AuthOAuth2:
		if _, ok := a.Params[constants.AccessToken]; !ok {
			return errors.NewAppError(http.StatusBadRequest, constants.ErrOAuth2Token, nil)
		}

	case AuthAWSSigV4:
		requiredParams := []string{constants.AccessKey, constants.SecretKey, constants.Region, constants.Service}
		for _, param := range requiredParams {
			if _, ok := a.Params[param]; !ok {
				return errors.NewAppError(http.StatusBadRequest, fmt.Sprintf(constants.ErrAuthAWS, param), nil)
			}
		}

	case AuthDigest:
		requiredParams := []string{constants.Username, constants.Password, constants.Realm, constants.Nonce, constants.Qop, constants.NC, constants.Cnonce}
		for _, param := range requiredParams {
			if _, ok := a.Params[param]; !ok {
				return errors.NewAppError(http.StatusBadRequest, fmt.Sprintf(constants.ErrDigestAuth, param), nil)
			}
		}

	default:
		return errors.NewAppError(http.StatusBadRequest, fmt.Sprintf(constants.ErrUnsupportedType, a.Type), nil)
	}

	return nil
}

type OAuth2Config struct {
	ClientID     string   `json:"clientId"`
	ClientSecret string   `json:"clientSecret"`
	AccessToken  string   `json:"accessToken"`
	RefreshToken string   `json:"refreshToken"`
	TokenURL     string   `json:"tokenUrl"`
	AuthURL      string   `json:"authUrl"`
	RedirectURL  string   `json:"redirectUrl"`
	Scopes       []string `json:"scopes"`
	GrantType    string   `json:"grantType"`
}
