package entity

import (
	"time"

	"github.com/s-404/ladno/internal/app/entity/constants"
)

type Workspace struct {
	Id               string       `json:"id"`
	Name             string       `json:"name"`
	ConnectionConfig string       `json:"connectionConfig"`
	Collections      []Collection `json:"collections"`
}

type WorkspaceLightWeight struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type Collection struct {
	Id        string           `json:"id"`
	Version   int              `json:"version"`
	Type      string           `json:"type"`
	Name      string           `json:"name"`
	Auth      Auth             `json:"auth"`
	Event     Event            `json:"event"`
	Items     []CollectionItem `json:"items"`
	CreatedAt time.Time        `json:"createdAt"`
	UpdatedAt time.Time        `json:"updatedAt"`
	DeletedAt *time.Time       `json:"deletedAt"`
}

type CollectionItem struct {
	Id      string           `json:"id"`
	Name    string           `json:"name"`
	Request *ItemRequest     `json:"request"`
	Item    []CollectionItem `json:"item"`
}

type ItemRequest struct {
	Method constants.RequestMethod `json:"method"`
	Header []Variable              `json:"header"`
	Auth   Auth                    `json:"auth"`
	Event  Event                   `json:"event"`
	Url    RequestUrl              `json:"url"`
}

type RequestUrl struct {
	Raw      string     `json:"raw"`
	Variable []Variable `json:"variable"`
}

type Auth struct {
	Type constants.AuthType `json:"type"`
	Data []Variable         `json:"data"`
}

type Event struct {
	PreRequest  []PreRequestEnvEvent  `json:"preRequestEvents"`
	PostRequest []PostRequestEnvEvent `json:"postRequestEvents"`
}

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
