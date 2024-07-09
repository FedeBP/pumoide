package constants

const (
	// General errors
	ErrInvalidAction         = "Invalid action"
	ErrMethodNotAllowed      = "Method not allowed"
	ErrInvalidHTTPMethod     = "Invalid HTTP method"
	ErrInvalidURL            = "Invalid URL"
	ErrEmptyHeaderKey        = "Header key cannot be empty"
	ErrFailedToCreateDir     = "Failed to create directory"
	ErrFailedToWriteResponse = "Failed to write response"

	// Collection errors
	ErrCollectionIDRequired     = "Collection ID is required"
	ErrCollectionNotFound       = "Collection not found"
	ErrInvalidCollection        = "Invalid collection"
	ErrEmptyCollectionName      = "collection name cannot be empty"
	ErrFailedToReadCollection   = "Failed to read collection"
	ErrFailedToLoadCollection   = "Failed to load collection"
	ErrFailedToSaveCollection   = "Failed to save collection"
	ErrFailedToDeleteCollection = "Failed to delete collection"

	// Environment errors
	ErrEnvironmentIDRequired     = "Environment ID is required"
	ErrEnvironmentNotFound       = "Environment not found"
	ErrFailedToReadEnvironment   = "Failed to read environment"
	ErrFailedToLoadEnvironment   = "Failed to load environment"
	ErrFailedToSaveEnvironment   = "Failed to save environment"
	ErrFailedToDeleteEnvironment = "Failed to delete environment"

	// Request errors
	ErrInvalidRequest          = "Invalid request"
	ErrInvalidRequestAt        = "invalid request at index %d: %w"
	ErrInvalidRequestBody      = "Invalid request body"
	ErrRequestNotFound         = "Request not found"
	ErrEmptyRequestName        = "Request name cannot be empty"
	ErrFailedToSaveRequest     = "Failed to save request"
	ErrFailedToEncodeRequest   = "Failed to encode request"
	ErrFailedToExecuteRequest  = "Failed to execute request"
	ErrFailedToCloseBody       = "Failed to close the body"
	ErrFailedToReadResponse    = "Failed to read response body"
	ErrFailedToReadRequestBody = "Failed to read request body"
	ErrFailedToDeleteRequest   = "Failed to delete request"

	// History errors
	ErrFailedToDeleteHistory = "Failed to delete history"
	ErrFailedToGetHistory    = "Failed to retrieve history entries"
	ErrMissingHistoryEntryID = "Missing history entry ID"

	// Authentication errors
	ErrInvalidAuth     = "Invalid authentication: %s"
	ErrUnknownAuth     = "Unknown authentication type"
	ErrFailedAwsSigV4  = "Failed to sign request with AWS SigV4"
	ErrBasicAuthUser   = "Basic auth requires a username"
	ErrBasicAuthPass   = "Basic auth requires a password"
	ErrBearerAuth      = "Bearer auth requires a token"
	ErrAPIKey          = "API key auth requires a key"
	ErrAPIKeyValue     = "API key auth requires a value"
	ErrAPIKeyIn        = "API key auth requires 'in' to be either 'header' or 'query'"
	ErrOAuth2Token     = "OAuth2 auth requires an access token"
	ErrAuthAWS         = "AWS SigV4 auth requires %s"
	ErrDigestAuth      = "Digest auth requires %s"
	ErrUnsupportedType = "Unsupported auth type: %s"

	// Validation errors
	ErrInvalidAssertionFormat = "Invalid assertion format. Expected at least 'path operator'"
	ErrUnknownOperator        = "Unknown operator: %s"
	ErrPathNotFound           = "Path '%s' not found in response"
	ErrPathExists             = "Path '%s' exists in response but was expected not to"
	ErrInvalidRegexPattern    = "Invalid regex pattern: %v"
	ErrTypeConversion         = "Cannot convert '%s' to number"
	ErrComparisonFailed       = "Comparison failed: %v %v %v"
	ErrExpectedNotEquals      = "Expected value to not equal '%s'"
	ErrExpectedContains       = "Expected '%s' to contain '%s'"
	ErrExpectedNotContains    = "Expected '%s' to not contain '%s'"
	ErrExpectedMatch          = "'%s' does not match pattern '%s'"
	ErrExpectedType           = "Expected type '%s', but got '%s'"
)
