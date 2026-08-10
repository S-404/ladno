package store

import (
	"github.com/s-404/ladno/internal/app/service"
)

type Store struct {
	Env       IEnvStore
	Rest      IRestStore
	Workspace IWorkspaceStore
}

func NewStore(service *service.Service) *Store {
	envStore := NewEnvStore(service.Env)
	return &Store{
		Env:       envStore,
		Rest:      NewRestStore(service.Rest, envStore),
		Workspace: NewWorkspaceStore(service.Workspace),
	}
}
