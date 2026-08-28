package entity

import (
	"testing"
	"time"
)

func TestParseSetCookieHeaders(t *testing.T) {
	got := ParseSetCookieHeaders("https://api.example.com/v1/login", map[string][]string{
		"Set-Cookie": {
			"session=abc; Path=/; HttpOnly",
			"theme=dark; Domain=example.com; Path=/",
		},
	})
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].Name != "session" || got[0].Value != "abc" || got[0].Domain != "api.example.com" || !got[0].HostOnly {
		t.Fatalf("session: %+v", got[0])
	}
	if got[1].Name != "theme" || got[1].Domain != "example.com" || got[1].HostOnly {
		t.Fatalf("theme: %+v", got[1])
	}
}

func TestCookiesMatchingURL(t *testing.T) {
	now := time.Now()
	all := []Cookie{
		{Name: "a", Value: "1", Domain: "api.example.com", Path: "/", HostOnly: true},
		{Name: "b", Value: "2", Domain: "example.com", Path: "/", HostOnly: false},
		{Name: "c", Value: "3", Domain: "other.com", Path: "/", HostOnly: true},
		{Name: "secure", Value: "x", Domain: "api.example.com", Path: "/", HostOnly: true, Secure: true},
		{Name: "old", Value: "z", Domain: "api.example.com", Path: "/", HostOnly: true, Expires: now.Add(-time.Hour)},
	}
	got := CookiesMatchingURL(all, "https://api.example.com/v1", now)
	names := map[string]bool{}
	for _, c := range got {
		names[c.Name] = true
	}
	if !names["a"] || !names["b"] || !names["secure"] || names["c"] || names["old"] {
		t.Fatalf("got names=%v", names)
	}
	httpGot := CookiesMatchingURL(all, "http://api.example.com/", now)
	for _, c := range httpGot {
		if c.Name == "secure" {
			t.Fatal("secure cookie must not match http")
		}
	}
}

func TestFormatCookieHeader(t *testing.T) {
	s := FormatCookieHeader([]Cookie{{Name: "a", Value: "1"}, {Name: "b", Value: "2"}})
	if s != "a=1; b=2" {
		t.Fatalf("got %q", s)
	}
}

func TestFormatParseCookieRaw(t *testing.T) {
	c := Cookie{
		Name:     "session",
		Value:    "abc",
		Domain:   "example.com",
		Path:     "/",
		HTTPOnly: true,
		HostOnly: true,
	}
	raw := FormatCookieRaw(c)
	got, ok := ParseCookieRaw(raw, "example.com", true)
	if !ok {
		t.Fatal("parse failed")
	}
	if got.Name != "session" || got.Value != "abc" || got.Path != "/" || !got.HTTPOnly || got.Domain != "example.com" {
		t.Fatalf("got %+v from %q", got, raw)
	}
}
