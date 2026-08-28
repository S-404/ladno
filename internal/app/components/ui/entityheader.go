package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// EntityHeader — [иконка типа] [имя] [сохранить].
type EntityHeader struct {
	Object    fyne.CanvasObject
	SetDirty  func(dirty bool)
	SetName   func(name string)
	GetName   func() string
	SetIcon   func(res fyne.Resource)
	FocusName func()
}

func NewEntityHeader(icon fyne.Resource, placeholder string, onNameChange func(string), onSave func()) *EntityHeader {
	iconW := widget.NewIcon(icon)
	nameField := NewEditableName(placeholder, onNameChange)

	saveBtn := widget.NewButtonWithIcon("", theme.DocumentSaveIcon(), func() {
		if onSave != nil {
			onSave()
		}
	})
	saveBtn.Importance = widget.LowImportance
	saveBtn.Disable()

	root := container.NewBorder(nil, nil, iconW, saveBtn, nameField.Object)
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
		SetName:   nameField.Set,
		GetName:   nameField.Get,
		FocusName: nameField.FocusEdit,
		SetIcon: func(res fyne.Resource) {
			iconW.SetResource(res)
		},
	}
}

// NewTitledEntityHeader — [заголовок] [сохранить] (без редактируемого имени).
func NewTitledEntityHeader(title string, onSave func()) *EntityHeader {
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
		SetName:   func(string) {},
		GetName:   func() string { return "" },
		FocusName: func() {},
		SetIcon:   func(fyne.Resource) {},
	}
}
