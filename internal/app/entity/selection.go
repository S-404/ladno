package entity

import "github.com/s-404/ladno/internal/app/entity/constants"

type SelectionKind string

const (
	SelectionNone       SelectionKind = "none"
	SelectionCollection SelectionKind = "collection"
	SelectionFolder     SelectionKind = "folder"
	SelectionRequest    SelectionKind = "request"
)

// Selection — текущий выбор в дереве коллекций.
type Selection struct {
	Kind           SelectionKind
	CollectionID   string
	CollectionType constants.CollectionType
	ItemID         string
	Path           []string
	Name           string
	Auth           Auth
	Nats           *NatsConnection
	Kafka          *KafkaConnection
	Request        *ItemRequest
	// FocusName — после выбора сразу открыть редактирование имени (новая запись).
	FocusName bool
}
