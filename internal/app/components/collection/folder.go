package collection

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
)

type FolderView struct {
	fyne.CanvasObject
	Set       func(name string, auth entity.Auth, colType constants.CollectionType)
	Get       func() (name string, auth entity.Auth)
	Save      func()
	SetDirty  func(dirty bool)
	FocusName func()
}

func NewFolderView(onChange func(name string, auth entity.Auth), onSave func(name string, auth entity.Auth)) *FolderView {
	var applying bool
	var header *ui.EntityHeader
	var currentType constants.CollectionType
	var lastAuth entity.Auth
	var authPanelFor func(t constants.CollectionType) *ui.AuthPanel

	var get func() (string, entity.Auth)

	notify := func() {
		if applying || onChange == nil || get == nil {
			return
		}
		name, auth := get()
		onChange(name, auth)
	}

	authNotify := func(entity.Auth) { notify() }
	applying = true
	restAuthPanel := ui.NewAuthPanel(ui.AuthPanelOptions{
		AllowInherited: true,
		TypeLabel:      "Auth",
		OnChange:       authNotify,
	})
	grpcAuthPanel := ui.NewAuthPanel(ui.AuthPanelOptions{
		AllowInherited: true,
		DisableAPIKey:  true,
		TypeLabel:      "Auth",
		OnChange:       authNotify,
	})
	sioAuthPanel := ui.NewAuthPanel(ui.AuthPanelOptions{
		AllowInherited:   true,
		DisableBasic:     true,
		AllowJSON:        true,
		APIKeyHeaderOnly: true,
		TypeLabel:        "Auth",
		OnChange:         authNotify,
	})
	authPanelFor = func(t constants.CollectionType) *ui.AuthPanel {
		switch constants.NormalizeCollectionType(t) {
		case constants.CollectionTypeGRPC:
			return grpcAuthPanel
		case constants.CollectionTypeSocketIO:
			return sioAuthPanel
		default:
			return restAuthPanel
		}
	}
	grpcAuthPanel.CanvasObject.Hide()
	sioAuthPanel.CanvasObject.Hide()
	applying = false

	noAuthHint := widget.NewLabel("Auth is available only for REST, WebSocket, Socket.IO, and gRPC collections.")
	noAuthHint.TextStyle = fyne.TextStyle{Italic: true}

	allPanels := []*ui.AuthPanel{restAuthPanel, grpcAuthPanel, sioAuthPanel}
	bodyObjs := make([]fyne.CanvasObject, 0, len(allPanels)+1)
	for _, p := range allPanels {
		bodyObjs = append(bodyObjs, p.CanvasObject)
	}
	bodyObjs = append(bodyObjs, container.NewPadded(noAuthHint))
	body := container.NewStack(bodyObjs...)

	header = ui.NewEntityHeader(theme.FolderIcon(), "Folder name", func(string) { notify() }, func() {
		if onSave == nil || get == nil {
			return
		}
		name, auth := get()
		onSave(name, auth)
	})

	get = func() (string, entity.Auth) {
		if constants.IsHTTPCollection(currentType) && authPanelFor != nil {
			return header.GetName(), authPanelFor(currentType).Get()
		}
		return header.GetName(), lastAuth
	}

	showAuthPanel := func(t constants.CollectionType) {
		want := authPanelFor(t)
		for _, p := range allPanels {
			if p == want {
				p.CanvasObject.Show()
			} else {
				p.CanvasObject.Hide()
			}
		}
	}

	root := container.NewBorder(header.Object, nil, nil, nil, body)

	v := &FolderView{CanvasObject: root}
	v.Set = func(name string, auth entity.Auth, colType constants.CollectionType) {
		applying = true
		currentType = constants.NormalizeCollectionType(colType)
		lastAuth = auth
		header.SetName(name)
		if constants.IsHTTPCollection(currentType) {
			noAuthHint.Hide()
			showAuthPanel(currentType)
			authPanelFor(currentType).Set(auth)
		} else {
			for _, p := range allPanels {
				p.CanvasObject.Hide()
			}
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
