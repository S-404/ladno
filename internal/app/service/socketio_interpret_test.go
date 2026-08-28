package service

import "testing"

func TestInterpretSocketIOFrame(t *testing.T) {
	pong, logLine, eventName, disconnect, err := InterpretSocketIOFrame("2probe")
	if err != nil || pong != "3probe" || logLine != "" || eventName != "" || disconnect {
		t.Fatalf("ping: pong=%q log=%q ev=%q disc=%v err=%v", pong, logLine, eventName, disconnect, err)
	}

	_, logLine, eventName, disconnect, err = InterpretSocketIOFrame(`42["chat",{"text":"hi"}]`)
	if err != nil || disconnect || eventName != "chat" || logLine != `event chat {"text":"hi"}` {
		t.Fatalf("event: %q ev=%q disc=%v err=%v", logLine, eventName, disconnect, err)
	}

	_, logLine, eventName, disconnect, err = InterpretSocketIOFrame("41")
	if err != nil || !disconnect || eventName != "" || logLine != "disconnected ns=/" {
		t.Fatalf("disconnect: %q ev=%q disc=%v err=%v", logLine, eventName, disconnect, err)
	}

	_, _, _, disconnect, err = InterpretSocketIOFrame(`44{"message":"auth"}`)
	if err == nil || !disconnect {
		t.Fatalf("connect error: err=%v disc=%v", err, disconnect)
	}
}
