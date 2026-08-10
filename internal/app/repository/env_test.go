package repository

import (
	"testing"

	"github.com/s-404/ladno/internal/app/entity"
)

func TestEnvRepositoryCRUDAndClone(t *testing.T) {
	repo := NewEnvRepository()

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
