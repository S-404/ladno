package service

import (
	"strings"
	"testing"
)

func TestEncodeDecodeEnginePacket(t *testing.T) {
	got := encodeEnginePacket(enginePing, "probe")
	if got != "2probe" {
		t.Fatalf("ping: %q", got)
	}
	pkt, err := decodeEnginePacket(got)
	if err != nil || pkt.Type != enginePing || pkt.Data != "probe" {
		t.Fatalf("decode ping: %+v %v", pkt, err)
	}
	if encodePong("probe") != "3probe" {
		t.Fatalf("pong %q", encodePong("probe"))
	}
}

func TestEncodeConnectPacket(t *testing.T) {
	if got := encodeConnectPacket("/", ""); got != "40" {
		t.Fatalf("default ns: %q", got)
	}
	if got := encodeConnectPacket("", `{"token":"abc"}`); got != `40{"token":"abc"}` {
		t.Fatalf("auth: %q", got)
	}
	if got := encodeConnectPacket("/admin", `{"token":"abc"}`); got != `40/admin,{"token":"abc"}` {
		t.Fatalf("nsp auth: %q", got)
	}
	if got := encodeConnectPacket("admin", ""); got != "40/admin," {
		t.Fatalf("nsp only: %q", got)
	}
}

func TestEncodeEventPacket(t *testing.T) {
	got, err := encodeEventPacket("/", "chat", `{"text":"hi"}`)
	if err != nil {
		t.Fatal(err)
	}
	if got != `42["chat",{"text":"hi"}]` {
		t.Fatalf("json payload: %q", got)
	}
	got, err = encodeEventPacket("/admin", "hello", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != `42/admin,["hello"]` {
		t.Fatalf("nsp event: %q", got)
	}
	got, err = encodeEventPacket("/", "raw", "not-json")
	if err != nil {
		t.Fatal(err)
	}
	if got != `42["raw","not-json"]` {
		t.Fatalf("string payload: %q", got)
	}
	if _, err := encodeEventPacket("/", "", "{}"); err == nil {
		t.Fatal("empty event should fail")
	}
}

func TestDecodeSocketPacket(t *testing.T) {
	p, err := decodeSocketPacket(`0{"sid":"abc"}`)
	if err != nil || p.Type != socketConnect || p.Namespace != "/" || p.Data != `{"sid":"abc"}` {
		t.Fatalf("connect: %+v %v", p, err)
	}
	p, err = decodeSocketPacket(`2/admin,["hello",1]`)
	if err != nil || p.Type != socketEvent || p.Namespace != "/admin" || p.Data != `["hello",1]` {
		t.Fatalf("event nsp: %+v %v", p, err)
	}
	p, err = decodeSocketPacket(`21["ack"]`)
	if err != nil || p.Type != socketEvent || p.ID != "1" || p.Data != `["ack"]` {
		t.Fatalf("ack id: %+v %v", p, err)
	}
}

func TestDecodeEventArgs(t *testing.T) {
	name, payload, err := decodeEventArgs(`["chat",{"text":"hi"}]`)
	if err != nil || name != "chat" || payload != `{"text":"hi"}` {
		t.Fatalf("got %q %q %v", name, payload, err)
	}
	name, payload, err = decodeEventArgs(`["ping"]`)
	if err != nil || name != "ping" || payload != "" {
		t.Fatalf("name only: %q %q %v", name, payload, err)
	}
	name, payload, err = decodeEventArgs(`["multi",1,"x"]`)
	if err != nil || name != "multi" || payload != `[1,"x"]` {
		t.Fatalf("multi: %q %q %v", name, payload, err)
	}
}

func TestParseEngineOpen(t *testing.T) {
	info, err := parseEngineOpen(`{"sid":"s1","pingInterval":25000,"pingTimeout":20000}`)
	if err != nil || info.Sid != "s1" || info.PingInterval != 25000 {
		t.Fatalf("%+v %v", info, err)
	}
	info, err = parseEngineOpen(`{"sid":"s2"}`)
	if err != nil || info.PingInterval != 25000 || info.PingTimeout != 20000 {
		t.Fatalf("defaults %+v %v", info, err)
	}
}

func TestDecodeEngineMessageToSocket(t *testing.T) {
	eng, err := decodeEnginePacket(`42["chat"]`)
	if err != nil || eng.Type != engineMessage {
		t.Fatal(err)
	}
	sock, err := decodeSocketPacket(eng.Data)
	if err != nil || sock.Type != socketEvent {
		t.Fatal(err)
	}
	name, _, err := decodeEventArgs(sock.Data)
	if err != nil || name != "chat" {
		t.Fatalf("%q %v", name, err)
	}
	if !strings.HasPrefix(encodeEngineMessage("2"), "4") {
		t.Fatal("engine message prefix")
	}
}
