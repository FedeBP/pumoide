package validators

import (
	"context"
	"crypto/md5"
	"fmt"
	"net/http"
	"time"

	"github.com/FedeBP/pumoide/backend/internal/models"
	"github.com/FedeBP/pumoide/backend/internal/utils"
	"github.com/FedeBP/pumoide/backend/pkg/constants"
	"github.com/FedeBP/pumoide/backend/pkg/errors"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/signer/v4"
)

var oauth2Managers = make(map[string]*OAuth2Manager)

func ApplyAuthentication(req *http.Request, auth *models.Auth, env *models.Environment) error {
	if auth == nil || auth.Type == models.AuthNone {
		return nil
	}

	switch auth.Type {
	case models.AuthBasic:
		username := utils.SubstituteVariables(auth.Params[constants.Username], env)
		password := utils.SubstituteVariables(auth.Params[constants.Password], env)
		req.SetBasicAuth(username, password)

	case models.AuthBearer:
		token := utils.SubstituteVariables(auth.Params[constants.Token], env)
		req.Header.Set(constants.Authorization, constants.Bearer+token)

	case models.AuthAPIKey:
		key := utils.SubstituteVariables(auth.Params[constants.Key], env)
		value := utils.SubstituteVariables(auth.Params[constants.Value], env)
		if auth.Params[constants.In] == constants.Header {
			req.Header.Set(key, value)
		} else if auth.Params[constants.In] == constants.Query {
			q := req.URL.Query()
			q.Add(key, value)
			req.URL.RawQuery = q.Encode()
		}

	case models.AuthOAuth2:
		if auth.OAuth2 == nil {
			return errors.NewAppError(http.StatusBadRequest, "OAuth2 configuration is missing", nil)
		}

		managerKey := auth.OAuth2.ClientID + auth.OAuth2.TokenURL
		manager, ok := oauth2Managers[managerKey]
		if !ok {
			manager = NewOAuth2Manager(auth.OAuth2)
			oauth2Managers[managerKey] = manager
		}

		ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
		defer cancel()

		token, err := manager.GetToken(ctx, auth.OAuth2.GrantType, auth.Params)
		if err != nil {
			return errors.NewAppError(http.StatusInternalServerError, constants.ErrOAuth2Token, err)
		}

		if token.Expiry.Before(time.Now()) {
			token, err = manager.RefreshToken(ctx)
			if err != nil {
				return errors.NewAppError(http.StatusInternalServerError, "Failed to refresh OAuth2 token", err)
			}
		}

		req.Header.Set(constants.Authorization, constants.Bearer+token.AccessToken)

	case models.AuthAWSSigV4:
		accessKey := utils.SubstituteVariables(auth.Params[constants.AccessKey], env)
		secretKey := utils.SubstituteVariables(auth.Params[constants.SecretKey], env)
		sessionToken := utils.SubstituteVariables(auth.Params[constants.SessionToken], env)
		region := utils.SubstituteVariables(auth.Params[constants.Region], env)
		service := utils.SubstituteVariables(auth.Params[constants.Service], env)

		creds := credentials.NewStaticCredentials(accessKey, secretKey, sessionToken)
		signer := v4.NewSigner(creds)

		_, err := signer.Sign(req, nil, service, region, time.Now())
		if err != nil {
			return errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedAwsSigV4, err)
		}

	case models.AuthDigest:
		username := utils.SubstituteVariables(auth.Params[constants.Username], env)
		password := utils.SubstituteVariables(auth.Params[constants.Password], env)
		realm := utils.SubstituteVariables(auth.Params[constants.Realm], env)
		nonce := utils.SubstituteVariables(auth.Params[constants.Nonce], env)
		qop := utils.SubstituteVariables(auth.Params[constants.Qop], env)
		nc := utils.SubstituteVariables(auth.Params[constants.NC], env)
		cnonce := utils.SubstituteVariables(auth.Params[constants.Cnonce], env)

		ha1 := md5.Sum([]byte(username + ":" + realm + ":" + password))
		ha2 := md5.Sum([]byte(req.Method + ":" + req.URL.Path))
		response := md5.Sum([]byte(fmt.Sprintf("%x:%s:%s:%s:%s:%x", ha1, nonce, nc, cnonce, qop, ha2)))

		auth := fmt.Sprintf(constants.DigestAuth, username, realm, nonce, req.URL.Path, qop, nc, cnonce, response)
		req.Header.Set(constants.Authorization, auth)

	default:
		return errors.NewAppError(http.StatusBadRequest, constants.ErrUnknownAuth, fmt.Errorf("%s", auth.Type))
	}

	return nil
}
