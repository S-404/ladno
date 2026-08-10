package uitheme

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// ForcedVariant wraps a theme and always uses a fixed light/dark variant.
// Pattern from fyne_demo (DarkTheme/LightTheme are deprecated).
type ForcedVariant struct {
	fyne.Theme
	Variant fyne.ThemeVariant
}

func (f *ForcedVariant) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	return f.Theme.Color(name, f.Variant)
}

func Light() fyne.Theme {
	return &ForcedVariant{Theme: theme.DefaultTheme(), Variant: theme.VariantLight}
}

func Dark() fyne.Theme {
	return &ForcedVariant{Theme: theme.DefaultTheme(), Variant: theme.VariantDark}
}
