package service

import (
	"strings"
	"testing"

	"github.com/s-404/ladno/internal/app/entity"
)

func TestResolveSocketIOURL(t *testing.T) {
	got, ns, err := ResolveSocketIOURL("localhost:3000")
	if err != nil {
		t.Fatal(err)
	}
	if ns != "/" {
		t.Fatalf("ns=%q", ns)
	}
	if !strings.HasPrefix(got, "ws://localhost:3000/socket.io/") ||
		!strings.Contains(got, "EIO=4") ||
		!strings.Contains(got, "transport=websocket") {
		t.Fatalf("default: %q", got)
	}

	got, ns, err = ResolveSocketIOURL("https://example.com/admin?foo=bar")
	if err != nil {
		t.Fatal(err)
	}
	if ns != "/admin" {
		t.Fatalf("ns=%q", ns)
	}
	if !strings.HasPrefix(got, "wss://example.com/socket.io/") ||
		!strings.Contains(got, "foo=bar") ||
		!strings.Contains(got, "EIO=4") {
		t.Fatalf("https+ns: %q", got)
	}

	got, ns, err = ResolveSocketIOURL("http://host/socket.io/chat")
	if err != nil {
		t.Fatal(err)
	}
	if ns != "/chat" {
		t.Fatalf("socket.io + nsp: ns=%q", ns)
	}
	if !strings.HasPrefix(got, "ws://host/socket.io/") {
		t.Fatalf("dial: %q", got)
	}

	got, ns, err = ResolveSocketIOURL("http://host/engine.io/chat")
	if err != nil {
		t.Fatal(err)
	}
	if ns != "/chat" {
		t.Fatalf("engine.io + nsp: ns=%q", ns)
	}
	if !strings.HasPrefix(got, "ws://host/engine.io/") {
		t.Fatalf("engine.io dial: %q", got)
	}

	got, ns, err = ResolveSocketIOURL("http://host/socket.io/#/admin")
	if err != nil {
		t.Fatal(err)
	}
	if ns != "/admin" {
		t.Fatalf("fragment ns=%q", ns)
	}

	if _, _, err := ResolveSocketIOURL(""); err == nil {
		t.Fatal("empty url")
	}
}

func TestEnginePathAndNamespace(t *testing.T) {
	cases := []struct {
		path, frag, engine, ns string
	}{
		{"", "", "/socket.io/", "/"},
		{"/", "", "/socket.io/", "/"},
		{"/admin", "", "/socket.io/", "/admin"},
		{"/socket.io", "", "/socket.io/", "/"},
		{"/socket.io/", "", "/socket.io/", "/"},
		{"/socket.io/admin", "", "/socket.io/", "/admin"},
		{"/engine.io", "", "/engine.io/", "/"},
		{"/engine.io/admin", "", "/engine.io/", "/admin"},
		{"/", "/admin", "/socket.io/", "/admin"},
	}
	for _, tc := range cases {
		engine, ns := enginePathAndNamespace(tc.path, tc.frag)
		if engine != tc.engine || ns != tc.ns {
			t.Errorf("path=%q frag=%q → %q %q, want %q %q", tc.path, tc.frag, engine, ns, tc.engine, tc.ns)
		}
	}
}

func TestNormalizeSocketNamespace(t *testing.T) {
	if got := normalizeSocketNamespace(""); got != "/" {
		t.Fatalf("%q", got)
	}
	if got := normalizeSocketNamespace("admin"); got != "/admin" {
		t.Fatalf("%q", got)
	}
	if got := normalizeSocketNamespace("/admin/"); got != "/admin" {
		t.Fatalf("%q", got)
	}
}

func TestMergeLegacySocketIOURL(t *testing.T) {
	if got := MergeLegacySocketIOURL("http://host", "/admin"); got != "http://host/admin" {
		t.Fatalf("%q", got)
	}
	if got := MergeLegacySocketIOURL("http://host/socket.io", "/admin"); got != "http://host/socket.io/admin" {
		t.Fatalf("%q", got)
	}
	if got := MergeLegacySocketIOURL("http://host/chat", "/admin"); got != "http://host/chat" {
		t.Fatalf("keep existing ns: %q", got)
	}
	if got := MergeLegacySocketIOURL("localhost:3000", "admin"); got != "localhost:3000/admin" {
		t.Fatalf("%q", got)
	}
}

func TestMergeSocketIOQuery(t *testing.T) {
	got := MergeSocketIOQuery("http://host/admin", []entity.Variable{{Key: "foo", Value: "bar"}})
	if got != "http://host/admin?foo=bar" {
		t.Fatalf("%q", got)
	}
	got = MergeSocketIOQuery("http://host?a=1", []entity.Variable{{Key: "foo", Value: "bar"}})
	if got != "http://host?a=1" {
		t.Fatalf("keep existing query: %q", got)
	}
	if got := MergeSocketIOQuery("http://host", nil); got != "http://host" {
		t.Fatalf("%q", got)
	}
}
