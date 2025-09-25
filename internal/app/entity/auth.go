package entity

type AuthType string

const (
	AuthTypeInherited AuthType = "Inherited"
	AuthTypeNoAuth    AuthType = "NoAuth"
	AuthTypeBasic     AuthType = "Basic"
	AuthTypeJWT       AuthType = "JWT"
)

type AuthData struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

type Auth struct {
	Type AuthType   `json:"type"`
	Data []AuthData `json:"data"`
}
