package store

import (
	"fyne.io/fyne/v2"
	uitheme "github.com/s-404/ladno/internal/app/theme"
)

const (
	prefMessageLimitKey    = "messageHistoryLimit"
	prefThemeKey           = "uiTheme"
	prefFontSizeKey        = "uiFontSize"
	prefLastWorkspaceKey   = "lastWorkspaceId"
	prefActiveEnvKey       = "activeEnvId"
	defaultMessageLimit    = 1000
	minMessageLimit        = 10
	maxMessageLimitAllowed = 100000

	ThemeLight = "light"
	ThemeDark  = "dark"

	FontSizeSmall  = "small"
	FontSizeMedium = "medium"
	FontSizeLarge  = "large"
	FontSizeXLarge = "xlarge"
)

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

func (s *SettingsStore) GetFontSize() string {
	return normalizeFontSize(s.prefs().StringWithFallback(prefFontSizeKey, FontSizeMedium))
}

func (s *SettingsStore) SetFontSize(name string) string {
	name = normalizeFontSize(name)
	s.prefs().SetString(prefFontSizeKey, name)
	s.ApplyTheme()
	return name
}

func (s *SettingsStore) ApplyTheme() {
	textSize := fontSizePoints(s.GetFontSize())
	switch s.GetTheme() {
	case ThemeLight:
		fyne.CurrentApp().Settings().SetTheme(uitheme.Light(textSize))
	default:
		fyne.CurrentApp().Settings().SetTheme(uitheme.Dark(textSize))
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

func normalizeFontSize(name string) string {
	switch name {
	case FontSizeSmall, FontSizeLarge, FontSizeXLarge:
		return name
	default:
		return FontSizeMedium
	}
}

func fontSizePoints(name string) float32 {
	switch normalizeFontSize(name) {
	case FontSizeSmall:
		return 12
	case FontSizeLarge:
		return 16
	case FontSizeXLarge:
		return 18
	default:
		return 14
	}
}
