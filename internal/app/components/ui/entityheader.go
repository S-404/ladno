package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// EntityHeader — заголовок типа сущности + Save (активна только когда dirty).
type EntityHeader struct {
	Object   fyne.CanvasObject
	SetDirty func(dirty bool)
	SetTitle func(title string)
}

func NewEntityHeader(title string, onSave func()) *EntityHeader {
	label := widget.NewLabel(title)
	label.TextStyle = fyne.TextStyle{Bold: true}

	saveBtn := widget.NewButtonWithIcon("", theme.DocumentSaveIcon(), func() {
		if onSave != nil {
			onSave()
		}
	})
	saveBtn.Importance = widget.LowImportance
	saveBtn.Disable()

	root := container.NewBorder(nil, nil, label, saveBtn, nil)
	return &EntityHeader{
		Object: root,
		SetDirty: func(dirty bool) {
			if dirty {
				saveBtn.Enable()
				saveBtn.Importance = widget.HighImportance
			} else {
				saveBtn.Disable()
				saveBtn.Importance = widget.LowImportance
			}
			saveBtn.Refresh()
		},
		SetTitle: func(t string) {
			label.SetText(t)
		},
	}
}
