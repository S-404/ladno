package store

import (
	"github.com/s-404/ladno/internal/app/service"
)

type Store struct {
	Draft     IDraftStore
	Env       IEnvStore
	Kafka     IKafkaStore
	Log       ILogStore
	Nats      INatsStore
	Rest      IRestStore
	Selection ISelectionStore
	Settings  ISettingsStore
	Workspace IWorkspaceStore
}

func NewStore(service *service.Service) *Store {
	settingsStore := NewSettingsStore()
	envStore := NewEnvStore(service.Env, settingsStore)
	logStore := NewLogStore(settingsStore)
	wsStore := NewWorkspaceStore(service.Workspace)
	selStore := NewSelectionStore(wsStore)
	return &Store{
		Draft:     NewDraftStore(wsStore, selStore, envStore),
		Env:       envStore,
		Kafka:     NewKafkaStore(service.Kafka, envStore, logStore, wsStore, settingsStore),
		Log:       logStore,
		Nats:      NewNatsStore(service.Nats, envStore, logStore, wsStore, settingsStore),
		Rest:      NewRestStore(service.Rest, envStore, logStore),
		Selection: selStore,
		Settings:  settingsStore,
		Workspace: wsStore,
	}
}
