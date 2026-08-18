package collection

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity"
)

type FolderView struct {
	fyne.CanvasObject
	Set       func(name string, auth entity.Auth, showAuth bool)
	Get       func() (name string, auth entity.Auth)
	Save      func()
	SetDirty  func(dirty bool)
	FocusName func()
}

func NewFolderView(onChange func(name string, auth entity.Auth), onSave func(name string, auth entity.Auth)) *FolderView {
	var applying bool
	var header *ui.EntityHeader
	var showAuth bool
	var lastAuth entity.Auth

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
	noAuthHint := widget.NewLabel("Auth is available only for REST collections.")
	noAuthHint.TextStyle = fyne.TextStyle{Italic: true}
	body := container.NewStack(authPanel.CanvasObject, container.NewPadded(noAuthHint))

	header = ui.NewEntityHeader(theme.FolderIcon(), "Folder name", func(string) { notify() }, func() {
		if onSave == nil || get == nil {
			return
		}
		name, auth := get()
		onSave(name, auth)
	})

	get = func() (string, entity.Auth) {
		if showAuth {
			return header.GetName(), authPanel.Get()
		}
		return header.GetName(), lastAuth
	}

	root := container.NewBorder(header.Object, nil, nil, nil, body)

	v := &FolderView{CanvasObject: root}
	v.Set = func(name string, auth entity.Auth, withAuth bool) {
		applying = true
		showAuth = withAuth
		lastAuth = auth
		header.SetName(name)
		if withAuth {
			authPanel.CanvasObject.Show()
			noAuthHint.Hide()
			authPanel.Set(auth)
		} else {
			authPanel.CanvasObject.Hide()
			noAuthHint.Show()
		}
		body.Refresh()
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
