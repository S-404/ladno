package service

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/s-404/ladno/internal/app/entity"
)

const wsHandshakeTimeout = 10 * time.Second

type WsService struct{}

func NewWsService() *WsService {
	return &WsService{}
}

func (s *WsService) Dial(rawURL string, extra http.Header, cb func(*websocket.Conn, error)) {
	go func() {
		d := websocket.Dialer{
			HandshakeTimeout: wsHandshakeTimeout,
			Proxy:            http.ProxyFromEnvironment,
		}
		conn, _, err := d.Dial(rawURL, stripWSHandshakeHeaders(extra))
		cb(conn, err)
	}()
}

// NewSecWebSocketKey returns a RFC 6455 Sec-WebSocket-Key (16 random bytes, Base64).
func NewSecWebSocketKey() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(b)
}

// HostFromWSURL is the Host header value: hostname (and port if present) of the URL.
func HostFromWSURL(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	if at := strings.LastIndex(s, "@"); at >= 0 {
		s = s[at+1:]
	}
	for i := 0; i < len(s); i++ {
		if s[i] == '/' || s[i] == '?' {
			return s[:i]
		}
	}
	return s
}

// HandshakeAutoHeaders are the default WebSocket upgrade headers shown in the UI.
func HandshakeAutoHeaders(rawURL, secKey string) []entity.Variable {
	if secKey == "" {
		secKey = NewSecWebSocketKey()
	}
	return []entity.Variable{
		{Key: "Host", Value: HostFromWSURL(rawURL)},
		{Key: "Connection", Value: "Upgrade"},
		{Key: "Upgrade", Value: "websocket"},
		{Key: "Sec-WebSocket-Key", Value: secKey},
		{Key: "Sec-WebSocket-Version", Value: "13"},
	}
}

// ExtraWSHeaders are headers passed to Dial. Handshake fields are omitted —
// gorilla/websocket sets Upgrade/Connection/Key/Version on the wire.
func ExtraWSHeaders(resolvedURL string, manual []entity.Variable) http.Header {
	out := make(http.Header)
	if host := HostFromWSURL(resolvedURL); host != "" {
		out.Set("Host", host)
	}
	for _, v := range manual {
		k := strings.TrimSpace(v.Key)
		if k == "" {
			continue
		}
		switch strings.ToLower(k) {
		case "host", "upgrade", "connection", "sec-websocket-key", "sec-websocket-version", "sec-websocket-extensions", "sec-websocket-protocol":
			continue
		default:
			out.Add(k, v.Value)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func ResolveWSURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("url is empty")
	}
	resolved := raw
	if !strings.Contains(resolved, "://") {
		resolved = "ws://" + resolved
	}
	u, err := url.Parse(resolved)
	if err != nil {
		return "", fmt.Errorf("invalid url: %w", err)
	}
	switch strings.ToLower(u.Scheme) {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", fmt.Errorf("invalid websocket scheme %q", u.Scheme)
	}
	if u.Host == "" {
		return "", fmt.Errorf("invalid url: missing host")
	}
	return u.String(), nil
}

func stripWSHandshakeHeaders(in http.Header) http.Header {
	if len(in) == 0 {
		return nil
	}
	out := make(http.Header, len(in))
	for k, vs := range in {
		switch strings.ToLower(k) {
		case "upgrade", "connection", "sec-websocket-key", "sec-websocket-version", "sec-websocket-extensions":
			continue
		default:
			out[k] = vs
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
