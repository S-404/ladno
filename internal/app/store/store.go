package store

import (
	"github.com/s-404/ladno/internal/app/service"
)

type Store struct {
	Env       IEnvStore
	Log       ILogStore
	Rest      IRestStore
	Workspace IWorkspaceStore
}

func NewStore(service *service.Service) *Store {
	envStore := NewEnvStore(service.Env)
	logStore := NewLogStore()
	return &Store{
		Env:       envStore,
		Log:       logStore,
		Rest:      NewRestStore(service.Rest, envStore, logStore),
		Workspace: NewWorkspaceStore(service.Workspace),
	}
}
