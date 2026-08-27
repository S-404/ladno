package constants

type CollectionType string

const (
	CollectionTypeREST     CollectionType = "rest"
	CollectionTypeWS       CollectionType = "ws"
	CollectionTypeSocketIO CollectionType = "socketio"
	CollectionTypeGRPC     CollectionType = "grpc"
	CollectionTypeNATS     CollectionType = "nats"
	CollectionTypeKafka    CollectionType = "kafka"
)

func NormalizeCollectionType(t CollectionType) CollectionType {
	switch t {
	case CollectionTypeREST, CollectionTypeWS, CollectionTypeSocketIO, CollectionTypeGRPC,
		CollectionTypeNATS, CollectionTypeKafka:
		return t
	default:
		return CollectionTypeREST
	}
}

// IsHTTPCollection is true for REST, WebSocket, Socket.IO, and gRPC collections.
func IsHTTPCollection(t CollectionType) bool {
	switch NormalizeCollectionType(t) {
	case CollectionTypeREST, CollectionTypeWS, CollectionTypeSocketIO, CollectionTypeGRPC:
		return true
	default:
		return false
	}
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
	case CollectionTypeGRPC:
		return RequestKindGRPC
	case CollectionTypeWS:
		return RequestKindWS
	case CollectionTypeSocketIO:
		return RequestKindSocketIO
	default:
		return RequestKindREST
	}
}

func CollectionTypeForKind(k RequestKind) CollectionType {
	switch k {
	case RequestKindNATS:
		return CollectionTypeNATS
	case RequestKindKafka:
		return CollectionTypeKafka
	case RequestKindGRPC:
		return CollectionTypeGRPC
	case RequestKindWS:
		return CollectionTypeWS
	case RequestKindSocketIO:
		return CollectionTypeSocketIO
	default:
		return CollectionTypeREST
	}
}

func CollectionTypeLabel(t CollectionType) string {
	switch NormalizeCollectionType(t) {
	case CollectionTypeSocketIO:
		return "socket.io"
	default:
		return string(NormalizeCollectionType(t))
	}
}

func AddRequestMenuLabel(t CollectionType) string {
	switch NormalizeCollectionType(t) {
	case CollectionTypeNATS:
		return "Add subject"
	case CollectionTypeKafka:
		return "Add topic"
	case CollectionTypeGRPC:
		return "Add method"
	case CollectionTypeWS:
		return "Add connection"
	case CollectionTypeSocketIO:
		return "Add Socket.IO"
	default:
		return "Add request"
	}
}

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
