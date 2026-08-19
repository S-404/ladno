package store

import (
	"testing"

	"github.com/s-404/ladno/internal/app/entity"
)

func TestCookieStoreAbsorbAndHeader(t *testing.T) {
	s := &CookieStore{}
	s.AbsorbResponse("https://api.example.com/login", map[string][]string{
		"Set-Cookie": {"session=abc; Path=/; HttpOnly"},
	})
	if s.Count() != 1 {
		t.Fatalf("count=%d", s.Count())
	}
	h := s.CookieHeaderForURL("https://api.example.com/v1")
	if h != "session=abc" {
		t.Fatalf("header=%q", h)
	}
	s.Delete("session", "api.example.com", "/")
	if s.Count() != 0 {
		t.Fatal("expected empty after delete")
	}
}

func TestCookieStoreAddDomainAndRename(t *testing.T) {
	s := &CookieStore{}
	s.AddDomain("Example.COM")
	doms := s.Domains()
	if len(doms) != 1 || doms[0] != "example.com" {
		t.Fatalf("domains=%v", doms)
	}
	s.Add(entity.Cookie{Name: "a", Value: "1", Domain: "example.com", Path: "/", HostOnly: true})
	s.Replace(
		entity.Cookie{Name: "a", Domain: "example.com", Path: "/"},
		entity.Cookie{Name: "b", Value: "1", Domain: "example.com", Path: "/", HostOnly: true},
	)
	if s.Count() != 1 {
		t.Fatalf("count=%d", s.Count())
	}
	h := s.CookieHeaderForURL("https://example.com/")
	if h != "b=1" {
		t.Fatalf("header=%q", h)
	}
	s.DeleteDomain("example.com")
	if s.Count() != 0 || len(s.Domains()) != 0 {
		t.Fatal("domain delete failed")
	}
}
