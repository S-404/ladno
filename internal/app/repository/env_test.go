package repository

import (
	"testing"

	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/storage"
)

func TestEnvRepositoryCRUDAndClone(t *testing.T) {
	store, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo := NewEnvRepository(store)

	created, err := repo.Create(&entity.Env{
		Name: "Test",
		Variables: []entity.EnvVariable{
			{Key: "host", Value: "localhost", Enabled: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Id == "" || created.Name != "Test" {
		t.Fatalf("unexpected create: %+v", created)
	}

	created.Variables = append(created.Variables, entity.EnvVariable{Key: "port", Value: "8080", Enabled: true})
	updated, err := repo.Update(created)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Variables) != 2 {
		t.Fatalf("expected 2 vars, got %d", len(updated.Variables))
	}

	cloned, err := repo.Clone(updated.Id)
	if err != nil {
		t.Fatal(err)
	}
	if cloned.Id == updated.Id || cloned.Name != "Test (copy)" {
		t.Fatalf("unexpected clone: %+v", cloned)
	}
	if len(cloned.Variables) != 2 {
		t.Fatalf("clone vars: %d", len(cloned.Variables))
	}

	if err := repo.Delete(updated.Id); err != nil {
		t.Fatal(err)
	}
	if repo.FindById(updated.Id) != nil {
		t.Fatal("expected deleted")
	}
	if repo.FindById(cloned.Id) == nil {
		t.Fatal("clone should remain")
	}
}

func TestEnvRepositoryPersistsAcrossReload(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	repo := NewEnvRepository(store)
	// First open seeds mock data onto disk.
	if len(repo.FindAll()) == 0 {
		t.Fatal("expected seeded envs")
	}
	created, err := repo.Create(&entity.Env{
		Name: "Persisted",
		Variables: []entity.EnvVariable{
			{Key: "token", Value: "abc", Enabled: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	reloaded := NewEnvRepository(store)
	got := reloaded.FindById(created.Id)
	if got == nil {
		t.Fatal("env missing after reload")
	}
	if got.Name != "Persisted" || len(got.Variables) != 1 || got.Variables[0].Key != "token" {
		t.Fatalf("unexpected reload: %+v", got)
	}

	if err := reloaded.Delete(created.Id); err != nil {
		t.Fatal(err)
	}
	again := NewEnvRepository(store)
	if again.FindById(created.Id) != nil {
		t.Fatal("deleted env still present after reload")
	}
}

func TestEnvRepositorySeedOnlyWhenMissing(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	first := NewEnvRepository(store)
	n := len(first.FindAll())
	if n == 0 {
		t.Fatal("expected seed")
	}
	if _, err := first.Create(&entity.Env{Name: "Extra"}); err != nil {
		t.Fatal(err)
	}

	second := NewEnvRepository(store)
	if len(second.FindAll()) != n+1 {
		t.Fatalf("want %d envs after reload, got %d", n+1, len(second.FindAll()))
	}
}
