package wsui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity"
)

type RequestView struct {
	fyne.CanvasObject
	Set       func(req *entity.WsRequest, name string)
	SetDirty  func(dirty bool)
	FocusName func()
}

func NewRequestView(onChange func(name string, req entity.WsRequest), onSave func()) *RequestView {
	var applying bool
	var header *ui.EntityHeader

	urlEntry := ui.NewEntry()
	urlEntry.SetPlaceHolder("ws://host/path")
	message := ui.NewMultiLineEntry()
	message.SetPlaceHolder("Message")
	message.SetMinRowsVisible(8)

	var headers *ui.KVTable
	var getReq func() entity.WsRequest

	notify := func() {
		if applying || onChange == nil || header == nil || getReq == nil {
			return
		}
		onChange(header.GetName(), getReq())
	}

	header = ui.NewEntityHeader(theme.DocumentIcon(), "Request name", func(string) { notify() }, onSave)

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
	urlEntry.OnChanged = func(string) { notify() }
	message.OnChanged = func(string) { notify() }

	connect := widget.NewButton("Connect", nil)
	connect.Disable()
	send := widget.NewButton("Send", nil)
	send.Disable()
	hint := widget.NewLabel("WebSocket is not implemented yet")
	hint.TextStyle = fyne.TextStyle{Italic: true}

	root := container.NewBorder(
		container.NewVBox(
			header.Object,
			widget.NewForm(widget.NewFormItem("URL", urlEntry)),
			widget.NewLabel("Headers"),
			headers,
			widget.NewLabel("Message"),
		),
		container.NewVBox(container.NewHBox(connect, send), hint),
		nil, nil,
		message,
	)

	v := &RequestView{CanvasObject: container.NewPadded(root)}
	v.Set = func(req *entity.WsRequest, name string) {
		applying = true
		header.SetName(name)
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
	v.FocusName = header.FocusName
	return v
}
