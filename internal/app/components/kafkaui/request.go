package kafkaui

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
	Set              func(req *entity.KafkaRequest, name string, consuming bool, event entity.Event)
	Get              func() entity.KafkaRequest
	GetEvent         func() entity.Event
	SetRunning       func(running bool)
	SetConsumeActive func(active bool)
	SetDirty         func(dirty bool)
	SetEnvKeys       func(keys []string)
	SetScriptError   func(msg string)
	SetScriptIcon    func(icon fyne.Resource)
	Messages         *MessagesView
	Topic            func() string
	FocusName        func()
}

func NewRequestView(
	onRun func(method constants.KafkaMethod, req entity.KafkaRequest, event entity.Event),
	onStop func(),
	onChange func(name string, req entity.KafkaRequest, event entity.Event),
	onSave func(),
	messages *MessagesView,
) *RequestView {
	var applying bool
	var header *ui.EntityHeader
	statusLabel := widget.NewLabel("")
	statusLabel.TextStyle = fyne.TextStyle{Italic: true}

	topic := ui.NewEnvInput()
	topic.SetPlaceHolder("{{kafkaTopic}} or demo.events")
	key := ui.NewEnvInput()
	key.SetPlaceHolder("optional message key")
	payload := ui.NewEnvMultiLineInput()
	payload.SetPlaceHolder(`{"ok": true}`)
	payload.SetMinRowsVisible(8)

	var headers *ui.KVTable
	var scriptView *scripttab.ScriptView
	var scriptTab *container.TabItem
	var requestTabs *container.AppTabs
	var getReq func() entity.KafkaRequest

	notify := func() {
		if applying || onChange == nil || header == nil || getReq == nil || scriptView == nil {
			return
		}
		onChange(header.GetName(), getReq(), scriptView.Get())
	}

	header = ui.NewEntityHeader(theme.DocumentIcon(), "Request name", func(string) { notify() }, onSave)

	headers = ui.NewKVTable(nil, func([]ui.KVRow) { notify() })
	scriptView = scripttab.NewScriptView(func(entity.Event) { notify() })

	getReq = func() entity.KafkaRequest {
		rows := headers.GetRows()
		vars := make([]entity.Variable, 0, len(rows))
		for _, r := range rows {
			if r.Key == "" {
				continue
			}
			vars = append(vars, entity.Variable{Key: r.Key, Value: r.Value, Type: "string"})
		}
		return entity.KafkaRequest{Topic: topic.Text(), Key: key.Text(), Headers: vars, Payload: payload.Text()}
	}
	topic.OnChanged(func(string) { notify() })
	key.OnChanged(func(string) { notify() })
	payload.OnChanged(func(string) { notify() })

	produceBtn := widget.NewButton("Produce", func() {
		if onRun != nil {
			onRun(constants.KafkaMethodProduce, getReq(), scriptView.Get())
		}
	})
	produceBtn.Importance = widget.MediumImportance

	var consuming bool
	var consumeBtn *widget.Button
	applyConsumeBtn := func() {
		if consuming {
			consumeBtn.SetText("Stop consuming")
			consumeBtn.Importance = widget.DangerImportance
			return
		}
		consumeBtn.SetText("Consume")
		consumeBtn.Importance = widget.MediumImportance
	}
	consumeBtn = widget.NewButton("Consume", func() {
		if consuming {
			if onStop != nil {
				onStop()
			}
			return
		}
		if onRun != nil {
			onRun(constants.KafkaMethodConsume, getReq(), entity.Event{})
		}
	})
	applyConsumeBtn()

	headersPanel := container.NewBorder(
		nil, nil, nil, nil,
		ui.NewListVScroll(container.NewVBox(headers)),
	)
	payloadPanel := container.NewBorder(nil, nil, nil, nil, payload)
	scriptTab = container.NewTabItem("Script", scriptView.Object)
	requestTabs = container.NewAppTabs(
		container.NewTabItem("Headers", headersPanel),
		container.NewTabItem("Payload", payloadPanel),
		scriptTab,
	)

	actions := container.NewBorder(nil, nil, container.NewHBox(produceBtn, consumeBtn), statusLabel, nil)
	requestPanel := container.NewBorder(
		container.NewVBox(
			header.Object,
			container.NewGridWithColumns(2,
				widget.NewForm(widget.NewFormItem("Topic", topic)),
				widget.NewForm(widget.NewFormItem("Key", key)),
			),
		),
		actions, nil, nil, requestTabs,
	)
	split := container.NewVSplit(
		ui.NewMinSizeBox(fyne.NewSize(200, 80), container.NewPadded(requestPanel)),
		ui.NewMinSizeBox(fyne.NewSize(200, 80), messages.Object()),
	)
	split.SetOffset(0.55)

	v := &RequestView{CanvasObject: split, Messages: messages, Topic: func() string { return topic.Text() }}
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
			produceBtn.Disable()
			consumeBtn.Disable()
			statusLabel.SetText("…")
			return
		}
		produceBtn.Enable()
		consumeBtn.Enable()
	}
	v.SetConsumeActive = func(active bool) {
		consuming = active
		applyConsumeBtn()
		consumeBtn.Refresh()
		if active {
			statusLabel.SetText("Consuming")
			return
		}
		statusLabel.SetText("")
	}
	v.Set = func(req *entity.KafkaRequest, name string, isConsuming bool, event entity.Event) {
		applying = true
		header.SetName(name)
		statusLabel.SetText("")
		if req == nil {
			topic.SetText("")
			key.SetText("")
			payload.SetText("")
			headers.SetRows(nil)
			scriptView.Set(entity.Event{})
			v.SetConsumeActive(false)
		} else {
			topic.SetText(req.Topic)
			key.SetText(req.Key)
			payload.SetText(req.Payload)
			rows := make([]ui.KVRow, 0, len(req.Headers))
			for _, h := range req.Headers {
				rows = append(rows, ui.KVRow{Enabled: true, Key: h.Key, Value: h.Value})
			}
			headers.SetRows(rows)
			scriptView.Set(event)
			v.SetConsumeActive(isConsuming)
		}
		scriptView.SetError("")
		applying = false
	}
	return v
}
