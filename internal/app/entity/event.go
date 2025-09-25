package entity

type EnvEventAction string

const (
	EnvEventActionSet   EnvEventAction = "set"
	EnvEventActionClear EnvEventAction = "clear"
)

type PreRequestEnvEvent struct {
	EnvKey string         `json:"envKey"`
	Action EnvEventAction `json:"action"`
	Value  string         `json:"value"`
}

type PostRequestEnvEvent struct {
	EnvKey   string         `json:"envKey"`
	Action   EnvEventAction `json:"action"`
	Value    string         `json:"value"`
	JSONPath string         `json:"JSONPath"`
}

type Event struct {
	PreRequest  []PreRequestEnvEvent  `json:"preRequestEvents"`
	PostRequest []PostRequestEnvEvent `json:"postRequestEvents"`
}
