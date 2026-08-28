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

func NewPreviewView(onRefresh func(showSecrets bool) string) *PreviewView {
	text := widget.NewLabel("")
	text.Wrapping = fyne.TextWrapBreak
	text.TextStyle = fyne.TextStyle{Monospace: true}
	text.Selectable = true

	showSecrets := false
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

	eyeBtn := widget.NewButtonWithIcon("", theme.VisibilityOffIcon(), nil)
	eyeBtn.Importance = widget.LowImportance

	doRefresh := func() {
		if onRefresh == nil {
			return
		}
		setText(onRefresh(showSecrets))
	}
	refreshBtn.OnTapped = doRefresh
	eyeBtn.OnTapped = func() {
		showSecrets = !showSecrets
		if showSecrets {
			eyeBtn.SetIcon(theme.VisibilityIcon())
		} else {
			eyeBtn.SetIcon(theme.VisibilityOffIcon())
		}
		eyeBtn.Refresh()
		doRefresh()
	}

	toolbar := container.NewBorder(
		nil, nil,
		widget.NewLabel("Preview (resolved request)"),
		container.NewHBox(eyeBtn, refreshBtn, copyBtn),
		nil,
	)

	scroll := container.NewVScroll(container.NewPadded(text))

	return &PreviewView{
		Object:  container.NewBorder(toolbar, nil, nil, nil, scroll),
		SetText: setText,
		Refresh: doRefresh,
	}
}
