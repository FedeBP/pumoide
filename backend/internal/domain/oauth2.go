package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

type OAuth2Manager struct {
	config *OAuth2Config
	Token  *oauth2.Token
	client *http.Client
}

func NewOAuth2Manager(config *OAuth2Config) *OAuth2Manager {
	return &OAuth2Manager{
		config: config,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func RefreshOAuth2TokenIfNeeded(req *http.Request, auth *Auth) error {
	if auth.OAuth2 == nil || auth.OAuth2.AccessToken == "" {
		return nil
	}

	token := &oauth2.Token{
		AccessToken:  auth.OAuth2.AccessToken,
		RefreshToken: auth.OAuth2.RefreshToken,
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}

	if token.Expiry.Before(time.Now()) {
		config := &oauth2.Config{
			ClientID:     auth.OAuth2.ClientID,
			ClientSecret: auth.OAuth2.ClientSecret,
			Endpoint: oauth2.Endpoint{
				TokenURL: auth.OAuth2.TokenURL,
				AuthURL:  auth.OAuth2.AuthURL,
			},
			RedirectURL: auth.OAuth2.RedirectURL,
			Scopes:      auth.OAuth2.Scopes,
		}

		newToken, err := config.TokenSource(req.Context(), token).Token()
		if err != nil {
			return err
		}

		auth.OAuth2.AccessToken = newToken.AccessToken
		auth.OAuth2.RefreshToken = newToken.RefreshToken

		req.Header.Set("Authorization", "Bearer "+newToken.AccessToken)
	}

	return nil
}

func (m *OAuth2Manager) GetToken(ctx context.Context, grantType string, params map[string]string) (*oauth2.Token, error) {
	if m.Token != nil && m.Token.Valid() {
		return m.Token, nil
	}

	var err error
	switch grantType {
	case "authorization_code":
		m.Token, err = m.handleAuthorizationCodeGrant(ctx, params)
	case "client_credentials":
		m.Token, err = m.handleClientCredentialsGrant(ctx)
	case "password":
		m.Token, err = m.handlePasswordGrant(ctx, params)
	default:
		return nil, fmt.Errorf("unsupported grant type: %s", grantType)
	}

	if err != nil {
		return nil, err
	}

	return m.Token, nil
}

func (m *OAuth2Manager) RefreshToken(ctx context.Context) (*oauth2.Token, error) {
	if m.Token == nil || m.Token.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available")
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", m.Token.RefreshToken)
	form.Set("client_id", m.config.ClientID)
	form.Set("client_secret", m.config.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", m.config.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send refresh token request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			_ = fmt.Errorf("error closing the body: %w", err)
			return
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh token request failed with status: %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode refresh token response: %w", err)
	}

	m.Token = &oauth2.Token{
		AccessToken:  tokenResp.AccessToken,
		TokenType:    tokenResp.TokenType,
		RefreshToken: tokenResp.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}

	return m.Token, nil
}

func (m *OAuth2Manager) handleAuthorizationCodeGrant(ctx context.Context, params map[string]string) (*oauth2.Token, error) {
	code, ok := params["code"]
	if !ok {
		return nil, fmt.Errorf("authorization code not provided")
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", m.config.ClientID)
	form.Set("client_secret", m.config.ClientSecret)
	form.Set("redirect_uri", m.config.RedirectURL)

	return m.sendTokenRequest(ctx, form)
}

func (m *OAuth2Manager) handleClientCredentialsGrant(ctx context.Context) (*oauth2.Token, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", m.config.ClientID)
	form.Set("client_secret", m.config.ClientSecret)

	if len(m.config.Scopes) > 0 {
		form.Set("scope", strings.Join(m.config.Scopes, " "))
	}

	return m.sendTokenRequest(ctx, form)
}

func (m *OAuth2Manager) handlePasswordGrant(ctx context.Context, params map[string]string) (*oauth2.Token, error) {
	username, ok := params["username"]
	if !ok {
		return nil, fmt.Errorf("username not provided")
	}

	password, ok := params["password"]
	if !ok {
		return nil, fmt.Errorf("password not provided")
	}

	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("username", username)
	form.Set("password", password)
	form.Set("client_id", m.config.ClientID)
	form.Set("client_secret", m.config.ClientSecret)

	if len(m.config.Scopes) > 0 {
		form.Set("scope", strings.Join(m.config.Scopes, " "))
	}

	return m.sendTokenRequest(ctx, form)
}

func (m *OAuth2Manager) sendTokenRequest(ctx context.Context, form url.Values) (*oauth2.Token, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", m.config.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send token request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			_ = fmt.Errorf("error closing the body: %w", err)
			return
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request failed with status: %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &oauth2.Token{
		AccessToken:  tokenResp.AccessToken,
		TokenType:    tokenResp.TokenType,
		RefreshToken: tokenResp.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}, nil
}

func (m *OAuth2Manager) GetAuthorizationURL() string {
	u, err := url.Parse(m.config.AuthURL)
	if err != nil {
		return ""
	}

	q := u.Query()
	q.Set("client_id", m.config.ClientID)
	q.Set("redirect_uri", m.config.RedirectURL)
	q.Set("response_type", "code")
	if len(m.config.Scopes) > 0 {
		q.Set("scope", strings.Join(m.config.Scopes, " "))
	}

	u.RawQuery = q.Encode()
	return u.String()
}
