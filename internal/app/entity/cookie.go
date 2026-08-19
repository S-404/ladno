package entity

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Cookie — запись в cookie jar приложения.
type Cookie struct {
	Name     string    `json:"name"`
	Value    string    `json:"value"`
	Domain   string    `json:"domain"`
	Path     string    `json:"path"`
	Expires  time.Time `json:"expires,omitempty"`
	MaxAge   int       `json:"maxAge,omitempty"`
	Secure   bool      `json:"secure,omitempty"`
	HTTPOnly bool      `json:"httpOnly,omitempty"`
	HostOnly bool      `json:"hostOnly,omitempty"`
}

// CookieKey uniquely identifies a cookie in the jar.
func (c Cookie) CookieKey() string {
	return strings.ToLower(c.Domain) + "\n" + c.Path + "\n" + c.Name
}

// Expired reports whether the cookie should be discarded.
func (c Cookie) Expired(now time.Time) bool {
	if c.MaxAge < 0 {
		return true
	}
	if !c.Expires.IsZero() && !c.Expires.After(now) {
		return true
	}
	return false
}

// ParseSetCookieHeaders parses Set-Cookie header values for a request URL.
func ParseSetCookieHeaders(requestURL string, headers map[string][]string) []Cookie {
	if len(headers) == 0 {
		return nil
	}
	var lines []string
	for k, vals := range headers {
		if !strings.EqualFold(k, "Set-Cookie") {
			continue
		}
		lines = append(lines, vals...)
	}
	if len(lines) == 0 {
		return nil
	}
	u, err := url.Parse(strings.TrimSpace(requestURL))
	if err != nil || u.Host == "" {
		u = nil
	}
	out := make([]Cookie, 0, len(lines))
	for _, line := range lines {
		hc, err := http.ParseSetCookie(line)
		if err != nil || hc == nil || hc.Name == "" {
			continue
		}
		out = append(out, cookieFromHTTP(hc, u))
	}
	return out
}

func cookieFromHTTP(hc *http.Cookie, reqURL *url.URL) Cookie {
	c := Cookie{
		Name:     hc.Name,
		Value:    hc.Value,
		Path:     hc.Path,
		Secure:   hc.Secure,
		HTTPOnly: hc.HttpOnly,
		MaxAge:   hc.MaxAge,
	}
	if !hc.Expires.IsZero() {
		c.Expires = hc.Expires.UTC()
	}
	if hc.MaxAge > 0 {
		c.Expires = time.Now().UTC().Add(time.Duration(hc.MaxAge) * time.Second)
	}
	if c.Path == "" {
		c.Path = "/"
	}
	domain := strings.TrimSpace(hc.Domain)
	if domain == "" && reqURL != nil {
		c.Domain = strings.ToLower(reqURL.Hostname())
		c.HostOnly = true
	} else {
		c.Domain = strings.ToLower(strings.TrimPrefix(domain, "."))
		c.HostOnly = false
	}
	return c
}

// CookiesMatchingURL returns jar cookies that should be sent to u.
func CookiesMatchingURL(all []Cookie, rawURL string, now time.Time) []Cookie {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil
	}
	if !strings.Contains(rawURL, "://") {
		rawURL = "http://" + rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return nil
	}
	host := strings.ToLower(u.Hostname())
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	https := u.Scheme == "https"
	var out []Cookie
	for _, c := range all {
		if c.Expired(now) || c.Name == "" {
			continue
		}
		if c.Secure && !https {
			continue
		}
		if !cookieDomainMatch(c, host) {
			continue
		}
		if !cookiePathMatch(c.Path, path) {
			continue
		}
		out = append(out, c)
	}
	return out
}

// FormatCookieHeader builds a Cookie request header value.
func FormatCookieHeader(cookies []Cookie) string {
	if len(cookies) == 0 {
		return ""
	}
	parts := make([]string, 0, len(cookies))
	for _, c := range cookies {
		if c.Name == "" {
			continue
		}
		parts = append(parts, c.Name+"="+c.Value)
	}
	return strings.Join(parts, "; ")
}

// FormatCookieRaw serializes a cookie as a Set-Cookie line (without Domain).
func FormatCookieRaw(c Cookie) string {
	if c.Name == "" {
		return c.Value
	}
	var b strings.Builder
	b.WriteString(c.Name)
	b.WriteByte('=')
	b.WriteString(c.Value)
	path := c.Path
	if path == "" {
		path = "/"
	}
	b.WriteString("; Path=")
	b.WriteString(path)
	if c.Secure {
		b.WriteString("; Secure")
	}
	if c.HTTPOnly {
		b.WriteString("; HttpOnly")
	}
	if !c.Expires.IsZero() {
		b.WriteString("; Expires=")
		b.WriteString(c.Expires.UTC().Format(time.RFC1123))
	} else if c.MaxAge > 0 {
		b.WriteString("; Max-Age=")
		b.WriteString(strconv.Itoa(c.MaxAge))
	}
	return b.String()
}

// ParseCookieRaw parses a Set-Cookie line, keeping domain/hostOnly from the group.
func ParseCookieRaw(raw, domain string, hostOnly bool) (Cookie, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Cookie{}, false
	}
	hc, err := http.ParseSetCookie(raw)
	if err != nil || hc == nil || hc.Name == "" {
		// Allow typing name without '=' yet — treat whole line as name.
		if !strings.Contains(raw, "=") && !strings.Contains(raw, ";") {
			name := strings.TrimSpace(raw)
			if name == "" {
				return Cookie{}, false
			}
			return Cookie{Name: name, Domain: domain, Path: "/", HostOnly: hostOnly}, true
		}
		return Cookie{}, false
	}
	c := cookieFromHTTP(hc, nil)
	c.Domain = strings.ToLower(strings.TrimSpace(domain))
	c.HostOnly = hostOnly
	return c, true
}

func cookieDomainMatch(c Cookie, host string) bool {
	d := strings.ToLower(c.Domain)
	if d == "" {
		return false
	}
	if c.HostOnly {
		return host == d
	}
	return host == d || strings.HasSuffix(host, "."+d)
}

func cookiePathMatch(cookiePath, reqPath string) bool {
	if cookiePath == "" {
		cookiePath = "/"
	}
	if !strings.HasPrefix(reqPath, cookiePath) {
		return false
	}
	if len(reqPath) == len(cookiePath) {
		return true
	}
	if strings.HasSuffix(cookiePath, "/") {
		return true
	}
	return len(reqPath) > len(cookiePath) && reqPath[len(cookiePath)] == '/'
}

// HasHeaderKey reports whether headers already contain key (case-insensitive).
func HasHeaderKey(headers []Variable, key string) bool {
	for _, h := range headers {
		if strings.EqualFold(strings.TrimSpace(h.Key), key) {
			return true
		}
	}
	return false
}
