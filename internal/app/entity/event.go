package entity

import "github.com/s-404/goose/internal/app/entity/constants"

type PreRequestEnvEvent struct {
	EnvKey string                   `json:"envKey"`
	Action constants.EnvEventAction `json:"action"`
	Value  string                   `json:"value"`
}

type PostRequestEnvEvent struct {
	EnvKey   string                   `json:"envKey"`
	Action   constants.EnvEventAction `json:"action"`
	Value    string                   `json:"value"`
	JSONPath string                   `json:"JSONPath"`
}

type Event struct {
	PreRequest  []PreRequestEnvEvent  `json:"preRequestEvents"`
	PostRequest []PostRequestEnvEvent `json:"postRequestEvents"`
}
