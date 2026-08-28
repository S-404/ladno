package logs

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/entity"
)

// NewDetailView — детали лога целиком (без внутреннего скролла), с переносами и копированием.
func NewDetailView(entry *entity.LogEntry) fyne.CanvasObject {
	original := entry.Detail

	statusBadge := canvas.NewText(
		FormatStatusLabel(entry.StatusCode, entry.IsError, entry.Highlight),
		BadgeColor(entry),
	)
	statusBadge.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	statusBadge.TextSize = theme.TextSize()

	hint := canvas.NewText("  select text to copy  ·", colorMuted)
	hint.TextStyle = fyne.TextStyle{Italic: true}
	hint.TextSize = theme.CaptionTextSize()

	copyBtn := widget.NewButtonWithIcon("Copy", theme.ContentCopyIcon(), func() {
		fyne.CurrentApp().Clipboard().SetContent(original)
	})
	copyBtn.Importance = widget.LowImportance

	toolbar := container.NewBorder(
		nil, nil,
		container.NewHBox(statusBadge, hint),
		copyBtn,
		nil,
	)

	text := widget.NewLabel(original)
	text.Wrapping = fyne.TextWrapBreak
	text.TextStyle = fyne.TextStyle{Monospace: true}
	text.Selectable = true

	return container.NewBorder(toolbar, nil, nil, nil, text)
}
