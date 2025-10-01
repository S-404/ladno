package constants

type AuthType string

const (
	AuthTypeInherited AuthType = "Inherited"
	AuthTypeNoAuth    AuthType = "NoAuth"
	AuthTypeBasic     AuthType = "Basic"
	AuthTypeJWT       AuthType = "JWT"
)
