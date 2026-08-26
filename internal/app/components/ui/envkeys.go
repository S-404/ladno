package ui

import (
	"sync"

	"fyne.io/fyne/v2"
)

var (
	knownEnvKeysMu sync.RWMutex
	knownEnvKeys   map[string]struct{}

	liveEnvHighlights sync.Map // *EnvInput | *UrlInput -> struct{}
)

// SetKnownEnvKeys updates the set of keys present in the active environment.
// Missing {{key}} tokens in EnvInput/UrlInput are highlighted in red.
func SetKnownEnvKeys(keys []string) {
	next := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		if k == "" {
			continue
		}
		next[k] = struct{}{}
	}
	knownEnvKeysMu.Lock()
	knownEnvKeys = next
	knownEnvKeysMu.Unlock()
	refreshEnvHighlights()
}

// IsKnownEnvKey reports whether key exists in the active environment list.
func IsKnownEnvKey(key string) bool {
	knownEnvKeysMu.RLock()
	defer knownEnvKeysMu.RUnlock()
	_, ok := knownEnvKeys[key]
	return ok
}

func registerEnvHighlight(v any) {
	liveEnvHighlights.Store(v, struct{}{})
}

func unregisterEnvHighlight(v any) {
	liveEnvHighlights.Delete(v)
}

func refreshEnvHighlights() {
	liveEnvHighlights.Range(func(k, _ any) bool {
		switch w := k.(type) {
		case *EnvInput:
			e := w
			fyne.Do(func() {
				if !e.focused {
					e.Refresh()
				}
			})
		case *UrlInput:
			u := w
			fyne.Do(func() {
				if !u.focused {
					u.Refresh()
				}
			})
		}
		return true
	})
}
