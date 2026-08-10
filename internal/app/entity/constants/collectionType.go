package constants

type CollectionType string

const (
	CollectionTypeREST CollectionType = "rest"
	CollectionTypeGRPC CollectionType = "grpc"
	CollectionTypeWS   CollectionType = "ws"
	CollectionTypeNATS CollectionType = "nats"
)

func NormalizeCollectionType(t CollectionType) CollectionType {
	switch t {
	case CollectionTypeGRPC, CollectionTypeWS, CollectionTypeNATS:
		return t
	default:
		return CollectionTypeREST
	}
}
