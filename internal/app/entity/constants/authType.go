package constants

type AuthType string

const (
	AuthTypeInherited AuthType = "Inherited"
	AuthTypeNoAuth    AuthType = "NoAuth"
	AuthTypeBasic     AuthType = "Basic"
	AuthTypeBearer    AuthType = "Bearer"
	AuthTypeAPIKey    AuthType = "ApiKey"
	AuthTypeJSON      AuthType = "Json"
)

// Auth data keys stored in entity.Auth.Data.
const (
	AuthDataUsername = "username"
	AuthDataPassword = "password"
	AuthDataToken    = "token"
	AuthDataPrefix   = "prefix" // Authorization scheme prefix, e.g. "Bearer"
	AuthDataKey      = "key"
	AuthDataValue    = "value"
	AuthDataAddTo    = "addTo" // "header" | "body"
	AuthDataJSON     = "json"
)

const (
	AuthAddToHeader = "header"
	AuthAddToBody   = "body"

	AuthDefaultTokenPrefix = "Bearer"
)

func NormalizeAuthType(t AuthType) AuthType {
	switch t {
	case AuthTypeInherited, AuthTypeNoAuth, AuthTypeBasic, AuthTypeBearer, AuthTypeAPIKey, AuthTypeJSON:
		return t
	case "":
		return AuthTypeNoAuth
	default:
		return t
	}
}
