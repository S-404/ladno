package store

import (
	"github.com/s-404/ladno/internal/app/service"
)

type Store struct {
	Env       IEnvStore
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
	return &Store{
		Env:       envStore,
		Log:       logStore,
		Nats:      NewNatsStore(service.Nats, envStore, logStore, wsStore, settingsStore),
		Rest:      NewRestStore(service.Rest, envStore, logStore),
		Selection: NewSelectionStore(wsStore),
		Settings:  settingsStore,
		Workspace: wsStore,
	}
}
