package validators

import (
	"context"
	"github.com/FedeBP/pumoide/backend/internal/validators"
	"github.com/FedeBP/pumoide/backend/pkg/constants"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/FedeBP/pumoide/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

func TestOAuth2Manager_GetToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{
            "access_token": "test_access_token",
            "token_type": "Bearer",
            "expires_in": 3600,
            "refresh_token": "test_refresh_token"
        }`))
		if err != nil {
			t.Errorf(constants.ErrFailedToWriteResponse)
			return
		}
	}))
	defer server.Close()

	config := &models.OAuth2Config{
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		TokenURL:     server.URL,
		AuthURL:      server.URL + "/auth",
		RedirectURL:  "http://localhost/callback",
		Scopes:       []string{"read", "write"},
		GrantType:    "client_credentials",
	}

	manager := validators.NewOAuth2Manager(config)

	token, err := manager.GetToken(context.Background(), "client_credentials", nil)
	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, "test_access_token", token.AccessToken)
	assert.Equal(t, "Bearer", token.TokenType)
	assert.Equal(t, "test_refresh_token", token.RefreshToken)
}

func TestOAuth2Manager_RefreshToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{
            "access_token": "new_access_token",
            "token_type": "Bearer",
            "expires_in": 3600,
            "refresh_token": "new_refresh_token"
        }`))
		if err != nil {
			t.Errorf(constants.ErrFailedToWriteResponse)
			return
		}
	}))
	defer server.Close()

	config := &models.OAuth2Config{
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		TokenURL:     server.URL,
		AuthURL:      server.URL + "/auth",
		RedirectURL:  "http://localhost/callback",
		Scopes:       []string{"read", "write"},
		GrantType:    "client_credentials",
	}

	manager := validators.NewOAuth2Manager(config)

	manager.Token = &oauth2.Token{
		AccessToken:  "old_access_token",
		TokenType:    "Bearer",
		RefreshToken: "old_refresh_token",
		Expiry:       time.Now().Add(-1 * time.Hour),
	}

	token, err := manager.RefreshToken(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, "new_access_token", token.AccessToken)
	assert.Equal(t, "Bearer", token.TokenType)
	assert.Equal(t, "new_refresh_token", token.RefreshToken)
}

func TestOAuth2Manager_GetAuthorizationURL(t *testing.T) {
	config := &models.OAuth2Config{
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		TokenURL:     "http://example.com/token",
		AuthURL:      "http://example.com/auth",
		RedirectURL:  "http://localhost/callback",
		Scopes:       []string{"read", "write"},
		GrantType:    "authorization_code",
	}

	manager := validators.NewOAuth2Manager(config)

	url := manager.GetAuthorizationURL()
	assert.Contains(t, url, "http://example.com/auth")
	assert.Contains(t, url, "client_id=test_client_id")
	assert.Contains(t, url, "redirect_uri=http%3A%2F%2Flocalhost%2Fcallback")
	assert.Contains(t, url, "response_type=code")
	assert.Contains(t, url, "scope=read+write")
}
