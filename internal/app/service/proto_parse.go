package service

import (
	"strings"
	"unicode"
)

// ProtoRPC is one rpc declared in a .proto file.
type ProtoRPC struct {
	Service         string
	Name            string
	ClientStreaming bool
	ServerStreaming bool
}

func (p ProtoRPC) FullName(pkg string) string {
	if pkg != "" {
		return pkg + "." + p.Service + "/" + p.Name
	}
	return p.Service + "/" + p.Name
}

func (p ProtoRPC) Streaming() bool {
	return p.ClientStreaming || p.ServerStreaming
}

// ParseProtoRPCs lists unary and streaming RPCs from proto source.
func ParseProtoRPCs(src string) (pkg string, methods []ProtoRPC) {
	src = stripProtoComments(src)
	pkg = protoPackage(src)
	for _, svc := range protoServiceBlocks(src) {
		for _, rpc := range parseServiceRPCs(svc.body) {
			rpc.Service = svc.name
			methods = append(methods, rpc)
		}
	}
	return pkg, methods
}

// ProtoMethodNames returns package.Service/Method names in file order.
func ProtoMethodNames(src string) []string {
	pkg, methods := ParseProtoRPCs(src)
	out := make([]string, 0, len(methods))
	seen := map[string]bool{}
	for _, m := range methods {
		name := m.FullName(pkg)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

func protoPackage(src string) string {
	rest := src
	for {
		i := strings.Index(rest, "package")
		if i < 0 {
			return ""
		}
		if i > 0 {
			prev := rune(rest[i-1])
			if unicode.IsLetter(prev) || unicode.IsDigit(prev) || prev == '_' {
				rest = rest[i+7:]
				continue
			}
		}
		after := strings.TrimLeft(rest[i+7:], " \t\r\n")
		ident, ok := readProtoIdent(after)
		if !ok {
			rest = rest[i+7:]
			continue
		}
		tail := strings.TrimLeft(after[len(ident):], " \t\r\n")
		if strings.HasPrefix(tail, ";") {
			return ident
		}
		rest = rest[i+7:]
	}
}

type protoService struct {
	name string
	body string
}

func protoServiceBlocks(src string) []protoService {
	var out []protoService
	rest := src
	for {
		i := strings.Index(rest, "service")
		if i < 0 {
			return out
		}
		if i > 0 {
			prev := rune(rest[i-1])
			if unicode.IsLetter(prev) || unicode.IsDigit(prev) || prev == '_' {
				rest = rest[i+7:]
				continue
			}
		}
		after := strings.TrimLeft(rest[i+7:], " \t\r\n")
		name, ok := readProtoIdent(after)
		if !ok {
			rest = rest[i+7:]
			continue
		}
		tail := strings.TrimLeft(after[len(name):], " \t\r\n")
		if !strings.HasPrefix(tail, "{") {
			rest = rest[i+7:]
			continue
		}
		body, consumed, ok := extractBraceBody(tail)
		if !ok {
			return out
		}
		out = append(out, protoService{name: name, body: body})
		rest = tail[consumed:]
	}
}

func parseServiceRPCs(body string) []ProtoRPC {
	var out []ProtoRPC
	rest := body
	for {
		i := strings.Index(rest, "rpc")
		if i < 0 {
			return out
		}
		if i > 0 {
			prev := rune(rest[i-1])
			if unicode.IsLetter(prev) || unicode.IsDigit(prev) || prev == '_' {
				rest = rest[i+3:]
				continue
			}
		}
		after := strings.TrimLeft(rest[i+3:], " \t\r\n")
		name, ok := readProtoIdent(after)
		if !ok {
			rest = rest[i+3:]
			continue
		}
		tail := strings.TrimLeft(after[len(name):], " \t\r\n")
		if !strings.HasPrefix(tail, "(") {
			rest = rest[i+3:]
			continue
		}
		inArgs, n1, ok := extractParenBody(tail)
		if !ok {
			return out
		}
		tail = strings.TrimLeft(tail[n1:], " \t\r\n")
		if !strings.HasPrefix(tail, "returns") {
			rest = rest[i+3:]
			continue
		}
		tail = strings.TrimLeft(tail[len("returns"):], " \t\r\n")
		outArgs, n2, ok := extractParenBody(tail)
		if !ok {
			return out
		}
		out = append(out, ProtoRPC{
			Name:            name,
			ClientStreaming: protoHasStream(inArgs),
			ServerStreaming: protoHasStream(outArgs),
		})
		rest = skipRPCSuffix(tail[n2:])
	}
}

func skipRPCSuffix(s string) string {
	s = strings.TrimLeft(s, " \t\r\n")
	if strings.HasPrefix(s, "{") {
		_, n, ok := extractBraceBody(s)
		if ok {
			return s[n:]
		}
	}
	if strings.HasPrefix(s, ";") {
		return s[1:]
	}
	return s
}

func protoHasStream(args string) bool {
	s := strings.TrimSpace(args)
	if !strings.HasPrefix(s, "stream") {
		return false
	}
	if len(s) == len("stream") {
		return true
	}
	r := rune(s[len("stream")])
	return unicode.IsSpace(r)
}

func readProtoIdent(s string) (string, bool) {
	if s == "" {
		return "", false
	}
	n := 0
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.' {
			n += len(string(r))
			continue
		}
		break
	}
	if n == 0 {
		return "", false
	}
	return s[:n], true
}

func extractBraceBody(s string) (body string, consumed int, ok bool) {
	if !strings.HasPrefix(s, "{") {
		return "", 0, false
	}
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[1:i], i + 1, true
			}
		}
	}
	return "", 0, false
}

func extractParenBody(s string) (body string, consumed int, ok bool) {
	if !strings.HasPrefix(s, "(") {
		return "", 0, false
	}
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return s[1:i], i + 1, true
			}
		}
	}
	return "", 0, false
}

func stripProtoComments(src string) string {
	var b strings.Builder
	b.Grow(len(src))
	i := 0
	for i < len(src) {
		if i+1 < len(src) && src[i] == '/' && src[i+1] == '/' {
			i += 2
			for i < len(src) && src[i] != '\n' {
				i++
			}
			continue
		}
		if i+1 < len(src) && src[i] == '/' && src[i+1] == '*' {
			i += 2
			for i+1 < len(src) && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			if i+1 < len(src) {
				i += 2
			}
			continue
		}
		b.WriteByte(src[i])
		i++
	}
	return b.String()
}
