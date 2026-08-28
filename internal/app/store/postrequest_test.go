package store

import (
	"testing"

	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
)

type memEnvMut struct {
	vars map[string]string
	ok   bool
}

func (m *memEnvMut) ActiveVariables() map[string]string { return m.vars }
func (m *memEnvMut) UpsertActiveVar(key, value string) bool {
	if !m.ok {
		return false
	}
	if m.vars == nil {
		m.vars = map[string]string{}
	}
	m.vars[key] = value
	return true
}
func (m *memEnvMut) ClearActiveVar(key string) bool {
	if !m.ok {
		return false
	}
	if m.vars == nil {
		m.vars = map[string]string{}
	}
	m.vars[key] = ""
	return true
}

func TestApplyPostRequest(t *testing.T) {
	env := &memEnvMut{ok: true, vars: map[string]string{"token": "old"}}
	err := ApplyPostRequest(
		`{"access":"new","user":{"id":7}}`,
		[]entity.PostRequestEnvEvent{
			{EnvKey: "token", Action: constants.EnvEventActionSet, JSONPath: "access"},
			{EnvKey: "uid", Action: constants.EnvEventActionSet, JSONPath: "user.id"},
			{EnvKey: "skip", Action: constants.EnvEventActionSet, JSONPath: "missing?.x"},
		},
		env,
	)
	if err != nil {
		t.Fatal(err)
	}
	if env.vars["token"] != "new" || env.vars["uid"] != "7" {
		t.Fatalf("vars=%v", env.vars)
	}
	if _, ok := env.vars["skip"]; ok {
		t.Fatal("optional miss should not set")
	}
}

func TestApplyPostRequestError(t *testing.T) {
	env := &memEnvMut{ok: true}
	err := ApplyPostRequest(`{}`, []entity.PostRequestEnvEvent{
		{EnvKey: "token", JSONPath: "access"},
	}, env)
	if err == nil {
		t.Fatal("want error")
	}
}
