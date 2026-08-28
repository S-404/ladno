package store

import (
	"strings"
	"testing"
	"time"

	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
	"github.com/s-404/ladno/internal/app/service"
)

func TestFormatSIOMessage(t *testing.T) {
	got := formatSIOMessage(SIOMessage{
		Dir:  "out",
		Text: "emit chat {\"a\":1}",
		Time: time.Date(2026, 1, 1, 15, 4, 5, 0, time.UTC),
	})
	if !strings.Contains(got, "→") || !strings.Contains(got, "emit chat") {
		t.Fatalf("%q", got)
	}
}

func TestApplySocketIOEnv(t *testing.T) {
	req := applySocketIOEnv(entity.SocketIORequest{
		URL:       "http://{{host}}",
		Namespace: "/{{ns}}",
		Event:     "{{ev}}",
		Payload:   `{"n":"{{name}}"}`,
		Headers:   []entity.Variable{{Key: "X", Value: "{{token}}"}},
		Query:     []entity.Variable{{Key: "token", Value: "{{token}}"}},
	}, map[string]string{"host": "localhost:3000", "ns": "admin", "ev": "chat", "name": "kot", "token": "abc"})
	if req.URL != "http://localhost:3000/admin?token=abc" || req.Event != "chat" {
		t.Fatalf("%+v", req)
	}
	if req.Payload != `{"n":"kot"}` || req.Headers[0].Value != "abc" {
		t.Fatalf("payload/headers %+v", req)
	}
}

func TestApplySocketIOEnvAuth(t *testing.T) {
	req := applySocketIOEnv(entity.SocketIORequest{
		URL: "http://localhost:3000",
		Auth: entity.Auth{
			Type: constants.AuthTypeBearer,
			Data: []entity.Variable{
				{Key: constants.AuthDataPrefix, Value: "Bearer"},
				{Key: constants.AuthDataToken, Value: "{{token}}"},
			},
		},
	}, map[string]string{"token": "abc"})
	if entity.AuthVar(req.Auth.Data, constants.AuthDataToken) != "abc" {
		t.Fatalf("%+v", req.Auth)
	}
}

func TestApplySocketIOAPIKeyHeader(t *testing.T) {
	req := applySocketIOEnv(entity.SocketIORequest{
		URL: "http://localhost:3000",
		Auth: entity.Auth{
			Type: constants.AuthTypeAPIKey,
			Data: []entity.Variable{
				{Key: constants.AuthDataKey, Value: "X-API-Key"},
				{Key: constants.AuthDataValue, Value: "{{token}}"},
				{Key: constants.AuthDataAddTo, Value: constants.AuthAddToHeader},
			},
		},
	}, map[string]string{"token": "secret-1"})
	req.Headers = entity.ApplyAuthHeaders(req.Headers, req.Auth)
	if entity.AuthVar(req.Headers, "X-API-Key") != "secret-1" {
		t.Fatalf("header not substituted: %+v", req.Headers)
	}
	if js := entity.SocketIOAuthJSON(req.Auth, req.AuthJSON); js != "" {
		t.Fatalf("api key must not use CONNECT auth: %q", js)
	}
	hdr := service.ExtraSocketIOHeaders("ws://localhost:3000/socket.io/", req.Headers)
	if hdr.Get("X-API-Key") != "secret-1" {
		t.Fatalf("handshake header: %v", hdr)
	}
}

func TestShowSIOInbound(t *testing.T) {
	if !showSIOInbound(nil, "chat") {
		t.Fatal("no filter should show events")
	}
	if !showSIOInbound(map[string]bool{"chat": true}, "") {
		t.Fatal("system lines should show while listening")
	}
	if !showSIOInbound(map[string]bool{"chat": true}, "chat") {
		t.Fatal("listened event should show")
	}
	if showSIOInbound(map[string]bool{"chat": true}, "other") {
		t.Fatal("other events should be hidden while listening")
	}
}
