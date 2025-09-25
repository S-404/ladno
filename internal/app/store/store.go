package store

import (
	"github.com/s-404/goose/internal/app/service"
)

type Store struct {
	Foo       IFooStore
	Bar       IBarStore
	Workspace IWorkspaceStore
}

func NewStore(service *service.Service) *Store {
	return &Store{
		Foo:       NewFooStore(service.Foo),
		Bar:       NewBarStore(service.Bar),
		Workspace: NewWorkspaceStore(service.Workspace),
	}
}
