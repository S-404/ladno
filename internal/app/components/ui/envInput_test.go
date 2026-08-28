package ui

import "testing"

func TestParseEnvTokens(t *testing.T) {
	tokens := parseEnvTokens(`{{natsHost}}:{{natsPort}}/plain`)
	if len(tokens) != 4 {
		t.Fatalf("len=%d want 4: %+v", len(tokens), tokens)
	}
	if !tokens[0].variable || tokens[0].text != "{{natsHost}}" {
		t.Fatalf("tok0=%+v", tokens[0])
	}
	if tokens[1].variable || tokens[1].text != ":" {
		t.Fatalf("tok1=%+v", tokens[1])
	}
	if !tokens[2].variable || tokens[2].text != "{{natsPort}}" {
		t.Fatalf("tok2=%+v", tokens[2])
	}
	if tokens[3].variable || tokens[3].text != "/plain" {
		t.Fatalf("tok3=%+v", tokens[3])
	}

	if got := parseEnvTokens("no vars"); len(got) != 1 || got[0].variable {
		t.Fatalf("plain=%+v", got)
	}
	if got := parseEnvTokens("{{unclosed"); len(got) != 1 || got[0].variable {
		t.Fatalf("unclosed=%+v", got)
	}
}
