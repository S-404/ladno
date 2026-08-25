package natsui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/scripttab"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
)

type RequestView struct {
	fyne.CanvasObject
	Set            func(req *entity.NatsRequest, name string, subscribed bool, event entity.Event)
	Get            func() entity.NatsRequest
	GetEvent       func() entity.Event
	SetRunning     func(running bool)
	SetSubActive   func(active bool)
	SetDirty       func(dirty bool)
	SetEnvKeys     func(keys []string)
	SetScriptError func(msg string)
	SetScriptIcon  func(icon fyne.Resource)
	Messages       *MessagesView
	Subject        func() string
	FocusName      func()
}

func NewRequestView(
	onRun func(method constants.NatsMethod, req entity.NatsRequest, event entity.Event),
	onUnsub func(),
	onChange func(name string, req entity.NatsRequest, event entity.Event),
	onSave func(),
	messages *MessagesView,
) *RequestView {
	var applying bool
	var header *ui.EntityHeader
	statusLabel := widget.NewLabel("")
	statusLabel.TextStyle = fyne.TextStyle{Italic: true}

	subject := ui.NewEnvInput()
	subject.SetPlaceHolder("{{natsSubject}} or demo.events")
	payload := ui.NewEnvMultiLineInput()
	payload.SetPlaceHolder(`{"ok": true, "token": "{{token}}"}`)
	payload.SetMinRowsVisible(8)

	var headers *ui.KVTable
	var scriptView *scripttab.ScriptView
	var scriptTab *container.TabItem
	var requestTabs *container.AppTabs
	var getReq func() entity.NatsRequest

	notify := func() {
		if applying || onChange == nil || header == nil || getReq == nil || scriptView == nil {
			return
		}
		onChange(header.GetName(), getReq(), scriptView.Get())
	}

	header = ui.NewEntityHeader(theme.DocumentIcon(), "Request name", func(string) { notify() }, onSave)

	headers = ui.NewKVTable(nil, func([]ui.KVRow) { notify() })
	scriptView = scripttab.NewScriptView(func(entity.Event) { notify() })

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
	subject.OnChanged(func(string) { notify() })
	payload.OnChanged(func(string) { notify() })

	publishBtn := widget.NewButton("Publish", func() {
		if onRun != nil {
			onRun(constants.NatsMethodPublish, getReq(), entity.Event{})
		}
	})
	requestBtn := widget.NewButton("Request", func() {
		if onRun != nil {
			onRun(constants.NatsMethodRequest, getReq(), scriptView.Get())
		}
	})
	publishBtn.Importance = widget.MediumImportance
	requestBtn.Importance = widget.MediumImportance

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
			onRun(constants.NatsMethodSubscribe, getReq(), entity.Event{})
		}
	})
	applySubBtn()

	headersPanel := container.NewBorder(
		nil, nil, nil, nil,
		container.NewVScroll(container.NewVBox(headers)),
	)
	payloadPanel := container.NewBorder(nil, nil, nil, nil, payload)
	scriptTab = container.NewTabItem("Script", scriptView.Object)
	requestTabs = container.NewAppTabs(
		container.NewTabItem("Headers", headersPanel),
		container.NewTabItem("Payload", payloadPanel),
		scriptTab,
	)

	actions := container.NewBorder(nil, nil, container.NewHBox(publishBtn, requestBtn, subBtn), statusLabel, nil)
	requestPanel := container.NewBorder(
		container.NewVBox(
			header.Object,
			widget.NewForm(widget.NewFormItem("Subject", subject)),
		),
		actions, nil, nil, requestTabs,
	)
	split := container.NewVSplit(
		ui.NewMinSizeBox(fyne.NewSize(200, 80), container.NewPadded(requestPanel)),
		ui.NewMinSizeBox(fyne.NewSize(200, 80), messages.Object()),
	)
	split.SetOffset(0.55)

	v := &RequestView{CanvasObject: split, Messages: messages, Subject: func() string { return subject.Text() }}
	v.Get = getReq
	v.GetEvent = func() entity.Event {
		if scriptView == nil {
			return entity.Event{}
		}
		return scriptView.Get()
	}
	v.SetDirty = header.SetDirty
	v.FocusName = header.FocusName
	v.SetEnvKeys = scriptView.SetEnvKeys
	v.SetScriptError = scriptView.SetError
	v.SetScriptIcon = func(icon fyne.Resource) {
		scriptTab.Icon = icon
		scriptTab.Text = "Script"
		requestTabs.Refresh()
	}
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
	v.Set = func(req *entity.NatsRequest, name string, isSubscribed bool, event entity.Event) {
		applying = true
		header.SetName(name)
		statusLabel.SetText("")
		if req == nil {
			subject.SetText("")
			payload.SetText("")
			headers.SetRows(nil)
			scriptView.Set(entity.Event{})
			v.SetSubActive(false)
		} else {
			subject.SetText(req.Subject)
			payload.SetText(req.Payload)
			rows := make([]ui.KVRow, 0, len(req.Headers))
			for _, h := range req.Headers {
				rows = append(rows, ui.KVRow{Enabled: true, Key: h.Key, Value: h.Value})
			}
			headers.SetRows(rows)
			scriptView.Set(event)
			v.SetSubActive(isSubscribed)
		}
		scriptView.SetError("")
		applying = false
	}
	return v
}
