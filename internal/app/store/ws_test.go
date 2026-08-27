package store

import (
	"strings"
	"testing"
	"time"

	"github.com/s-404/ladno/internal/app/entity"
)

func TestFormatWsMessage(t *testing.T) {
	got := formatWsMessage(WsMessage{
		Dir:  "out",
		Text: "hello",
		Time: time.Date(2026, 1, 1, 15, 4, 5, 0, time.UTC),
	})
	if !strings.Contains(got, "→ hello") {
		t.Fatalf("out: %q", got)
	}
	got = formatWsMessage(WsMessage{
		Dir:  "in",
		Text: `{"ok":true}`,
		Time: time.Date(2026, 1, 1, 15, 4, 5, 0, time.UTC),
	})
	if !strings.Contains(got, "←") || !strings.Contains(got, `"ok"`) {
		t.Fatalf("in: %q", got)
	}
}

func TestApplyWsEnv(t *testing.T) {
	req := applyWsEnv(entity.WsRequest{
		URL:     "wss://{{host}}/:id",
		Message: "hi {{name}}",
		Headers: []entity.Variable{{Key: "X-Token", Value: "{{token}}"}},
		PathParams: []entity.Variable{
			{Key: "id", Value: "{{id}}"},
		},
	}, map[string]string{"host": "example.com", "name": "kot", "token": "abc", "id": "7"})
	if req.URL != "wss://example.com/:id" {
		t.Fatalf("url=%q", req.URL)
	}
	if req.Message != "hi kot" {
		t.Fatalf("msg=%q", req.Message)
	}
	if req.Headers[0].Value != "abc" {
		t.Fatalf("header=%q", req.Headers[0].Value)
	}
	if req.PathParams[0].Value != "7" {
		t.Fatalf("path=%q", req.PathParams[0].Value)
	}
}
