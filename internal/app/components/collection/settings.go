package collection

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
)

type SettingsSave struct {
	Name string
	Auth entity.Auth
	Nats *entity.NatsConnection
}

type SettingsCallbacks struct {
	OnSave    func(SettingsSave)
	OnConnect func(SettingsSave)
}

type SettingsView struct {
	fyne.CanvasObject
	Set              func(name string, auth entity.Auth, nats *entity.NatsConnection, colType constants.CollectionType)
	Get              func() SettingsSave
	Save             func()
	SetConnectStatus func(text string)
}

func NewSettingsView(cb SettingsCallbacks) *SettingsView {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Collection name")

	typeLabel := widget.NewLabel("")
	typeLabel.TextStyle = fyne.TextStyle{Italic: true}

	authPanel := ui.NewAuthPanel(ui.AuthPanelOptions{AllowInherited: false})

	hostEntry := widget.NewEntry()
	hostEntry.SetPlaceHolder("localhost")
	portEntry := widget.NewEntry()
	portEntry.SetPlaceHolder("4222")
	tokenEntry := widget.NewPasswordEntry()
	tokenEntry.SetPlaceHolder("token")

	connStatus := widget.NewLabel("")
	connStatus.TextStyle = fyne.TextStyle{Italic: true}

	var currentType constants.CollectionType
	content := container.NewStack()

	getSave := func() SettingsSave {
		out := SettingsSave{Name: nameEntry.Text}
		if currentType == constants.CollectionTypeNATS {
			out.Nats = &entity.NatsConnection{
				Host:  hostEntry.Text,
				Port:  portEntry.Text,
				Token: tokenEntry.Text,
			}
			out.Auth = entity.Auth{Type: constants.AuthTypeNoAuth}
		} else {
			out.Auth = authPanel.Get()
		}
		return out
	}

	doSave := func() {
		if cb.OnSave != nil {
			cb.OnSave(getSave())
		}
	}

	saveBtn := widget.NewButton("Save", doSave)
	saveBtn.Importance = widget.HighImportance

	connectBtn := widget.NewButton("Connect", func() {
		if cb.OnConnect != nil {
			cb.OnConnect(getSave())
		}
	})

	render := func() {
		content.Objects = nil
		if currentType == constants.CollectionTypeNATS {
			content.Objects = []fyne.CanvasObject{
				container.NewPadded(container.NewVBox(
					widget.NewLabel("Collection"),
					widget.NewForm(
						widget.NewFormItem("Name", nameEntry),
						widget.NewFormItem("Type", typeLabel),
						widget.NewFormItem("Host", hostEntry),
						widget.NewFormItem("Port", portEntry),
						widget.NewFormItem("Token", tokenEntry),
					),
					container.NewHBox(saveBtn, connectBtn),
					connStatus,
				)),
			}
		} else {
			general := container.NewPadded(container.NewVBox(
				widget.NewLabel("Collection"),
				widget.NewForm(
					widget.NewFormItem("Name", nameEntry),
					widget.NewFormItem("Type", typeLabel),
				),
			))
			tabs := container.NewAppTabs(
				container.NewTabItem("General", general),
				container.NewTabItem("Auth", authPanel.CanvasObject),
			)
			content.Objects = []fyne.CanvasObject{
				container.NewBorder(nil, container.NewPadded(saveBtn), nil, nil, tabs),
			}
		}
		content.Refresh()
	}

	v := &SettingsView{CanvasObject: content}
	v.Set = func(name string, auth entity.Auth, nats *entity.NatsConnection, colType constants.CollectionType) {
		currentType = constants.NormalizeCollectionType(colType)
		nameEntry.SetText(name)
		typeLabel.SetText(string(currentType))
		connStatus.SetText("")

		if currentType == constants.CollectionTypeNATS {
			if nats == nil {
				nats = &entity.NatsConnection{}
			}
			hostEntry.SetText(nats.Host)
			portEntry.SetText(nats.Port)
			tokenEntry.SetText(nats.Token)
		} else {
			authPanel.Set(auth)
		}
		render()
	}
	v.Get = getSave
	v.Save = doSave
	v.SetConnectStatus = func(text string) {
		connStatus.SetText(text)
	}
	return v
}
