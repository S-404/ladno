package collection

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity"
)

type FolderView struct {
	fyne.CanvasObject
	Set       func(name string, auth entity.Auth)
	Get       func() (name string, auth entity.Auth)
	Save      func()
	SetDirty  func(dirty bool)
	FocusName func()
}

func NewFolderView(onChange func(name string, auth entity.Auth), onSave func(name string, auth entity.Auth)) *FolderView {
	var applying bool
	var header *ui.EntityHeader

	var get func() (string, entity.Auth)

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

	header = ui.NewEntityHeader(theme.FolderIcon(), "Folder name", func(string) { notify() }, func() {
		if onSave == nil || get == nil {
			return
		}
		name, auth := get()
		onSave(name, auth)
	})

	get = func() (string, entity.Auth) {
		return header.GetName(), authPanel.Get()
	}

	root := container.NewBorder(header.Object, nil, nil, nil, authPanel.CanvasObject)

	v := &FolderView{CanvasObject: root}
	v.Set = func(name string, auth entity.Auth) {
		applying = true
		header.SetName(name)
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
	v.FocusName = header.FocusName
	return v
}
