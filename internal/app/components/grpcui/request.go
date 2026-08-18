package grpcui

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
	Set      func(req *entity.GrpcRequest, name string, auth entity.Auth)
	SetDirty func(dirty bool)
}

func NewRequestView(onChange func(name string, req entity.GrpcRequest, auth entity.Auth), onSave func()) *RequestView {
	var applying bool
	var header *ui.EntityHeader

	target := ui.NewEntry()
	target.SetPlaceHolder("host:port")
	method := ui.NewEntry()
	method.SetPlaceHolder("package.Service/Method")
	message := ui.NewMultiLineEntry()
	message.SetPlaceHolder("JSON message")
	message.SetMinRowsVisible(8)

	var meta *ui.KVTable
	var authPanel *ui.AuthPanel
	var getReq func() entity.GrpcRequest

	notify := func() {
		if applying || onChange == nil || header == nil || authPanel == nil || getReq == nil {
			return
		}
		onChange(header.GetName(), getReq(), authPanel.Get())
	}

	header = ui.NewEntityHeader(theme.DocumentIcon(), "Request name", func(string) { notify() }, onSave)

	meta = ui.NewKVTable(nil, func([]ui.KVRow) { notify() })
	getReq = func() entity.GrpcRequest {
		rows := meta.GetRows()
		vars := make([]entity.Variable, 0, len(rows))
		for _, r := range rows {
			if r.Key == "" {
				continue
			}
			vars = append(vars, entity.Variable{Key: r.Key, Value: r.Value, Type: "string"})
		}
		return entity.GrpcRequest{
			Target: target.Text, Method: method.Text, Message: message.Text, Metadata: vars,
		}
	}
	authPanel = ui.NewAuthPanel(ui.AuthPanelOptions{AllowInherited: true, OnChange: func(entity.Auth) { notify() }})
	target.OnChanged = func(string) { notify() }
	method.OnChanged = func(string) { notify() }
	message.OnChanged = func(string) { notify() }

	send := widget.NewButton("Send", nil)
	send.Disable()
	hint := widget.NewLabel("gRPC send is not implemented yet")
	hint.TextStyle = fyne.TextStyle{Italic: true}

	requestTab := container.NewBorder(
		container.NewVBox(
			header.Object,
			widget.NewForm(
				widget.NewFormItem("Target", target),
				widget.NewFormItem("Method", method),
			),
			widget.NewLabel("Metadata"),
			meta,
			widget.NewLabel("Message"),
		),
		container.NewVBox(send, hint),
		nil, nil,
		message,
	)
	tabs := container.NewAppTabs(
		container.NewTabItem("Request", container.NewPadded(requestTab)),
		container.NewTabItem("Auth", authPanel.CanvasObject),
	)

	v := &RequestView{CanvasObject: tabs}
	v.Set = func(req *entity.GrpcRequest, name string, auth entity.Auth) {
		applying = true
		header.SetName(name)
		authPanel.Set(auth)
		if req == nil {
			target.SetText("")
			method.SetText("")
			message.SetText("")
			meta.SetRows(nil)
		} else {
			target.SetText(req.Target)
			method.SetText(req.Method)
			message.SetText(req.Message)
			rows := make([]ui.KVRow, 0, len(req.Metadata))
			for _, m := range req.Metadata {
				rows = append(rows, ui.KVRow{Enabled: true, Key: m.Key, Value: m.Value})
			}
			meta.SetRows(rows)
		}
		applying = false
	}
	v.SetDirty = header.SetDirty
	return v
}
