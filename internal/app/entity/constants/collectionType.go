package constants

type CollectionType string

const (
	CollectionTypeHTTP  CollectionType = "http"
	CollectionTypeNATS  CollectionType = "nats"
	CollectionTypeKafka CollectionType = "kafka"
)

func NormalizeCollectionType(t CollectionType) CollectionType {
	switch t {
	case CollectionTypeNATS, CollectionTypeKafka:
		return t
	default:
		return CollectionTypeHTTP
	}
}

func IsHTTPCollection(t CollectionType) bool {
	return NormalizeCollectionType(t) == CollectionTypeHTTP
}

type RequestKind string

const (
	RequestKindREST     RequestKind = "rest"
	RequestKindGRPC     RequestKind = "grpc"
	RequestKindWS       RequestKind = "ws"
	RequestKindSocketIO RequestKind = "socketio"
	RequestKindNATS     RequestKind = "nats"
	RequestKindKafka    RequestKind = "kafka"
)

func RequestKindForCollection(t CollectionType) RequestKind {
	switch NormalizeCollectionType(t) {
	case CollectionTypeNATS:
		return RequestKindNATS
	case CollectionTypeKafka:
		return RequestKindKafka
	default:
		return RequestKindREST
	}
}

// AddRequestMenuLabel — пункт контекстного меню «добавить» для NATS/Kafka.
func AddRequestMenuLabel(t CollectionType) string {
	switch NormalizeCollectionType(t) {
	case CollectionTypeNATS:
		return "Add subject"
	case CollectionTypeKafka:
		return "Add topic"
	default:
		return "Add request"
	}
}

// DefaultNewRequestName — имя нового item по виду запроса.
func DefaultNewRequestName(kind RequestKind) string {
	switch kind {
	case RequestKindNATS:
		return "New subject"
	case RequestKindKafka:
		return "New topic"
	case RequestKindGRPC:
		return "New method"
	case RequestKindWS:
		return "New connection"
	case RequestKindSocketIO:
		return "New Socket.IO"
	default:
		return "New request"
	}
}
