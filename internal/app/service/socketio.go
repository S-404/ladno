package service

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/s-404/ladno/internal/app/entity"
)

const (
	defaultSocketIOPath = "/socket.io/"
	sioHandshakeTimeout = 10 * time.Second
)

type SocketIOService struct{}

func NewSocketIOService() *SocketIOService {
	return &SocketIOService{}
}

type SocketIOOpen struct {
	Sid          string
	PingInterval time.Duration
	PingTimeout  time.Duration
	MaxPayload   int
}

func (s *SocketIOService) Dial(rawURL string, extra http.Header, namespace, authJSON string, cb func(*websocket.Conn, SocketIOOpen, error)) {
	go func() {
		d := websocket.Dialer{
			HandshakeTimeout: sioHandshakeTimeout,
			Proxy:            http.ProxyFromEnvironment,
		}
		conn, _, err := d.Dial(rawURL, stripWSHandshakeHeaders(extra))
		if err != nil {
			cb(nil, SocketIOOpen{}, err)
			return
		}
		open, err := completeSocketIOHandshake(conn, namespace, authJSON)
		if err != nil {
			_ = conn.Close()
			cb(nil, SocketIOOpen{}, err)
			return
		}
		cb(conn, open, nil)
	}()
}

func completeSocketIOHandshake(conn *websocket.Conn, namespace, authJSON string) (SocketIOOpen, error) {
	_ = conn.SetReadDeadline(time.Now().Add(sioHandshakeTimeout))
	defer func() { _ = conn.SetReadDeadline(time.Time{}) }()

	_, data, err := conn.ReadMessage()
	if err != nil {
		return SocketIOOpen{}, fmt.Errorf("engine.io open: %w", err)
	}
	eng, err := decodeEnginePacket(string(data))
	if err != nil {
		return SocketIOOpen{}, err
	}
	if eng.Type != engineOpen {
		return SocketIOOpen{}, fmt.Errorf("expected engine.io open, got type %d", eng.Type)
	}
	info, err := parseEngineOpen(eng.Data)
	if err != nil {
		return SocketIOOpen{}, err
	}

	if err := conn.WriteMessage(websocket.TextMessage, []byte(encodeConnectPacket(namespace, authJSON))); err != nil {
		return SocketIOOpen{}, fmt.Errorf("socket.io connect: %w", err)
	}

	wantNS := normalizeSocketNamespace(namespace)
	deadline := time.Now().Add(sioHandshakeTimeout)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(deadline)
		_, data, err := conn.ReadMessage()
		if err != nil {
			return SocketIOOpen{}, fmt.Errorf("socket.io connect ack: %w", err)
		}
		eng, err := decodeEnginePacket(string(data))
		if err != nil {
			return SocketIOOpen{}, err
		}
		switch eng.Type {
		case enginePing:
			if err := conn.WriteMessage(websocket.TextMessage, []byte(encodePong(eng.Data))); err != nil {
				return SocketIOOpen{}, err
			}
			continue
		case enginePong:
			continue
		case engineClose:
			return SocketIOOpen{}, fmt.Errorf("engine.io closed during handshake")
		case engineMessage:
			sock, err := decodeSocketPacket(eng.Data)
			if err != nil {
				return SocketIOOpen{}, err
			}
			if sock.Namespace != wantNS {
				continue
			}
			switch sock.Type {
			case socketConnect:
				return SocketIOOpen{
					Sid:          info.Sid,
					PingInterval: time.Duration(info.PingInterval) * time.Millisecond,
					PingTimeout:  time.Duration(info.PingTimeout) * time.Millisecond,
					MaxPayload:   info.MaxPayload,
				}, nil
			case socketConnectError:
				return SocketIOOpen{}, fmt.Errorf("socket.io connect error: %s", sock.Data)
			}
		}
	}
	return SocketIOOpen{}, fmt.Errorf("timed out waiting for socket.io connect")
}

func ResolveSocketIOURL(raw string) (dialURL, namespace string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", fmt.Errorf("url is empty")
	}
	resolved := raw
	if !strings.Contains(resolved, "://") {
		resolved = "http://" + resolved
	}
	u, err := url.Parse(resolved)
	if err != nil {
		return "", "", fmt.Errorf("invalid url: %w", err)
	}
	switch strings.ToLower(u.Scheme) {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", "", fmt.Errorf("invalid socket.io scheme %q", u.Scheme)
	}
	if u.Host == "" {
		return "", "", fmt.Errorf("invalid url: missing host")
	}

	enginePath, namespace := enginePathAndNamespace(u.Path, u.Fragment)
	u.Path = enginePath
	u.Fragment = ""

	q := u.Query()
	q.Del("EIO")
	q.Del("eio")
	q.Del("transport")
	q.Set("EIO", "4")
	q.Set("transport", "websocket")
	u.RawQuery = q.Encode()
	return u.String(), namespace, nil
}

// enginePathAndNamespace reads Engine.IO path and Socket.IO namespace from the URL.
//
//	http://host                  → /socket.io/  + /
//	http://host/admin            → /socket.io/  + /admin
//	http://host/socket.io        → /socket.io/  + /
//	http://host/socket.io/admin  → /socket.io/  + /admin
//	http://host#/admin           → /socket.io/  + /admin
func enginePathAndNamespace(urlPath, fragment string) (enginePath, namespace string) {
	enginePath = defaultSocketIOPath
	namespace = "/"

	var segs []string
	for _, s := range strings.Split(strings.Trim(urlPath, "/"), "/") {
		if s != "" {
			segs = append(segs, s)
		}
	}
	if len(segs) > 0 && isEngineHandshakeSeg(segs[0]) {
		enginePath = "/" + segs[0] + "/"
		if len(segs) > 1 {
			namespace = "/" + strings.Join(segs[1:], "/")
		}
	} else if len(segs) > 0 {
		namespace = "/" + strings.Join(segs, "/")
	}
	if frag := strings.TrimSpace(fragment); frag != "" {
		namespace = frag
	}
	return enginePath, normalizeSocketNamespace(namespace)
}

func isEngineHandshakeSeg(s string) bool {
	return strings.EqualFold(s, "socket.io") || strings.EqualFold(s, "engine.io")
}

// MergeLegacySocketIOURL puts a stored namespace into the URL when the URL has none.
func MergeLegacySocketIOURL(raw, namespace string) string {
	raw = strings.TrimSpace(raw)
	ns := normalizeSocketNamespace(namespace)
	if raw == "" || ns == "/" {
		return raw
	}
	toParse := raw
	addedHTTP := false
	if !strings.Contains(toParse, "://") {
		toParse = "http://" + toParse
		addedHTTP = true
	}
	u, err := url.Parse(toParse)
	if err != nil {
		return raw
	}
	_, existing := enginePathAndNamespace(u.Path, u.Fragment)
	if existing != "/" {
		return raw
	}
	var segs []string
	for _, s := range strings.Split(strings.Trim(u.Path, "/"), "/") {
		if s != "" {
			segs = append(segs, s)
		}
	}
	if len(segs) > 0 && isEngineHandshakeSeg(segs[0]) {
		u.Path = "/" + segs[0] + ns
	} else {
		u.Path = ns
	}
	out := u.String()
	if addedHTTP {
		out = strings.TrimPrefix(out, "http://")
	}
	return out
}

// MergeSocketIOQuery puts stored query params into the URL when it has none.
func MergeSocketIOQuery(raw string, query []entity.Variable) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(query) == 0 {
		return raw
	}
	if strings.Contains(raw, "?") {
		return raw
	}
	var parts []string
	for _, v := range query {
		k := strings.TrimSpace(v.Key)
		if k == "" {
			continue
		}
		parts = append(parts, k+"="+v.Value)
	}
	if len(parts) == 0 {
		return raw
	}
	return raw + "?" + strings.Join(parts, "&")
}

func normalizeSocketNamespace(ns string) string {
	ns = strings.TrimSpace(ns)
	if ns == "" || ns == "/" {
		return "/"
	}
	if !strings.HasPrefix(ns, "/") {
		ns = "/" + ns
	}
	if len(ns) > 1 {
		ns = strings.TrimRight(ns, "/")
	}
	return ns
}

func ExtraSocketIOHeaders(resolvedURL string, extra []entity.Variable) http.Header {
	return ExtraWSHeaders(resolvedURL, extra)
}

func EncodeSocketIOEvent(namespace, event, payload string) (string, error) {
	return encodeEventPacket(namespace, event, payload)
}

// InterpretSocketIOFrame returns an optional pong payload, a UI log line,
// the Socket.IO event name (if this frame is an event), and whether the
// session should disconnect.
func InterpretSocketIOFrame(frame string) (pong, logLine, eventName string, disconnect bool, err error) {
	eng, err := decodeEnginePacket(frame)
	if err != nil {
		return "", "", "", false, err
	}
	switch eng.Type {
	case enginePing:
		return encodePong(eng.Data), "", "", false, nil
	case enginePong:
		return "", "", "", false, nil
	case engineClose:
		return "", "engine closed", "", true, nil
	case engineMessage:
		sock, err := decodeSocketPacket(eng.Data)
		if err != nil {
			return "", "", "", false, err
		}
		switch sock.Type {
		case socketEvent:
			name, payload, err := decodeEventArgs(sock.Data)
			if err != nil {
				return "", "event " + sock.Data, "", false, nil
			}
			line := "event " + name
			if payload != "" {
				line += " " + payload
			}
			return "", line, name, false, nil
		case socketDisconnect:
			return "", "disconnected ns=" + sock.Namespace, "", true, nil
		case socketConnectError:
			return "", "", "", true, fmt.Errorf("%s", sock.Data)
		case socketAck:
			line := "ack"
			if sock.Data != "" {
				line += " " + sock.Data
			}
			return "", line, "", false, nil
		default:
			return "", "", "", false, nil
		}
	default:
		return "", "", "", false, nil
	}
}
