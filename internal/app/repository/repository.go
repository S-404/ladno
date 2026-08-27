package repository

import (
	"log"

	"github.com/s-404/ladno/internal/app/storage"
)

type Repository struct {
	Env       *EnvRepository
	Workspace *WorkspaceRepository
	Storage   *storage.Store
}

func NewRepository() *Repository {
	store, err := storage.OpenApp()
	if err != nil {
		log.Printf("[storage] open app root failed: %v (data stay in-memory seed)", err)
		return newRepository(nil)
	}
	log.Printf("[storage] root=%s", store.Root())
	return newRepository(store)
}

// NewRepositoryWithStorage — для тестов с произвольным каталогом.
func NewRepositoryWithStorage(store *storage.Store) *Repository {
	return newRepository(store)
}

func newRepository(store *storage.Store) *Repository {
	env := NewEnvRepository(store)
	ws := NewWorkspaceRepository(store)
	ws.SetEnvRepository(env)
	ids := make([]string, 0)
	for _, w := range ws.FindAll() {
		if w != nil && w.Id != "" {
			ids = append(ids, w.Id)
		}
	}
	env.MigrateUnscoped(ids)
	return &Repository{
		Env:       env,
		Workspace: ws,
		Storage:   store,
	}
}
