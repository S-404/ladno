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
	Set      func(req *entity.WsRequest, name string, auth entity.Auth)
	SetDirty func(dirty bool)
}

func NewRequestView(onChange func(name string, req entity.WsRequest, auth entity.Auth), onSave func()) *RequestView {
	var applying bool
	header := ui.NewEntityHeader("WebSocket request", onSave)

	urlEntry := ui.NewEntry()
	urlEntry.SetPlaceHolder("ws://host/path")
	message := ui.NewMultiLineEntry()
	message.SetPlaceHolder("Message")
	message.SetMinRowsVisible(8)

	var nameField *ui.RequestNameField
	var headers *ui.KVTable
	var authPanel *ui.AuthPanel
	var getReq func() entity.WsRequest

	notify := func() {
		if applying || onChange == nil || nameField == nil || authPanel == nil || getReq == nil {
			return
		}
		onChange(nameField.Get(), getReq(), authPanel.Get())
	}
	headers = ui.NewKVTable(nil, func([]ui.KVRow) { notify() })
	getReq = func() entity.WsRequest {
		rows := headers.GetRows()
		vars := make([]entity.Variable, 0, len(rows))
		for _, r := range rows {
			if r.Key == "" {
				continue
			}
			vars = append(vars, entity.Variable{Key: r.Key, Value: r.Value, Type: "string"})
		}
		return entity.WsRequest{URL: urlEntry.Text, Message: message.Text, Headers: vars}
	}
	nameField = ui.NewRequestNameField(func(string) { notify() })
	authPanel = ui.NewAuthPanel(ui.AuthPanelOptions{AllowInherited: true, OnChange: func(entity.Auth) { notify() }})
	urlEntry.OnChanged = func(string) { notify() }
	message.OnChanged = func(string) { notify() }

	connect := widget.NewButton("Connect", nil)
	connect.Disable()
	send := widget.NewButton("Send", nil)
	send.Disable()
	hint := widget.NewLabel("WebSocket is not implemented yet")
	hint.TextStyle = fyne.TextStyle{Italic: true}

	requestTab := container.NewBorder(
		container.NewVBox(
			header.Object,
			nameField.Object,
			widget.NewForm(widget.NewFormItem("URL", urlEntry)),
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
		applying = true
		nameField.Set(name)
		authPanel.Set(auth)
		if req == nil {
			urlEntry.SetText("")
			message.SetText("")
			headers.SetRows(nil)
		} else {
			urlEntry.SetText(req.URL)
			message.SetText(req.Message)
			rows := make([]ui.KVRow, 0, len(req.Headers))
			for _, h := range req.Headers {
				rows = append(rows, ui.KVRow{Enabled: true, Key: h.Key, Value: h.Value})
			}
			headers.SetRows(rows)
		}
		applying = false
	}
	v.SetDirty = header.SetDirty
	return v
}
