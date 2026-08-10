package rest

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type PreviewView struct {
	Object  fyne.CanvasObject
	SetText func(text string)
	Refresh func()
}

func NewPreviewView(onRefresh func() string) *PreviewView {
	text := widget.NewLabel("")
	text.Wrapping = fyne.TextWrapBreak
	text.TextStyle = fyne.TextStyle{Monospace: true}
	text.Selectable = true

	var current string
	setText := func(v string) {
		current = v
		text.SetText(v)
	}

	copyBtn := widget.NewButtonWithIcon("Copy", theme.ContentCopyIcon(), func() {
		fyne.CurrentApp().Clipboard().SetContent(current)
	})
	copyBtn.Importance = widget.LowImportance

	refreshBtn := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), nil)
	refreshBtn.Importance = widget.LowImportance

	doRefresh := func() {
		if onRefresh == nil {
			return
		}
		setText(onRefresh())
	}
	refreshBtn.OnTapped = doRefresh

	toolbar := container.NewBorder(
		nil, nil,
		widget.NewLabel("Preview (resolved request)"),
		container.NewHBox(refreshBtn, copyBtn),
		nil,
	)

	scroll := container.NewVScroll(container.NewPadded(text))

	return &PreviewView{
		Object:  container.NewBorder(toolbar, nil, nil, nil, scroll),
		SetText: setText,
		Refresh: doRefresh,
	}
}
