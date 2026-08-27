package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	engineOpen    = 0
	engineClose   = 1
	enginePing    = 2
	enginePong    = 3
	engineMessage = 4

	socketConnect      = 0
	socketDisconnect   = 1
	socketEvent        = 2
	socketAck          = 3
	socketConnectError = 4
)

type enginePacket struct {
	Type int
	Data string
}

type socketPacket struct {
	Type      int
	Namespace string
	ID        string
	Data      string
}

type engineOpenInfo struct {
	Sid          string   `json:"sid"`
	Upgrades     []string `json:"upgrades"`
	PingInterval int      `json:"pingInterval"`
	PingTimeout  int      `json:"pingTimeout"`
	MaxPayload   int      `json:"maxPayload"`
}

func encodeEnginePacket(typ int, data string) string {
	return string(rune('0'+typ)) + data
}

func decodeEnginePacket(s string) (enginePacket, error) {
	if s == "" {
		return enginePacket{}, fmt.Errorf("empty engine.io packet")
	}
	typ := int(s[0] - '0')
	if typ < 0 || typ > 6 {
		return enginePacket{}, fmt.Errorf("invalid engine.io type %q", s[:1])
	}
	return enginePacket{Type: typ, Data: s[1:]}, nil
}

func encodeSocketPacket(p socketPacket) string {
	var b strings.Builder
	b.WriteByte(byte('0' + p.Type))
	ns := normalizeSocketNamespace(p.Namespace)
	if ns != "/" {
		b.WriteString(ns)
		b.WriteByte(',')
	}
	b.WriteString(p.ID)
	b.WriteString(p.Data)
	return b.String()
}

func decodeSocketPacket(s string) (socketPacket, error) {
	if s == "" {
		return socketPacket{}, fmt.Errorf("empty socket.io packet")
	}
	typ := int(s[0] - '0')
	if typ < 0 || typ > 6 {
		return socketPacket{}, fmt.Errorf("invalid socket.io type %q", s[:1])
	}
	i := 1
	ns := "/"
	if i < len(s) && s[i] == '/' {
		j := i + 1
		for j < len(s) && s[j] != ',' {
			j++
		}
		ns = s[i:j]
		if j < len(s) && s[j] == ',' {
			i = j + 1
		} else {
			i = j
		}
	}
	idStart := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	id := s[idStart:i]
	data := ""
	if i < len(s) {
		data = s[i:]
	}
	return socketPacket{Type: typ, Namespace: normalizeSocketNamespace(ns), ID: id, Data: data}, nil
}

func encodeEngineMessage(socketPayload string) string {
	return encodeEnginePacket(engineMessage, socketPayload)
}

func encodeConnectPacket(namespace, authJSON string) string {
	authJSON = strings.TrimSpace(authJSON)
	return encodeEngineMessage(encodeSocketPacket(socketPacket{
		Type:      socketConnect,
		Namespace: namespace,
		Data:      authJSON,
	}))
}

func encodeEventPacket(namespace, event, payload string) (string, error) {
	event = strings.TrimSpace(event)
	if event == "" {
		return "", fmt.Errorf("event name is empty")
	}
	args := []any{event}
	payload = strings.TrimSpace(payload)
	if payload != "" {
		var v any
		if json.Unmarshal([]byte(payload), &v) == nil {
			args = append(args, v)
		} else {
			args = append(args, payload)
		}
	}
	raw, err := json.Marshal(args)
	if err != nil {
		return "", err
	}
	return encodeEngineMessage(encodeSocketPacket(socketPacket{
		Type:      socketEvent,
		Namespace: namespace,
		Data:      string(raw),
	})), nil
}

func encodePong(data string) string {
	return encodeEnginePacket(enginePong, data)
}

func parseEngineOpen(data string) (engineOpenInfo, error) {
	var info engineOpenInfo
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		return engineOpenInfo{}, fmt.Errorf("invalid engine.io open packet: %w", err)
	}
	if info.PingInterval <= 0 {
		info.PingInterval = 25000
	}
	if info.PingTimeout <= 0 {
		info.PingTimeout = 20000
	}
	return info, nil
}

func decodeEventArgs(data string) (name string, payload string, err error) {
	data = strings.TrimSpace(data)
	if data == "" {
		return "", "", fmt.Errorf("empty event")
	}
	var args []json.RawMessage
	if err := json.Unmarshal([]byte(data), &args); err != nil {
		return "", "", err
	}
	if len(args) == 0 {
		return "", "", fmt.Errorf("event missing name")
	}
	if err := json.Unmarshal(args[0], &name); err != nil {
		return "", "", fmt.Errorf("event name: %w", err)
	}
	if len(args) == 1 {
		return name, "", nil
	}
	if len(args) == 2 {
		return name, compactJSON(args[1]), nil
	}
	rest, err := json.Marshal(args[1:])
	if err != nil {
		return name, data, nil
	}
	return name, string(rest), nil
}

func compactJSON(raw json.RawMessage) string {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return ""
	}
	var buf bytes.Buffer
	if json.Compact(&buf, trimmed) == nil {
		return buf.String()
	}
	return string(trimmed)
}
