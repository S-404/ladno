package constants

type CollectionType string

const (
	CollectionTypeREST  CollectionType = "rest"
	CollectionTypeGRPC  CollectionType = "grpc"
	CollectionTypeWS    CollectionType = "ws"
	CollectionTypeNATS  CollectionType = "nats"
	CollectionTypeKafka CollectionType = "kafka"
)

func NormalizeCollectionType(t CollectionType) CollectionType {
	switch t {
	case CollectionTypeGRPC, CollectionTypeWS, CollectionTypeNATS, CollectionTypeKafka:
		return t
	default:
		return CollectionTypeREST
	}
}

// AddRequestMenuLabel — пункт контекстного меню «добавить» по типу коллекции.
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
	default:
		return "Add request"
	}
}

// DefaultNewRequestName — имя нового item по типу коллекции.
func DefaultNewRequestName(t CollectionType) string {
	switch NormalizeCollectionType(t) {
	case CollectionTypeNATS:
		return "New subject"
	case CollectionTypeKafka:
		return "New topic"
	case CollectionTypeGRPC:
		return "New method"
	case CollectionTypeWS:
		return "New connection"
	default:
		return "New request"
	}
}
