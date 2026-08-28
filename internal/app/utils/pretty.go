package utils

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"strings"
)

// PrettyBody formats JSON/XML with indentation when content looks like JSON or XML.
// On failure or unsupported content returns body unchanged.
func PrettyBody(body string, contentType string) string {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return body
	}
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if isJSONContent(ct, trimmed) {
		if out, ok := prettyJSON(trimmed); ok {
			return out
		}
	}
	if isXMLContent(ct, trimmed) {
		if out, ok := prettyXML(trimmed); ok {
			return out
		}
	}
	return body
}

func HeaderContentType(headers map[string][]string) string {
	for k, vals := range headers {
		if strings.EqualFold(k, "Content-Type") && len(vals) > 0 {
			return vals[0]
		}
	}
	return ""
}

func isJSONContent(contentType, trimmed string) bool {
	if strings.Contains(contentType, "json") || strings.Contains(contentType, "+json") {
		return true
	}
	return strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")
}

func isXMLContent(contentType, trimmed string) bool {
	if strings.Contains(contentType, "xml") || strings.Contains(contentType, "+xml") {
		return true
	}
	return strings.HasPrefix(trimmed, "<") && !strings.HasPrefix(strings.ToLower(trimmed), "<!doctype html")
}

func prettyJSON(trimmed string) (string, bool) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(trimmed), "", "  "); err != nil {
		return "", false
	}
	return buf.String(), true
}

func prettyXML(trimmed string) (string, bool) {
	dec := xml.NewDecoder(strings.NewReader(trimmed))
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", false
		}
		if err := enc.EncodeToken(tok); err != nil {
			return "", false
		}
	}
	if err := enc.Flush(); err != nil {
		return "", false
	}
	out := buf.String()
	if out == "" {
		return "", false
	}
	return out, true
}
