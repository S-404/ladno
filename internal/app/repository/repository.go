package repository

import (
	"log"

	"github.com/s-404/ladno/internal/app/storage"
)

type Repository struct {
	Env       IEnvRepository
	Workspace IWorkspaceRepository
	Storage   *storage.Store
}

func NewRepository() *Repository {
	store, err := storage.OpenApp()
	if err != nil {
		log.Printf("[storage] open app root failed: %v (envs stay in-memory seed)", err)
		return &Repository{
			Env:       NewEnvRepository(nil),
			Workspace: NewWorkspaceRepository(),
		}
	}
	log.Printf("[storage] root=%s", store.Root())
	return &Repository{
		Env:       NewEnvRepository(store),
		Workspace: NewWorkspaceRepository(),
		Storage:   store,
	}
}

// NewRepositoryWithStorage — для тестов с произвольным каталогом.
func NewRepositoryWithStorage(store *storage.Store) *Repository {
	return &Repository{
		Env:       NewEnvRepository(store),
		Workspace: NewWorkspaceRepository(),
		Storage:   store,
	}
}
