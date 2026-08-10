package store

import (
	"github.com/s-404/ladno/internal/app/service"
)

type Store struct {
	Rest      IRestStore
	Workspace IWorkspaceStore
}

func NewStore(service *service.Service) *Store {
	return &Store{
		Rest:      NewRestStore(service.Rest),
		Workspace: NewWorkspaceStore(service.Workspace),
	}
}
