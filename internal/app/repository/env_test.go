package repository

import (
	"testing"

	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/storage"
)

const testWS = "ws-test"

func TestEnvRepositoryCRUDAndClone(t *testing.T) {
	store, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo := NewEnvRepository(store)

	created, err := repo.Create(testWS, &entity.Env{
		Name: "Test",
		Variables: []entity.EnvVariable{
			{Key: "host", Value: "localhost", Enabled: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Id == "" || created.Name != "Test" || created.WorkspaceId != testWS {
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
	if updated.WorkspaceId != testWS {
		t.Fatalf("workspace id dropped: %+v", updated)
	}

	cloned, err := repo.Clone(updated.Id)
	if err != nil {
		t.Fatal(err)
	}
	if cloned.Id == updated.Id || cloned.Name != "Test (copy)" || cloned.WorkspaceId != testWS {
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
	if len(repo.FindAll("ws-001")) == 0 {
		t.Fatal("expected seeded envs")
	}
	created, err := repo.Create(testWS, &entity.Env{
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
	if got.Name != "Persisted" || got.WorkspaceId != testWS || len(got.Variables) != 1 || got.Variables[0].Key != "token" {
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
	n := len(first.FindAll("ws-001"))
	if n == 0 {
		t.Fatal("expected seed")
	}
	if _, err := first.Create("ws-001", &entity.Env{Name: "Extra"}); err != nil {
		t.Fatal(err)
	}

	second := NewEnvRepository(store)
	if len(second.FindAll("ws-001")) != n+1 {
		t.Fatalf("want %d envs after reload, got %d", n+1, len(second.FindAll("ws-001")))
	}
}

func TestEnvRepositoryMove(t *testing.T) {
	store, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo := NewEnvRepository(store)
	for _, e := range repo.FindAll("ws-001") {
		if err := repo.Delete(e.Id); err != nil {
			t.Fatal(err)
		}
	}

	a, err := repo.Create(testWS, &entity.Env{Name: "A"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := repo.Create(testWS, &entity.Env{Name: "B"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := repo.Create(testWS, &entity.Env{Name: "C"})
	if err != nil {
		t.Fatal(err)
	}

	if err := repo.Move(testWS, a.Id, 2); err != nil {
		t.Fatal(err)
	}
	ids := envIDs(repo.FindAll(testWS))
	want := []string{b.Id, c.Id, a.Id}
	if len(ids) != 3 || ids[0] != want[0] || ids[1] != want[1] || ids[2] != want[2] {
		t.Fatalf("after move A→2: got %v want %v", ids, want)
	}

	if err := repo.Move(testWS, c.Id, 0); err != nil {
		t.Fatal(err)
	}
	ids = envIDs(repo.FindAll(testWS))
	want = []string{c.Id, b.Id, a.Id}
	if len(ids) != 3 || ids[0] != want[0] || ids[1] != want[1] || ids[2] != want[2] {
		t.Fatalf("after move C→0: got %v want %v", ids, want)
	}

	reloaded := NewEnvRepository(store)
	ids = envIDs(reloaded.FindAll(testWS))
	if len(ids) != 3 || ids[0] != want[0] || ids[1] != want[1] || ids[2] != want[2] {
		t.Fatalf("order not persisted: got %v want %v", ids, want)
	}
}

func TestEnvRepositoryWorkspaceIsolation(t *testing.T) {
	store, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo := NewEnvRepository(store)
	if _, err := repo.Create("ws-a", &entity.Env{Name: "A1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Create("ws-b", &entity.Env{Name: "B1"}); err != nil {
		t.Fatal(err)
	}

	a := repo.FindAll("ws-a")
	b := repo.FindAll("ws-b")
	if len(a) != 1 || a[0].Name != "A1" {
		t.Fatalf("ws-a: %+v", a)
	}
	if len(b) != 1 || b[0].Name != "B1" {
		t.Fatalf("ws-b: %+v", b)
	}

	if err := repo.DeleteByWorkspace("ws-a"); err != nil {
		t.Fatal(err)
	}
	if len(repo.FindAll("ws-a")) != 0 {
		t.Fatal("ws-a envs should be gone")
	}
	if len(repo.FindAll("ws-b")) != 1 {
		t.Fatal("ws-b envs should remain")
	}
}

func TestEnvRepositoryMigrateUnscoped(t *testing.T) {
	store, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo := NewEnvRepository(store)
	repo.mu.Lock()
	repo.envs = []*entity.Env{
		{Id: "env-legacy", Name: "Legacy", Variables: []entity.EnvVariable{{Key: "k", Value: "v", Enabled: true}}},
	}
	if err := repo.persistLocked(); err != nil {
		repo.mu.Unlock()
		t.Fatal(err)
	}
	repo.mu.Unlock()

	repo.MigrateUnscoped([]string{"ws-a", "ws-b"})
	a := repo.FindAll("ws-a")
	b := repo.FindAll("ws-b")
	if len(a) != 1 || a[0].Name != "Legacy" || a[0].Id != "env-legacy" {
		t.Fatalf("first workspace should keep id: %+v", a)
	}
	if len(b) != 1 || b[0].Name != "Legacy" || b[0].Id == "env-legacy" {
		t.Fatalf("second workspace should get a clone: %+v", b)
	}
}

func envIDs(envs []*entity.Env) []string {
	out := make([]string, 0, len(envs))
	for _, e := range envs {
		if e != nil {
			out = append(out, e.Id)
		}
	}
	return out
}
