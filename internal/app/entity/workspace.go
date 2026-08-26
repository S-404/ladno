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
	Id        string                   `json:"id"`
	Version   int                      `json:"version"`
	Type      constants.CollectionType `json:"type"`
	Name      string                   `json:"name"`
	Auth      Auth                     `json:"auth"`
	Nats      *NatsConnection          `json:"nats,omitempty"`
	Kafka     *KafkaConnection         `json:"kafka,omitempty"`
	Event     Event                    `json:"event"`
	Items     []CollectionItem         `json:"items"`
	CreatedAt time.Time                `json:"createdAt"`
	UpdatedAt time.Time                `json:"updatedAt"`
	DeletedAt *time.Time               `json:"deletedAt"`
}

type CollectionItem struct {
	Id      string           `json:"id"`
	Name    string           `json:"name"`
	Auth    Auth             `json:"auth"`
	Request *ItemRequest     `json:"request"`
	Item    []CollectionItem `json:"item"`
}

type ItemRequest struct {
	Method     constants.RequestMethod `json:"method"`
	Header     []Variable              `json:"header"`
	Auth       Auth                    `json:"auth"`
	Event      Event                   `json:"event"`
	Url        RequestUrl              `json:"url"`
	BodyMode   RestBodyMode            `json:"bodyMode,omitempty"`
	Body       string                  `json:"body,omitempty"`
	FormData   []Variable              `json:"formData,omitempty"`
	URLEncoded []Variable              `json:"urlencoded,omitempty"`
	Grpc       *GrpcRequest            `json:"grpc,omitempty"`
	Ws         *WsRequest              `json:"ws,omitempty"`
	Nats       *NatsRequest            `json:"nats,omitempty"`
	Kafka      *KafkaRequest           `json:"kafka,omitempty"`
}

type GrpcRequest struct {
	Target   string     `json:"target"`
	Method   string     `json:"method"`
	Metadata []Variable `json:"metadata"`
	Message  string     `json:"message"`
}

type WsRequest struct {
	URL     string     `json:"url"`
	Headers []Variable `json:"headers"`
	Message string     `json:"message"`
}

type NatsRequest struct {
	Subject string     `json:"subject"`
	Headers []Variable `json:"headers"`
	Payload string     `json:"payload"`
}

// NatsConnection — подключение на уровне NATS-коллекции (host/port/token).
type NatsConnection struct {
	Host  string `json:"host"`
	Port  string `json:"port"`
	Token string `json:"token"`
}

type KafkaRequest struct {
	Topic   string     `json:"topic"`
	Key     string     `json:"key"`
	Headers []Variable `json:"headers"`
	Payload string     `json:"payload"`
}

// KafkaConnection — подключение на уровне Kafka-коллекции (brokers).
type KafkaConnection struct {
	Brokers string `json:"brokers"` // "localhost:9092" or comma-separated
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
