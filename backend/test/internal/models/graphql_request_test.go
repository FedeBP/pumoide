package models_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/FedeBP/pumoide/backend/internal/domain"
	"github.com/FedeBP/pumoide/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraphQLRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request models.GraphQLRequest
		wantErr bool
	}{
		{
			name: "Valid request",
			request: models.GraphQLRequest{
				Name:  "Test GraphQL",
				URL:   "https://api.example.com/graphql",
				Query: "query { hello }",
			},
			wantErr: false,
		},
		{
			name: "Invalid URL",
			request: models.GraphQLRequest{
				Name:  "Test GraphQL",
				URL:   "invalid-url",
				Query: "query { hello }",
			},
			wantErr: true,
		},
		{
			name: "Empty query",
			request: models.GraphQLRequest{
				Name: "Test GraphQL",
				URL:  "https://api.example.com/graphql",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGraphQLRequest_Execute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data": {"hello": "world"}}`))
	}))
	defer server.Close()

	timeout := 5 * time.Second
	req := &models.GraphQLRequest{
		Name:    "Test GraphQL",
		URL:     server.URL,
		Query:   "query { hello }",
		Timeout: &timeout,
	}

	transport := &http.Transport{}
	resp, err := req.Execute(context.Background(), &domain.Environment{}, transport)
	require.NoError(t, err)
	require.NotNil(t, resp)

	graphqlResp, ok := resp.(*domain.GraphQLResponse)
	require.True(t, ok)
	assert.Equal(t, http.StatusOK, graphqlResp.StatusCode)

	body, ok := graphqlResp.Body.(map[string]interface{})
	require.True(t, ok, "Body should be a map[string]interface{}")

	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok, "data field should be a map[string]interface{}")

	hello, ok := data["hello"].(string)
	require.True(t, ok, "hello field should be a string")
	assert.Equal(t, "world", hello)
}

func TestGraphQLRequest_MarshalUnmarshalJSON(t *testing.T) {
	timeout := 5 * time.Second
	req := &models.GraphQLRequest{
		Name:    "Test GraphQL",
		URL:     "https://api.example.com/graphql",
		Query:   "query { hello }",
		Timeout: &timeout,
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var jsonMap map[string]interface{}
	err = json.Unmarshal(data, &jsonMap)
	require.NoError(t, err)

	assert.Equal(t, "Test GraphQL", jsonMap["name"])
	assert.Equal(t, "https://api.example.com/graphql", jsonMap["url"])
	assert.Equal(t, "query { hello }", jsonMap["query"])
	assert.Equal(t, "5s", jsonMap["timeout"])

	var unmarshaled models.GraphQLRequest
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, req.Name, unmarshaled.Name)
	assert.Equal(t, req.URL, unmarshaled.URL)
	assert.Equal(t, req.Query, unmarshaled.Query)
	require.NotNil(t, unmarshaled.Timeout)
	assert.Equal(t, *req.Timeout, *unmarshaled.Timeout)
}

func TestGraphQLRequest_SetQuery(t *testing.T) {
	req := models.GraphQLRequest{}
	query := `
        query {
            user(id: 1) {
                name
                email
            }
        }
    `
	req.SetQuery(query)

	expected := "query { user(id: 1) { name email } }"
	assert.Equal(t, expected, req.Query)
}
