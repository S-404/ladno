package entity

// RequestDraft — черновик request до Save.
type RequestDraft struct {
	CollectionID string
	Name         string
	Request      ItemRequest
}

// FolderDraft — черновик folder до Save.
type FolderDraft struct {
	CollectionID string
	Name         string
	Auth         Auth
}

// CollectionDraft — черновик collection до Save.
type CollectionDraft struct {
	Name  string
	Auth  Auth
	Nats  *NatsConnection
	Kafka *KafkaConnection
}

// EnvDraft — черновик env до Save.
type EnvDraft struct {
	Name      string
	Variables []EnvVariable
}

const (
	DefaultNewRequestName    = "New request" // REST; see constants.DefaultNewRequestName
	DefaultNewFolderName     = "New folder"
	DefaultNewCollectionName = "New collection"
	DefaultNewEnvName        = "New env"
)
