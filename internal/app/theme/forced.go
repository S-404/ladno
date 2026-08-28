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
	Variant  fyne.ThemeVariant
	TextSize float32 // 0 keeps default theme text sizes
}

func (f *ForcedVariant) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	return f.Theme.Color(name, f.Variant)
}

func (f *ForcedVariant) Size(name fyne.ThemeSizeName) float32 {
	base := f.Theme.Size(name)
	if f.TextSize <= 0 {
		return base
	}
	switch name {
	case theme.SizeNameText, theme.SizeNameHeadingText, theme.SizeNameSubHeadingText, theme.SizeNameCaptionText:
		defText := f.Theme.Size(theme.SizeNameText)
		if defText <= 0 {
			return f.TextSize
		}
		return base * (f.TextSize / defText)
	default:
		return base
	}
}

func Light(textSize float32) fyne.Theme {
	return &ForcedVariant{Theme: theme.DefaultTheme(), Variant: theme.VariantLight, TextSize: textSize}
}

func Dark(textSize float32) fyne.Theme {
	return &ForcedVariant{Theme: theme.DefaultTheme(), Variant: theme.VariantDark, TextSize: textSize}
}
