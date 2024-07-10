package factory

import (
	"fmt"
	"time"

	"github.com/FedeBP/pumoide/backend/internal/domain"
	"github.com/FedeBP/pumoide/backend/internal/models"
)

func CreateRequest(data map[string]interface{}) (domain.Request, error) {
	requestType, ok := data["type"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'type' field")
	}

	switch domain.RequestType(requestType) {
	case domain.RequestTypeREST:
		return createRESTRequest(data)
	case domain.RequestTypeWebSocket:
		return createWebSocketRequest(data)
	case domain.RequestTypeGraphQL:
		return createGraphQLRequest(data)
	default:
		return nil, fmt.Errorf("unsupported request type: %s", requestType)
	}
}

func createRESTRequest(data map[string]interface{}) (*models.RESTRequest, error) {
	req := &models.RESTRequest{
		Type: domain.RequestTypeREST,
	}

	if id, ok := data["id"].(string); ok {
		req.ID = id
	}

	if name, ok := data["name"].(string); ok {
		req.Name = name
	}

	method, ok := data["method"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'method' field")
	}
	req.Method = domain.Method(method)

	url, ok := data["url"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'url' field")
	}
	req.URL = url

	if body, ok := data["body"].(string); ok {
		req.Body = body
	}

	if auth, ok := data["auth"].(map[string]interface{}); ok {
		req.Auth = createAuth(auth)
	}

	if headers, ok := data["headers"].([]interface{}); ok {
		req.Headers = createHeaders(headers)
	}

	if queryParams, ok := data["queryParams"].(map[string]interface{}); ok {
		req.QueryParams = createQueryParams(queryParams)
	}

	if dependsOn, ok := data["dependsOn"].([]interface{}); ok {
		req.DependsOn = createStringSlice(dependsOn)
	}

	if extractVariables, ok := data["extractVariables"].(map[string]interface{}); ok {
		req.ExtractVariables = createStringMap(extractVariables)
	}

	if timeout, ok := data["timeout"].(float64); ok {
		*req.Timeout = time.Duration(timeout) * time.Millisecond
	}

	if validation, ok := data["responseValidation"].(map[string]interface{}); ok {
		req.ResponseValidation = createResponseValidation(validation)
	}

	return req, nil
}

func createWebSocketRequest(data map[string]interface{}) (*models.WebSocketRequest, error) {
	req := &models.WebSocketRequest{
		Type: domain.RequestTypeWebSocket,
	}

	if id, ok := data["id"].(string); ok {
		req.ID = id
	}

	if name, ok := data["name"].(string); ok {
		req.Name = name
	}

	url, ok := data["url"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'url' field")
	}
	req.URL = url

	if auth, ok := data["auth"].(map[string]interface{}); ok {
		req.Auth = createAuth(auth)
	}

	if headers, ok := data["headers"].([]interface{}); ok {
		req.Headers = createHeaders(headers)
	}

	if queryParams, ok := data["queryParams"].(map[string]interface{}); ok {
		req.QueryParams = createQueryParams(queryParams)
	}

	if dependsOn, ok := data["dependsOn"].([]interface{}); ok {
		req.DependsOn = createStringSlice(dependsOn)
	}

	if extractVariables, ok := data["extractVariables"].(map[string]interface{}); ok {
		req.ExtractVariables = createStringMap(extractVariables)
	}

	if protocols, ok := data["protocols"].([]interface{}); ok {
		req.Protocols = createStringSlice(protocols)
	}

	if messageType, ok := data["messageType"].(string); ok {
		req.MessageType = messageType
	}

	if message, ok := data["message"].(string); ok {
		req.Message = message
	}

	if timeout, ok := data["timeout"].(float64); ok {
		*req.Timeout = time.Duration(timeout) * time.Millisecond
	}

	if validation, ok := data["responseValidation"].(map[string]interface{}); ok {
		req.ResponseValidation = createResponseValidation(validation)
	}

	return req, nil
}

func createGraphQLRequest(data map[string]interface{}) (*models.GraphQLRequest, error) {
	req := &models.GraphQLRequest{
		Type: domain.RequestTypeGraphQL,
	}

	if id, ok := data["id"].(string); ok {
		req.ID = id
	}

	if name, ok := data["name"].(string); ok {
		req.Name = name
	}

	url, ok := data["url"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'url' field")
	}
	req.URL = url

	query, ok := data["query"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'query' field")
	}
	req.SetQuery(query)

	if variables, ok := data["variables"].(map[string]interface{}); ok {
		req.Variables = variables
	}

	if operationName, ok := data["operationName"].(string); ok {
		req.OperationName = operationName
	}

	if auth, ok := data["auth"].(map[string]interface{}); ok {
		req.Auth = createAuth(auth)
	}

	if headers, ok := data["headers"].([]interface{}); ok {
		req.Headers = createHeaders(headers)
	}

	if dependsOn, ok := data["dependsOn"].([]interface{}); ok {
		req.DependsOn = createStringSlice(dependsOn)
	}

	if extractVariables, ok := data["extractVariables"].(map[string]interface{}); ok {
		req.ExtractVariables = createStringMap(extractVariables)
	}

	if timeout, ok := data["timeout"].(float64); ok {
		*req.Timeout = time.Duration(timeout) * time.Millisecond
	}

	if validation, ok := data["responseValidation"].(map[string]interface{}); ok {
		req.ResponseValidation = createResponseValidation(validation)
	}

	return req, nil
}

func createAuth(data map[string]interface{}) *domain.Auth {
	auth := &domain.Auth{}

	if authType, ok := data["type"].(string); ok {
		auth.Type = domain.AuthType(authType)
	}

	if params, ok := data["params"].(map[string]interface{}); ok {
		auth.Params = createStringMap(params)
	}

	if oauth2, ok := data["oauth2"].(map[string]interface{}); ok {
		auth.OAuth2 = createOAuth2Config(oauth2)
	}

	return auth
}

func createOAuth2Config(data map[string]interface{}) *domain.OAuth2Config {
	config := &domain.OAuth2Config{}

	if clientID, ok := data["clientId"].(string); ok {
		config.ClientID = clientID
	}
	if clientSecret, ok := data["clientSecret"].(string); ok {
		config.ClientSecret = clientSecret
	}
	if accessToken, ok := data["accessToken"].(string); ok {
		config.AccessToken = accessToken
	}
	if refreshToken, ok := data["refreshToken"].(string); ok {
		config.RefreshToken = refreshToken
	}
	if tokenURL, ok := data["tokenUrl"].(string); ok {
		config.TokenURL = tokenURL
	}
	if authURL, ok := data["authUrl"].(string); ok {
		config.AuthURL = authURL
	}
	if redirectURL, ok := data["redirectUrl"].(string); ok {
		config.RedirectURL = redirectURL
	}
	if scopes, ok := data["scopes"].([]interface{}); ok {
		config.Scopes = createStringSlice(scopes)
	}
	if grantType, ok := data["grantType"].(string); ok {
		config.GrantType = grantType
	}

	return config
}

func createHeaders(data []interface{}) []domain.Header {
	headers := make([]domain.Header, 0, len(data))
	for _, item := range data {
		if header, ok := item.(map[string]interface{}); ok {
			key, _ := header["key"].(string)
			value, _ := header["value"].(string)
			headers = append(headers, domain.Header{Key: key, Value: value})
		}
	}
	return headers
}

func createQueryParams(data map[string]interface{}) map[string]string {
	params := make(map[string]string)
	for key, value := range data {
		if strValue, ok := value.(string); ok {
			params[key] = strValue
		}
	}
	return params
}

func createStringSlice(data []interface{}) []string {
	result := make([]string, 0, len(data))
	for _, item := range data {
		if strItem, ok := item.(string); ok {
			result = append(result, strItem)
		}
	}
	return result
}

func createStringMap(data map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for key, value := range data {
		if strValue, ok := value.(string); ok {
			result[key] = strValue
		}
	}
	return result
}

func createResponseValidation(data map[string]interface{}) *domain.ResponseValidation {
	validation := &domain.ResponseValidation{}

	if statusCode, ok := data["expectedStatusCode"].(float64); ok {
		validation.ExpectedStatusCode = int(statusCode)
	}

	if headers, ok := data["expectedHeaders"].(map[string]interface{}); ok {
		validation.ExpectedHeaders = createStringMap(headers)
	}

	if schema, ok := data["jsonSchema"].(string); ok {
		validation.JSONSchema = schema
	}

	if assertions, ok := data["customAssertions"].([]interface{}); ok {
		validation.CustomAssertions = createStringSlice(assertions)
	}

	return validation
}
