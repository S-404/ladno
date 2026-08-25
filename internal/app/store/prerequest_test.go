package store

import (
	"testing"

	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
)

func TestApplyPreRequest(t *testing.T) {
	env := &memEnvMut{ok: true, vars: map[string]string{"token": "old"}}
	err := ApplyPreRequest([]entity.PreRequestEnvEvent{
		{EnvKey: "token", Action: constants.EnvEventActionSet, Value: "new"},
		{EnvKey: "ready", Action: constants.EnvEventActionSet, Value: "1"},
		{EnvKey: "token", Action: constants.EnvEventActionClear},
	}, env)
	if err != nil {
		t.Fatal(err)
	}
	if env.vars["token"] != "" {
		t.Fatalf("token should be empty, vars=%v", env.vars)
	}
	if env.vars["ready"] != "1" {
		t.Fatalf("vars=%v", env.vars)
	}
}

func TestApplyPreRequestNoEnv(t *testing.T) {
	env := &memEnvMut{ok: false}
	err := ApplyPreRequest([]entity.PreRequestEnvEvent{
		{EnvKey: "token", Value: "x"},
	}, env)
	if err == nil {
		t.Fatal("want error")
	}
}
