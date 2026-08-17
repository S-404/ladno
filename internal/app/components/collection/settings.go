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
	Name  string
	Auth  entity.Auth
	Nats  *entity.NatsConnection
	Kafka *entity.KafkaConnection
}

type SettingsCallbacks struct {
	OnChange     func(SettingsSave)
	OnSave       func(SettingsSave)
	OnConnect    func(SettingsSave)
	OnDisconnect func()
}

type SettingsView struct {
	fyne.CanvasObject
	Set              func(name string, auth entity.Auth, nats *entity.NatsConnection, kafka *entity.KafkaConnection, colType constants.CollectionType, connected bool)
	Get              func() SettingsSave
	Save             func()
	SetDirty         func(dirty bool)
	SetConnectStatus func(text string)
	SetConnected     func(connected bool)
}

func NewSettingsView(cb SettingsCallbacks) *SettingsView {
	var applying bool
	var getSave func() SettingsSave
	header := ui.NewEntityHeader("Collection", func() {
		if cb.OnSave != nil && getSave != nil {
			cb.OnSave(getSave())
		}
	})

	nameEntry := ui.NewEntry()
	nameEntry.SetPlaceHolder("Collection name")

	typeLabel := widget.NewLabel("")
	typeLabel.TextStyle = fyne.TextStyle{Italic: true}

	hostEntry := ui.NewEnvInput()
	hostEntry.SetPlaceHolder("{{natsHost}} or localhost")
	portEntry := ui.NewEnvInput()
	portEntry.SetPlaceHolder("{{natsPort}} or 4222")
	tokenEntry := ui.NewEnvInput()
	tokenEntry.SetPlaceHolder("{{natsToken}}")
	brokersEntry := ui.NewEnvInput()
	brokersEntry.SetPlaceHolder("{{kafkaBrokers}} or localhost:9092")

	envHint := widget.NewLabel("Supports {{var}} from the active environment.")
	envHint.TextStyle = fyne.TextStyle{Italic: true}
	connStatus := widget.NewLabel("")
	connStatus.TextStyle = fyne.TextStyle{Italic: true}

	var currentType constants.CollectionType
	var connected bool
	var authPanel *ui.AuthPanel
	content := container.NewStack()

	getSave = func() SettingsSave {
		out := SettingsSave{Name: nameEntry.Text}
		switch currentType {
		case constants.CollectionTypeNATS:
			out.Nats = &entity.NatsConnection{
				Host:  hostEntry.Text(),
				Port:  portEntry.Text(),
				Token: tokenEntry.Text(),
			}
			out.Auth = entity.Auth{Type: constants.AuthTypeNoAuth}
		case constants.CollectionTypeKafka:
			out.Kafka = &entity.KafkaConnection{Brokers: brokersEntry.Text()}
			out.Auth = entity.Auth{Type: constants.AuthTypeNoAuth}
		default:
			if authPanel != nil {
				out.Auth = authPanel.Get()
			}
		}
		return out
	}

	notify := func() {
		if applying || cb.OnChange == nil {
			return
		}
		cb.OnChange(getSave())
	}

	authPanel = ui.NewAuthPanel(ui.AuthPanelOptions{
		AllowInherited: false,
		OnChange:       func(entity.Auth) { notify() },
	})

	nameEntry.OnChanged = func(string) { notify() }
	hostEntry.OnChanged(func(string) { notify() })
	portEntry.OnChanged(func(string) { notify() })
	tokenEntry.OnChanged(func(string) { notify() })
	brokersEntry.OnChanged(func(string) { notify() })

	connBtn := widget.NewButton("Connect", nil)
	applyConnBtn := func() {
		if connected {
			connBtn.SetText("Disconnect")
			connBtn.Importance = widget.DangerImportance
			return
		}
		connBtn.SetText("Connect")
		connBtn.Importance = widget.MediumImportance
	}
	connBtn.OnTapped = func() {
		if connected {
			if cb.OnDisconnect != nil {
				cb.OnDisconnect()
			}
			return
		}
		if cb.OnConnect != nil {
			cb.OnConnect(getSave())
		}
	}
	applyConnBtn()

	isConnectable := func() bool {
		return currentType == constants.CollectionTypeNATS || currentType == constants.CollectionTypeKafka
	}

	render := func() {
		content.Objects = nil
		switch currentType {
		case constants.CollectionTypeNATS:
			content.Objects = []fyne.CanvasObject{
				container.NewBorder(header.Object, nil, nil, nil,
					container.NewPadded(container.NewVBox(
						widget.NewForm(
							widget.NewFormItem("Name", nameEntry),
							widget.NewFormItem("Type", typeLabel),
							widget.NewFormItem("Host", hostEntry),
							widget.NewFormItem("Port", portEntry),
							widget.NewFormItem("Token", tokenEntry),
						),
						envHint,
						connBtn,
						connStatus,
					)),
				),
			}
		case constants.CollectionTypeKafka:
			content.Objects = []fyne.CanvasObject{
				container.NewBorder(header.Object, nil, nil, nil,
					container.NewPadded(container.NewVBox(
						widget.NewForm(
							widget.NewFormItem("Name", nameEntry),
							widget.NewFormItem("Type", typeLabel),
							widget.NewFormItem("Brokers", brokersEntry),
						),
						envHint,
						connBtn,
						connStatus,
					)),
				),
			}
		default:
			general := container.NewPadded(container.NewVBox(
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
				container.NewBorder(header.Object, nil, nil, nil, tabs),
			}
		}
		content.Refresh()
	}

	setConnected := func(ok bool) {
		connected = ok
		applyConnBtn()
		connBtn.Refresh()
	}

	v := &SettingsView{CanvasObject: content}
	v.Set = func(name string, auth entity.Auth, nats *entity.NatsConnection, kafka *entity.KafkaConnection, colType constants.CollectionType, isConnected bool) {
		applying = true
		currentType = constants.NormalizeCollectionType(colType)
		nameEntry.SetText(name)
		typeLabel.SetText(string(currentType))
		connStatus.SetText("")
		setConnected(isConnected)
		if isConnected && isConnectable() {
			connStatus.SetText("Connected")
		}
		switch currentType {
		case constants.CollectionTypeNATS:
			if nats == nil {
				nats = &entity.NatsConnection{}
			}
			hostEntry.SetText(nats.Host)
			portEntry.SetText(nats.Port)
			tokenEntry.SetText(nats.Token)
		case constants.CollectionTypeKafka:
			if kafka == nil {
				kafka = &entity.KafkaConnection{}
			}
			brokersEntry.SetText(kafka.Brokers)
		default:
			authPanel.Set(auth)
		}
		applying = false
		render()
	}
	v.Get = getSave
	v.Save = func() {
		if cb.OnSave != nil {
			cb.OnSave(getSave())
		}
	}
	v.SetDirty = header.SetDirty
	v.SetConnectStatus = func(text string) { connStatus.SetText(text) }
	v.SetConnected = func(ok bool) {
		setConnected(ok)
		if ok {
			connStatus.SetText("Connected")
		} else if connStatus.Text == "Connected" || connStatus.Text == "Connecting…" {
			connStatus.SetText("")
		}
	}
	return v
}
