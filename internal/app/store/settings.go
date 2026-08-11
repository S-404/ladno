package store

import (
	"fyne.io/fyne/v2"
	uitheme "github.com/s-404/ladno/internal/app/theme"
)

const (
	prefMessageLimitKey    = "messageHistoryLimit"
	prefThemeKey           = "uiTheme"
	prefLastWorkspaceKey   = "lastWorkspaceId"
	prefActiveEnvKey       = "activeEnvId"
	defaultMessageLimit    = 1000
	minMessageLimit        = 10
	maxMessageLimitAllowed = 100000

	ThemeLight = "light"
	ThemeDark  = "dark"
)

type ISettingsStore interface {
	GetMessageLimit() int
	SetMessageLimit(n int) int

	GetTheme() string
	SetTheme(name string) string
	ApplyTheme()

	GetLastWorkspaceID() string
	SetLastWorkspaceID(id string)

	GetActiveEnvID() string
	SetActiveEnvID(id string)
}

type SettingsStore struct{}

func NewSettingsStore() *SettingsStore {
	return &SettingsStore{}
}

func (s *SettingsStore) prefs() fyne.Preferences {
	return fyne.CurrentApp().Preferences()
}

func (s *SettingsStore) GetMessageLimit() int {
	n := s.prefs().IntWithFallback(prefMessageLimitKey, defaultMessageLimit)
	return clampMessageLimit(n)
}

// SetMessageLimit сохраняет лимит и возвращает фактически записанное значение.
func (s *SettingsStore) SetMessageLimit(n int) int {
	n = clampMessageLimit(n)
	s.prefs().SetInt(prefMessageLimitKey, n)
	return n
}

func (s *SettingsStore) GetTheme() string {
	return normalizeTheme(s.prefs().StringWithFallback(prefThemeKey, ThemeDark))
}

func (s *SettingsStore) SetTheme(name string) string {
	name = normalizeTheme(name)
	s.prefs().SetString(prefThemeKey, name)
	s.ApplyTheme()
	return name
}

func (s *SettingsStore) ApplyTheme() {
	switch s.GetTheme() {
	case ThemeLight:
		fyne.CurrentApp().Settings().SetTheme(uitheme.Light())
	default:
		fyne.CurrentApp().Settings().SetTheme(uitheme.Dark())
	}
}

func (s *SettingsStore) GetLastWorkspaceID() string {
	return s.prefs().String(prefLastWorkspaceKey)
}

func (s *SettingsStore) SetLastWorkspaceID(id string) {
	s.prefs().SetString(prefLastWorkspaceKey, id)
}

func (s *SettingsStore) GetActiveEnvID() string {
	return s.prefs().String(prefActiveEnvKey)
}

func (s *SettingsStore) SetActiveEnvID(id string) {
	s.prefs().SetString(prefActiveEnvKey, id)
}

func clampMessageLimit(n int) int {
	if n < minMessageLimit {
		return minMessageLimit
	}
	if n > maxMessageLimitAllowed {
		return maxMessageLimitAllowed
	}
	return n
}

func normalizeTheme(name string) string {
	if name == ThemeLight {
		return ThemeLight
	}
	return ThemeDark
}
