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
	Workspace IWorkspaceStore
}

func NewStore(service *service.Service) *Store {
	envStore := NewEnvStore(service.Env)
	logStore := NewLogStore()
	wsStore := NewWorkspaceStore(service.Workspace)
	return &Store{
		Env:       envStore,
		Log:       logStore,
		Nats:      NewNatsStore(service.Nats, envStore, logStore),
		Rest:      NewRestStore(service.Rest, envStore, logStore),
		Selection: NewSelectionStore(wsStore),
		Workspace: wsStore,
	}
}
