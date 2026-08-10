package wsui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity"
)

type RequestView struct {
	fyne.CanvasObject
	Set     func(req *entity.WsRequest, name string, auth entity.Auth)
	GetAuth func() entity.Auth
}

func NewRequestView(onAuthSave func(auth entity.Auth)) *RequestView {
	title := widget.NewLabel("WebSocket request")
	title.TextStyle = fyne.TextStyle{Bold: true}
	nameLabel := widget.NewLabel("")

	urlEntry := widget.NewEntry()
	urlEntry.SetPlaceHolder("ws://host/path")
	message := widget.NewMultiLineEntry()
	message.SetPlaceHolder("Message")
	message.SetMinRowsVisible(8)

	headers := ui.NewKVTable(nil, nil)
	authPanel := ui.NewAuthPanel(ui.AuthPanelOptions{
		AllowInherited: true,
		OnSave:         onAuthSave,
	})

	connect := widget.NewButton("Connect", nil)
	connect.Disable()
	send := widget.NewButton("Send", nil)
	send.Disable()
	hint := widget.NewLabel("WebSocket is not implemented yet")
	hint.TextStyle = fyne.TextStyle{Italic: true}

	requestTab := container.NewBorder(
		container.NewVBox(
			title,
			nameLabel,
			widget.NewForm(
				widget.NewFormItem("URL", urlEntry),
			),
			widget.NewLabel("Headers"),
			headers,
			widget.NewLabel("Message"),
		),
		container.NewVBox(container.NewHBox(connect, send), hint),
		nil, nil,
		message,
	)

	tabs := container.NewAppTabs(
		container.NewTabItem("Request", container.NewPadded(requestTab)),
		container.NewTabItem("Auth", authPanel.CanvasObject),
	)

	v := &RequestView{CanvasObject: tabs}
	v.Set = func(req *entity.WsRequest, name string, auth entity.Auth) {
		nameLabel.SetText(name)
		authPanel.Set(auth)
		if req == nil {
			urlEntry.SetText("")
			message.SetText("")
			headers.SetRows(nil)
			return
		}
		urlEntry.SetText(req.URL)
		message.SetText(req.Message)
		rows := make([]ui.KVRow, 0, len(req.Headers))
		for _, h := range req.Headers {
			rows = append(rows, ui.KVRow{Enabled: true, Key: h.Key, Value: h.Value})
		}
		headers.SetRows(rows)
	}
	v.GetAuth = authPanel.Get
	return v
}
