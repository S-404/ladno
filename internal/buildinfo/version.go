package buildinfo

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var raw string

func Version() string {
	return strings.TrimSpace(raw)
}
