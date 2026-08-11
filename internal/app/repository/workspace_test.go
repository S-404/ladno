package repository

import (
	"testing"
	"time"

	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
	"github.com/s-404/ladno/internal/app/storage"
)

func TestWorkspaceRepositoryPersistsCollectionsAcrossReload(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	repo := NewWorkspaceRepository(store)
	all := repo.FindAll()
	if len(all) == 0 {
		t.Fatal("expected seeded workspaces")
	}

	ws := all[0]
	ws.Name = "Persisted Workspace"
	ws.Collections = append(ws.Collections, entity.Collection{
		Id:   "col-persist-1",
		Name: "API",
		Type: constants.CollectionTypeREST,
		Items: []entity.CollectionItem{
			{
				Id:   "req-1",
				Name: "Get users",
				Request: &entity.ItemRequest{
					Method: constants.GET,
					Url:    entity.RequestUrl{Raw: "https://example.com/users"},
				},
			},
			{
				Id:   "folder-1",
				Name: "Nested",
				Item: []entity.CollectionItem{
					{
						Id:   "req-2",
						Name: "Ping",
						Request: &entity.ItemRequest{
							Method: constants.GET,
							Url:    entity.RequestUrl{Raw: "https://example.com/ping"},
						},
					},
				},
			},
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	})
	if err := repo.Save(ws); err != nil {
		t.Fatal(err)
	}

	reloaded := NewWorkspaceRepository(store)
	got := reloaded.FindById(ws.Id)
	if got == nil {
		t.Fatal("workspace missing after reload")
	}
	if got.Name != "Persisted Workspace" {
		t.Fatalf("name: got %q", got.Name)
	}

	var found *entity.Collection
	for i := range got.Collections {
		if got.Collections[i].Id == "col-persist-1" {
			found = &got.Collections[i]
			break
		}
	}
	if found == nil {
		t.Fatal("collection missing after reload")
	}
	if found.Name != "API" || found.Type != constants.CollectionTypeREST {
		t.Fatalf("unexpected collection: %+v", found)
	}
	if len(found.Items) != 2 {
		t.Fatalf("items: want 2, got %d", len(found.Items))
	}
	if found.Items[0].Request == nil || found.Items[0].Request.Url.Raw != "https://example.com/users" {
		t.Fatalf("request: %+v", found.Items[0].Request)
	}
	if len(found.Items[1].Item) != 1 || found.Items[1].Item[0].Name != "Ping" {
		t.Fatalf("nested folder: %+v", found.Items[1])
	}
}

func TestWorkspaceRepositorySaveIsIsolatedFromCaller(t *testing.T) {
	store, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo := NewWorkspaceRepository(store)
	ws := repo.FindAll()[0]

	ws.Name = "Mutated"
	ws.Collections = append(ws.Collections, entity.Collection{
		Id:   "iso-1",
		Name: "Iso",
		Type: constants.CollectionTypeREST,
	})
	if err := repo.Save(ws); err != nil {
		t.Fatal(err)
	}

	ws.Name = "Caller changed again"
	ws.Collections[len(ws.Collections)-1].Name = "Caller renamed"

	stored := repo.FindById(ws.Id)
	if stored.Name != "Mutated" {
		t.Fatalf("repo should keep saved name, got %q", stored.Name)
	}
	last := stored.Collections[len(stored.Collections)-1]
	if last.Id != "iso-1" || last.Name != "Iso" {
		t.Fatalf("repo should keep saved collection, got %+v", last)
	}
}

func TestWorkspaceRepositorySeedOnlyWhenMissing(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	first := NewWorkspaceRepository(store)
	n := len(first.FindAll())
	if n == 0 {
		t.Fatal("expected seed")
	}

	ws := first.FindAll()[0]
	ws.Name = "Custom"
	if err := first.Save(ws); err != nil {
		t.Fatal(err)
	}

	second := NewWorkspaceRepository(store)
	if len(second.FindAll()) != n {
		t.Fatalf("want %d workspaces after reload, got %d", n, len(second.FindAll()))
	}
	got := second.FindById(ws.Id)
	if got == nil || got.Name != "Custom" {
		t.Fatalf("seed should not overwrite: %+v", got)
	}
}

func TestWorkspaceRepositoryDeletePersists(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	repo := NewWorkspaceRepository(store)
	ws := repo.FindAll()[0]
	id := ws.Id
	if err := repo.Delete(id); err != nil {
		t.Fatal(err)
	}

	reloaded := NewWorkspaceRepository(store)
	if reloaded.FindById(id) != nil {
		t.Fatal("deleted workspace still present after reload")
	}
}
