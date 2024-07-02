package utils

// Collection-Request constants
const (
	Action                 = "action"
	ActionExport           = "export"
	ActionImport           = "import"
	ActionAddRequest       = "addRequest"
	ActionUpdateCollection = "updateCollection"
	ActionDeleteCollection = "deleteCollection"
	ActionDeleteRequest    = "deleteRequest"
	CollectionID           = "collectionId"
	RequestID              = "requestId"
	ID                     = "id"
	Path                   = "path"
	Env                    = "env"
	EmptyString            = ""
)

// Basic error messages
const (
	InvalidActionErr         = "Invalid action"
	MethodNotAllowedErr      = "Method not allowed"
	InvalidHTTPMethodErr     = "Invalid HTTP method"
	InvalidURLErr            = "Invalid URL"
	EmptyHeaderKeyErr        = "Header key cannot be empty"
	FailedToCreateDirErr     = "Failed to create directory"
	FailedToSaveDirErr       = "Failed to save directory"
	FailedToWriteResponseErr = "Failed to write response"
)

// Collection error messages
const (
	CollectionIdRequiredErr     = "Collection ID is required"
	CollectionNotFoundErr       = "Collection not found"
	InvalidCollectionErr        = "Invalid collection"
	EmptyCollectionNameErr      = "collection name cannot be empty"
	FailedToReadCollectionErr   = "Failed to read collection"
	FailedToLoadCollectionErr   = "Failed to load collection"
	FailedToSaveCollectionErr   = "Failed to save collection"
	FailedToDeleteCollectionErr = "Failed to delete collection"
)

// Environment error messages
const (
	EnvironmentIdRequiredErr     = "Environment ID is required"
	EnvironmentNotFoundErr       = "Environment not found"
	FailedToReadEnvironmentErr   = "Failed to read environment"
	FailedToLoadEnvironmentErr   = "Failed to load environment"
	FailedToSaveEnvironmentErr   = "Failed to save environment"
	FailedToDeleteEnvironmentErr = "Failed to delete environment"
)

// Request error messages
const (
	InvalidRequestErr         = "Invalid request"
	InvalidRequestAtErr       = "invalid request at index %d: %w"
	InvalidRequestBodyErr     = "Invalid request body"
	RequestNotFoundErr        = "Request not found"
	EmptyRequestNameErr       = "Request name cannot be empty"
	FailedToSaveRequestErr    = "Failed to save request"
	FailedToEncodeRequestErr  = "Failed to encode request"
	FailedToExecuteRequestErr = "Failed to execute request"
	FailedToCreateRequestErr  = "Failed to create request"
	FailedToCloseBodyErr      = "Failed to close the body"
	FailedToReadResponseErr   = "Failed to read response body"
)

// Authentication error messages
const (
	InvalidAuthErr          = "Invalid authentication: %s"
	UnknownAuthErr          = "Unknown authentication type"
	FailedToAuthenticateErr = "Failed to apply authentication"
	FailedAwsSigV4Err       = "Failed to sign request with AWS SigV4"
	BasicAuthUserErr        = "Basic auth requires a username"
	BasicAuthPassErr        = "Basic auth requires a password"
	BearerAuthErr           = "Bearer auth requires a token"
	APIKeyErr               = "API key auth requires a key"
	APIKeyValueErr          = "API key auth requires a value"
	APIKeyInErr             = "API key auth requires 'in' to be either 'header' or 'query'"
	OAuth2TokenErr          = "OAuth2 auth requires an access token"
	AuthAWSErr              = "AWS SigV4 auth requires %s"
	DigestAuthErr           = "Digest auth requires %s"
	UnsupportedTypeErr      = "Unsupported auth type: %s"
)

// API constants
const (
	ContentType        = "Content-Type"
	ContentDisposition = "Content-Disposition"
	AppJson            = "application/json"
)

// Authentication constants
const (
	Authorization = "Authorization"
	Username      = "username"
	Password      = "password"
	Bearer        = "Bearer "
	Token         = "token"
	Header        = "header"
	Query         = "query"
	In            = "in"
	Key           = "key"
	Value         = "value"
	AccessToken   = "access_token"
	SessionToken  = "session_token"
	AccessKey     = "access_key"
	SecretKey     = "secret_key"
	Region        = "region"
	Service       = "service"
	Realm         = "realm"
	Nonce         = "nonce"
	Qop           = "qop"
	NC            = "nc"
	Cnonce        = "cnonce"
	DigestAuth    = `Digest username="%s", realm="%s", nonce="%s", uri="%s", qop=%s, nc=%s, cnonce="%s", response="%x"`
)

// Success messages
const (
	CollectionDeletedSuccess  = "Collection deleted successfully"
	RequestDeletedSuccess     = "Request deleted successfully"
	EnvironmentDeletedSuccess = "Environment deleted successfully"
)
