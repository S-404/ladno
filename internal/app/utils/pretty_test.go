package utils

import (
	"strings"
	"testing"
)

func TestPrettyBodyJSON(t *testing.T) {
	in := `{"a":1,"b":{"c":true}}`
	out := PrettyBody(in, "application/json")
	if !strings.Contains(out, "\n") || !strings.Contains(out, `"a"`) {
		t.Fatalf("pretty json=%q", out)
	}
}

func TestPrettyBodyXML(t *testing.T) {
	in := `<root><item id="1">x</item></root>`
	out := PrettyBody(in, "application/xml")
	if !strings.Contains(out, "\n") || !strings.Contains(out, "<root>") {
		t.Fatalf("pretty xml=%q", out)
	}
}

func TestPrettyBodyPassthrough(t *testing.T) {
	in := "plain text"
	if PrettyBody(in, "text/plain") != in {
		t.Fatal("expected unchanged")
	}
}
