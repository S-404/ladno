package natsui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
)

type RequestView struct {
	fyne.CanvasObject
	Set          func(req *entity.NatsRequest, name string, subscribed bool)
	Get          func() entity.NatsRequest
	SetRunning   func(running bool)
	SetSubActive func(active bool)
	SetDirty     func(dirty bool)
	Messages     *MessagesView
	Subject      func() string
}

func NewRequestView(
	onRun func(method constants.NatsMethod, req entity.NatsRequest),
	onUnsub func(),
	onChange func(name string, req entity.NatsRequest),
	onSave func(),
	messages *MessagesView,
) *RequestView {
	var applying bool
	header := ui.NewEntityHeader("NATS request", onSave)
	statusLabel := widget.NewLabel("")
	statusLabel.TextStyle = fyne.TextStyle{Italic: true}

	subject := ui.NewEnvInput()
	subject.SetPlaceHolder("{{natsSubject}} or demo.events")
	payload := ui.NewEnvMultiLineInput()
	payload.SetPlaceHolder(`{"ok": true, "token": "{{token}}"}`)
	payload.SetMinRowsVisible(6)

	var nameField *ui.RequestNameField
	var headers *ui.KVTable
	var getReq func() entity.NatsRequest

	notify := func() {
		if applying || onChange == nil || nameField == nil || getReq == nil {
			return
		}
		onChange(nameField.Get(), getReq())
	}
	headers = ui.NewKVTable(nil, func([]ui.KVRow) { notify() })
	getReq = func() entity.NatsRequest {
		rows := headers.GetRows()
		vars := make([]entity.Variable, 0, len(rows))
		for _, r := range rows {
			if r.Key == "" {
				continue
			}
			vars = append(vars, entity.Variable{Key: r.Key, Value: r.Value, Type: "string"})
		}
		return entity.NatsRequest{Subject: subject.Text(), Headers: vars, Payload: payload.Text()}
	}
	nameField = ui.NewRequestNameField(func(string) { notify() })
	subject.OnChanged(func(string) { notify() })
	payload.OnChanged(func(string) { notify() })

	envHint := widget.NewLabel("Subject, headers and payload support {{var}} from the active environment.")
	envHint.TextStyle = fyne.TextStyle{Italic: true}

	publishBtn := widget.NewButton("Publish", func() {
		if onRun != nil {
			onRun(constants.NatsMethodPublish, getReq())
		}
	})
	requestBtn := widget.NewButton("Request", func() {
		if onRun != nil {
			onRun(constants.NatsMethodRequest, getReq())
		}
	})
	publishBtn.Importance = widget.HighImportance

	var subscribed bool
	var subBtn *widget.Button
	applySubBtn := func() {
		if subscribed {
			subBtn.SetText("Unsubscribe")
			subBtn.Importance = widget.DangerImportance
			return
		}
		subBtn.SetText("Subscribe")
		subBtn.Importance = widget.MediumImportance
	}
	subBtn = widget.NewButton("Subscribe", func() {
		if subscribed {
			if onUnsub != nil {
				onUnsub()
			}
			return
		}
		if onRun != nil {
			onRun(constants.NatsMethodSubscribe, getReq())
		}
	})
	applySubBtn()

	actions := container.NewBorder(nil, nil, container.NewHBox(publishBtn, requestBtn, subBtn), statusLabel, nil)
	requestPanel := container.NewBorder(
		container.NewVBox(
			header.Object,
			nameField.Object,
			widget.NewForm(widget.NewFormItem("Subject", subject)),
			widget.NewLabel("Headers"),
			headers,
			widget.NewLabel("Payload"),
			envHint,
		),
		actions, nil, nil, payload,
	)
	split := container.NewVSplit(
		ui.NewMinSizeBox(fyne.NewSize(200, 80), container.NewPadded(requestPanel)),
		ui.NewMinSizeBox(fyne.NewSize(200, 80), messages.Object()),
	)
	split.SetOffset(0.55)

	v := &RequestView{CanvasObject: split, Messages: messages, Subject: func() string { return subject.Text() }}
	v.Get = getReq
	v.SetDirty = header.SetDirty
	v.SetRunning = func(running bool) {
		if running {
			publishBtn.Disable()
			requestBtn.Disable()
			subBtn.Disable()
			statusLabel.SetText("…")
			return
		}
		publishBtn.Enable()
		requestBtn.Enable()
		subBtn.Enable()
	}
	v.SetSubActive = func(active bool) {
		subscribed = active
		applySubBtn()
		subBtn.Refresh()
		if active {
			statusLabel.SetText("Subscribed")
			return
		}
		statusLabel.SetText("")
	}
	v.Set = func(req *entity.NatsRequest, name string, subscribed bool) {
		applying = true
		nameField.Set(name)
		statusLabel.SetText("")
		if req == nil {
			subject.SetText("")
			payload.SetText("")
			headers.SetRows(nil)
			v.SetSubActive(false)
		} else {
			subject.SetText(req.Subject)
			payload.SetText(req.Payload)
			rows := make([]ui.KVRow, 0, len(req.Headers))
			for _, h := range req.Headers {
				rows = append(rows, ui.KVRow{Enabled: true, Key: h.Key, Value: h.Value})
			}
			headers.SetRows(rows)
			v.SetSubActive(subscribed)
		}
		applying = false
	}
	return v
}
