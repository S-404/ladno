package grpcui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity"
)

type RequestView struct {
	fyne.CanvasObject
	Set     func(req *entity.GrpcRequest, name string, auth entity.Auth)
	GetAuth func() entity.Auth
}

func NewRequestView(onAuthSave func(auth entity.Auth), onNameSave func(name string)) *RequestView {
	title := widget.NewLabel("gRPC request")
	title.TextStyle = fyne.TextStyle{Bold: true}
	nameField := ui.NewRequestNameField(onNameSave)

	target := widget.NewEntry()
	target.SetPlaceHolder("host:port")
	method := widget.NewEntry()
	method.SetPlaceHolder("package.Service/Method")
	message := widget.NewMultiLineEntry()
	message.SetPlaceHolder("JSON message")
	message.SetMinRowsVisible(8)

	meta := ui.NewKVTable(nil, nil)
	authPanel := ui.NewAuthPanel(ui.AuthPanelOptions{
		AllowInherited: true,
		OnSave:         onAuthSave,
	})

	send := widget.NewButton("Send", nil)
	send.Disable()
	hint := widget.NewLabel("gRPC send is not implemented yet")
	hint.TextStyle = fyne.TextStyle{Italic: true}

	requestTab := container.NewBorder(
		container.NewVBox(
			title,
			nameField.Object,
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
		nameField.Set(name)
		authPanel.Set(auth)
		if req == nil {
			target.SetText("")
			method.SetText("")
			message.SetText("")
			meta.SetRows(nil)
			return
		}
		target.SetText(req.Target)
		method.SetText(req.Method)
		message.SetText(req.Message)
		rows := make([]ui.KVRow, 0, len(req.Metadata))
		for _, m := range req.Metadata {
			rows = append(rows, ui.KVRow{Enabled: true, Key: m.Key, Value: m.Value})
		}
		meta.SetRows(rows)
	}
	v.GetAuth = authPanel.Get
	return v
}
