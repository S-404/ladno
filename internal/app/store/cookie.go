package store

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"github.com/s-404/ladno/internal/app/entity"
)

const prefCookiesKey = "cookieJar"

type cookieJarData struct {
	Domains []string        `json:"domains,omitempty"`
	Cookies []entity.Cookie `json:"cookies"`
}

// CookieStore — in-app HTTP cookie jar (persisted in preferences).
type CookieStore struct {
	mu        sync.Mutex
	cookies   []entity.Cookie
	domains   []string // explicit domains (may have zero cookies)
	listeners []func()
}

func NewCookieStore() *CookieStore {
	s := &CookieStore{}
	s.load()
	return s
}

func (s *CookieStore) prefs() fyne.Preferences {
	if fyne.CurrentApp() == nil {
		return nil
	}
	return fyne.CurrentApp().Preferences()
}

func (s *CookieStore) load() {
	p := s.prefs()
	if p == nil {
		return
	}
	raw := p.String(prefCookiesKey)
	if raw == "" {
		return
	}
	now := time.Now().UTC()

	// Legacy format: bare []Cookie
	var legacy []entity.Cookie
	if err := json.Unmarshal([]byte(raw), &legacy); err == nil && (len(legacy) > 0 || raw == "[]") {
		out := make([]entity.Cookie, 0, len(legacy))
		for _, c := range legacy {
			if c.Name == "" || c.Expired(now) {
				continue
			}
			out = append(out, c)
		}
		s.cookies = out
		s.domains = domainsFromCookies(out)
		return
	}

	var data cookieJarData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return
	}
	out := make([]entity.Cookie, 0, len(data.Cookies))
	for _, c := range data.Cookies {
		if c.Name == "" || c.Expired(now) {
			continue
		}
		out = append(out, c)
	}
	s.cookies = out
	s.domains = normalizeDomains(append(data.Domains, domainsFromCookies(out)...))
}

func (s *CookieStore) saveLocked() {
	p := s.prefs()
	if p == nil {
		return
	}
	data := cookieJarData{
		Domains: append([]string{}, s.domains...),
		Cookies: s.cookies,
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	p.SetString(prefCookiesKey, string(raw))
}

// AddListener registers a callback for jar changes.
func (s *CookieStore) AddListener(fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, fn)
}

func (s *CookieStore) notifyListeners() {
	s.mu.Lock()
	ls := append([]func(){}, s.listeners...)
	s.mu.Unlock()
	for _, fn := range ls {
		if fn != nil {
			fn()
		}
	}
}

// List returns a copy of all non-expired cookies.
func (s *CookieStore) List() []entity.Cookie {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(time.Now().UTC())
	out := make([]entity.Cookie, len(s.cookies))
	copy(out, s.cookies)
	return out
}

// Domains returns known domains (including empty ones).
func (s *CookieStore) Domains() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(time.Now().UTC())
	out := make([]string, len(s.domains))
	copy(out, s.domains)
	return out
}

// Count returns the number of stored cookies.
func (s *CookieStore) Count() int {
	return len(s.List())
}

// Clear removes all cookies and domains.
func (s *CookieStore) Clear() {
	s.mu.Lock()
	s.cookies = nil
	s.domains = nil
	s.saveLocked()
	s.mu.Unlock()
	s.notifyListeners()
}

// AddDomain registers a domain (even with no cookies yet).
func (s *CookieStore) AddDomain(domain string) {
	domain = normalizeDomain(domain)
	if domain == "" {
		return
	}
	s.mu.Lock()
	s.ensureDomainLocked(domain)
	s.saveLocked()
	s.mu.Unlock()
	s.notifyListeners()
}

// DeleteDomain removes a domain and all its cookies.
func (s *CookieStore) DeleteDomain(domain string) {
	domain = normalizeDomain(domain)
	if domain == "" {
		return
	}
	s.mu.Lock()
	next := s.cookies[:0]
	for _, c := range s.cookies {
		if normalizeDomain(c.Domain) != domain {
			next = append(next, c)
		}
	}
	s.cookies = next
	doms := s.domains[:0]
	for _, d := range s.domains {
		if d != domain {
			doms = append(doms, d)
		}
	}
	s.domains = doms
	s.saveLocked()
	s.mu.Unlock()
	s.notifyListeners()
}

// Add inserts a cookie (or replaces same name/domain/path).
func (s *CookieStore) Add(c entity.Cookie) {
	c.Domain = normalizeDomain(c.Domain)
	if c.Name == "" || c.Domain == "" {
		return
	}
	if c.Path == "" {
		c.Path = "/"
	}
	s.mu.Lock()
	s.ensureDomainLocked(c.Domain)
	s.upsertLocked(c)
	s.saveLocked()
	s.mu.Unlock()
	s.notifyListeners()
}

// Delete removes a cookie by name/domain/path.
func (s *CookieStore) Delete(name, domain, path string) {
	s.mu.Lock()
	key := entity.Cookie{Name: name, Domain: normalizeDomain(domain), Path: path}.CookieKey()
	s.removeKeyLocked(key)
	s.saveLocked()
	s.mu.Unlock()
	s.notifyListeners()
}

// Update replaces a cookie matched by name/domain/path without UI notify.
func (s *CookieStore) Update(c entity.Cookie) {
	c.Domain = normalizeDomain(c.Domain)
	if c.Name == "" || c.Domain == "" {
		return
	}
	if c.Path == "" {
		c.Path = "/"
	}
	s.mu.Lock()
	s.ensureDomainLocked(c.Domain)
	s.upsertLocked(c)
	s.saveLocked()
	s.mu.Unlock()
}

// Replace removes prev identity and upserts next (for rename) without UI notify.
func (s *CookieStore) Replace(prev, next entity.Cookie) {
	next.Domain = normalizeDomain(next.Domain)
	prev.Domain = normalizeDomain(prev.Domain)
	if next.Name == "" || next.Domain == "" {
		return
	}
	if next.Path == "" {
		next.Path = "/"
	}
	if prev.Path == "" {
		prev.Path = "/"
	}
	s.mu.Lock()
	s.removeKeyLocked(prev.CookieKey())
	s.ensureDomainLocked(next.Domain)
	s.upsertLocked(next)
	s.saveLocked()
	s.mu.Unlock()
}

// AbsorbResponse stores Set-Cookie values from an HTTP response.
func (s *CookieStore) AbsorbResponse(requestURL string, headers map[string][]string) {
	parsed := entity.ParseSetCookieHeaders(requestURL, headers)
	if len(parsed) == 0 {
		return
	}
	s.mu.Lock()
	now := time.Now().UTC()
	s.pruneLocked(now)
	for _, c := range parsed {
		c.Domain = normalizeDomain(c.Domain)
		if c.Expired(now) {
			s.removeKeyLocked(c.CookieKey())
			continue
		}
		if c.Domain != "" {
			s.ensureDomainLocked(c.Domain)
		}
		s.upsertLocked(c)
	}
	s.saveLocked()
	s.mu.Unlock()
	s.notifyListeners()
}

// CookieHeaderForURL returns Cookie header value for the URL, or empty.
func (s *CookieStore) CookieHeaderForURL(rawURL string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	s.pruneLocked(now)
	matched := entity.CookiesMatchingURL(s.cookies, rawURL, now)
	return entity.FormatCookieHeader(matched)
}

func (s *CookieStore) pruneLocked(now time.Time) {
	next := s.cookies[:0]
	changed := false
	for _, c := range s.cookies {
		if c.Expired(now) {
			changed = true
			continue
		}
		next = append(next, c)
	}
	s.cookies = next
	if changed {
		s.saveLocked()
	}
}

func (s *CookieStore) upsertLocked(c entity.Cookie) {
	key := c.CookieKey()
	for i := range s.cookies {
		if s.cookies[i].CookieKey() == key {
			s.cookies[i] = c
			return
		}
	}
	s.cookies = append(s.cookies, c)
}

func (s *CookieStore) removeKeyLocked(key string) {
	next := s.cookies[:0]
	for _, c := range s.cookies {
		if c.CookieKey() != key {
			next = append(next, c)
		}
	}
	s.cookies = next
}

func (s *CookieStore) ensureDomainLocked(domain string) {
	domain = normalizeDomain(domain)
	if domain == "" {
		return
	}
	for _, d := range s.domains {
		if d == domain {
			return
		}
	}
	s.domains = append(s.domains, domain)
}

func normalizeDomain(domain string) string {
	return strings.ToLower(strings.TrimSpace(strings.TrimPrefix(domain, ".")))
}

func domainsFromCookies(cookies []entity.Cookie) []string {
	seen := map[string]bool{}
	var out []string
	for _, c := range cookies {
		d := normalizeDomain(c.Domain)
		if d == "" || seen[d] {
			continue
		}
		seen[d] = true
		out = append(out, d)
	}
	return out
}

func normalizeDomains(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, d := range in {
		d = normalizeDomain(d)
		if d == "" || seen[d] {
			continue
		}
		seen[d] = true
		out = append(out, d)
	}
	return out
}
