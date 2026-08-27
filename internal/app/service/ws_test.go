package service

import (
	"testing"

	"github.com/s-404/ladno/internal/app/entity"
)

func TestHostFromWSURL(t *testing.T) {
	cases := map[string]string{
		"":                                 "",
		"ws://example.com":                 "example.com",
		"wss://example.com:8443/chat":      "example.com:8443",
		"ws://{{host}}:8080/path?x=1":      "{{host}}:8080",
		"wss://user:pass@api.example.com/": "api.example.com",
		"example.com/foo":                  "example.com",
	}
	for in, want := range cases {
		if got := HostFromWSURL(in); got != want {
			t.Errorf("HostFromWSURL(%q)=%q want %q", in, got, want)
		}
	}
}

func TestNewSecWebSocketKey(t *testing.T) {
	a := NewSecWebSocketKey()
	b := NewSecWebSocketKey()
	if a == "" || b == "" {
		t.Fatal("empty key")
	}
	if a == b {
		t.Fatal("keys should be unique")
	}
	if len(a) < 20 {
		t.Fatalf("key too short: %q", a)
	}
}

func TestHandshakeAutoHeaders(t *testing.T) {
	hs := HandshakeAutoHeaders("wss://echo.example.com/ws", "dGhlIHNhbXBsZSBrZXk=")
	got := map[string]string{}
	for _, h := range hs {
		got[h.Key] = h.Value
	}
	if got["Host"] != "echo.example.com" {
		t.Fatalf("Host=%q", got["Host"])
	}
	if got["Connection"] != "Upgrade" || got["Upgrade"] != "websocket" {
		t.Fatalf("upgrade headers: %+v", got)
	}
	if got["Sec-WebSocket-Key"] != "dGhlIHNhbXBsZSBrZXk=" {
		t.Fatalf("key=%q", got["Sec-WebSocket-Key"])
	}
	if got["Sec-WebSocket-Version"] != "13" {
		t.Fatalf("version=%q", got["Sec-WebSocket-Version"])
	}
}

func TestExtraWSHeaders(t *testing.T) {
	h := ExtraWSHeaders("wss://echo.example.com/ws", []entity.Variable{
		{Key: "Authorization", Value: "Bearer x"},
		{Key: "Upgrade", Value: "websocket"},
		{Key: "Host", Value: "ignored.example.com"},
	})
	if h.Get("Host") != "echo.example.com" {
		t.Fatalf("Host=%q", h.Get("Host"))
	}
	if h.Get("Authorization") != "Bearer x" {
		t.Fatalf("Authorization=%q", h.Get("Authorization"))
	}
	if h.Get("Upgrade") != "" {
		t.Fatalf("Upgrade should be stripped, got %q", h.Get("Upgrade"))
	}
}

func TestResolveWSURL(t *testing.T) {
	got, err := ResolveWSURL("ws://example.com/:id/chat", map[string]string{"id": "42"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "ws://example.com/42/chat" {
		t.Fatalf("got %q", got)
	}
	got, err = ResolveWSURL("https://example.com/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "wss://example.com/ws" {
		t.Fatalf("https→wss got %q", got)
	}
}
