package constants

const (
	// Auth parameters
	Username     = "username"
	Password     = "password"
	Bearer       = "Bearer "
	Token        = "token"
	Header       = "header"
	Query        = "query"
	In           = "in"
	Key          = "key"
	Value        = "value"
	AccessToken  = "access_token"
	SessionToken = "session_token"
	AccessKey    = "access_key"
	SecretKey    = "secret_key"
	Region       = "region"
	Service      = "service"
	Realm        = "realm"
	Nonce        = "nonce"
	Qop          = "qop"
	NC           = "nc"
	Cnonce       = "cnonce"

	// Digest auth format
	DigestAuth = `Digest username="%s", realm="%s", nonce="%s", uri="%s", qop=%s, nc=%s, cnonce="%s", response="%x"`
)
