package natsui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity"
)

type RequestView struct {
	fyne.CanvasObject
	Set     func(req *entity.NatsRequest, name string, auth entity.Auth)
	GetAuth func() entity.Auth
}

func NewRequestView(onAuthSave func(auth entity.Auth)) *RequestView {
	title := widget.NewLabel("NATS request")
	title.TextStyle = fyne.TextStyle{Bold: true}
	nameLabel := widget.NewLabel("")

	subject := widget.NewEntry()
	subject.SetPlaceHolder("subject")
	payload := widget.NewMultiLineEntry()
	payload.SetPlaceHolder("Payload")
	payload.SetMinRowsVisible(8)

	headers := ui.NewKVTable(nil, nil)
	authPanel := ui.NewAuthPanel(ui.AuthPanelOptions{
		AllowInherited: true,
		OnSave:         onAuthSave,
	})

	publish := widget.NewButton("Publish", nil)
	publish.Disable()
	hint := widget.NewLabel("NATS is not implemented yet")
	hint.TextStyle = fyne.TextStyle{Italic: true}

	requestTab := container.NewBorder(
		container.NewVBox(
			title,
			nameLabel,
			widget.NewForm(
				widget.NewFormItem("Subject", subject),
			),
			widget.NewLabel("Headers"),
			headers,
			widget.NewLabel("Payload"),
		),
		container.NewVBox(publish, hint),
		nil, nil,
		payload,
	)

	tabs := container.NewAppTabs(
		container.NewTabItem("Request", container.NewPadded(requestTab)),
		container.NewTabItem("Auth", authPanel.CanvasObject),
	)

	v := &RequestView{CanvasObject: tabs}
	v.Set = func(req *entity.NatsRequest, name string, auth entity.Auth) {
		nameLabel.SetText(name)
		authPanel.Set(auth)
		if req == nil {
			subject.SetText("")
			payload.SetText("")
			headers.SetRows(nil)
			return
		}
		subject.SetText(req.Subject)
		payload.SetText(req.Payload)
		rows := make([]ui.KVRow, 0, len(req.Headers))
		for _, h := range req.Headers {
			rows = append(rows, ui.KVRow{Enabled: true, Key: h.Key, Value: h.Value})
		}
		headers.SetRows(rows)
	}
	v.GetAuth = authPanel.Get
	return v
}
