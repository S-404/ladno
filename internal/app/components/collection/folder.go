package collection

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity"
)

type FolderView struct {
	fyne.CanvasObject
	Set  func(name string, auth entity.Auth)
	Get  func() (name string, auth entity.Auth)
	Save func()
}

func NewFolderView(onSave func(name string, auth entity.Auth)) *FolderView {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Folder name")

	authPanel := ui.NewAuthPanel(ui.AuthPanelOptions{AllowInherited: true})

	saveBtn := widget.NewButton("Save", func() {
		if onSave != nil {
			onSave(nameEntry.Text, authPanel.Get())
		}
	})
	saveBtn.Importance = widget.HighImportance

	general := container.NewPadded(container.NewVBox(
		widget.NewLabel("Folder"),
		widget.NewForm(
			widget.NewFormItem("Name", nameEntry),
		),
	))

	tabs := container.NewAppTabs(
		container.NewTabItem("General", general),
		container.NewTabItem("Auth", authPanel.CanvasObject),
	)

	root := container.NewBorder(nil, container.NewPadded(saveBtn), nil, nil, tabs)

	v := &FolderView{CanvasObject: root}
	v.Set = func(name string, auth entity.Auth) {
		nameEntry.SetText(name)
		authPanel.Set(auth)
	}
	v.Get = func() (string, entity.Auth) {
		return nameEntry.Text, authPanel.Get()
	}
	v.Save = func() { saveBtn.OnTapped() }
	return v
}
