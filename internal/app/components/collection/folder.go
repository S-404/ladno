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
	Set      func(name string, auth entity.Auth)
	Get      func() (name string, auth entity.Auth)
	Save     func()
	SetDirty func(dirty bool)
}

func NewFolderView(onChange func(name string, auth entity.Auth), onSave func(name string, auth entity.Auth)) *FolderView {
	var applying bool
	nameEntry := ui.NewEntry()
	nameEntry.SetPlaceHolder("Folder name")

	var get func() (string, entity.Auth)
	header := ui.NewEntityHeader("Folder", func() {
		if onSave == nil || get == nil {
			return
		}
		name, auth := get()
		onSave(name, auth)
	})

	notify := func() {
		if applying || onChange == nil || get == nil {
			return
		}
		name, auth := get()
		onChange(name, auth)
	}

	authPanel := ui.NewAuthPanel(ui.AuthPanelOptions{
		AllowInherited: true,
		OnChange:       func(entity.Auth) { notify() },
	})
	nameEntry.OnChanged = func(string) { notify() }

	get = func() (string, entity.Auth) {
		return nameEntry.Text, authPanel.Get()
	}

	general := container.NewPadded(container.NewVBox(
		widget.NewForm(widget.NewFormItem("Name", nameEntry)),
	))
	tabs := container.NewAppTabs(
		container.NewTabItem("General", general),
		container.NewTabItem("Auth", authPanel.CanvasObject),
	)
	root := container.NewBorder(header.Object, nil, nil, nil, tabs)

	v := &FolderView{CanvasObject: root}
	v.Set = func(name string, auth entity.Auth) {
		applying = true
		nameEntry.SetText(name)
		authPanel.Set(auth)
		applying = false
	}
	v.Get = get
	v.Save = func() {
		if onSave != nil {
			name, auth := get()
			onSave(name, auth)
		}
	}
	v.SetDirty = header.SetDirty
	return v
}
