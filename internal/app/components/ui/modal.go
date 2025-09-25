package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

func NewModal(title string, dismiss string, content fyne.CanvasObject, window fyne.Window) dialog.CustomDialog {
	dlg := dialog.NewCustom(title, dismiss, content, window)
	dlg.Resize(fyne.NewSize(300, 300))
	return *dlg
}
