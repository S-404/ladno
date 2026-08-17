package kafkaui

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
	Set              func(req *entity.KafkaRequest, name string, consuming bool)
	Get              func() entity.KafkaRequest
	SetRunning       func(running bool)
	SetConsumeActive func(active bool)
	SetDirty         func(dirty bool)
	Messages         *MessagesView
	Topic            func() string
}

func NewRequestView(
	onRun func(method constants.KafkaMethod, req entity.KafkaRequest),
	onStop func(),
	onChange func(name string, req entity.KafkaRequest),
	onSave func(),
	messages *MessagesView,
) *RequestView {
	var applying bool
	header := ui.NewEntityHeader("Kafka request", onSave)
	statusLabel := widget.NewLabel("")
	statusLabel.TextStyle = fyne.TextStyle{Italic: true}

	topic := ui.NewEnvInput()
	topic.SetPlaceHolder("{{kafkaTopic}} or demo.events")
	key := ui.NewEnvInput()
	key.SetPlaceHolder("optional message key")
	payload := ui.NewEnvMultiLineInput()
	payload.SetPlaceHolder(`{"ok": true}`)
	payload.SetMinRowsVisible(6)

	var nameField *ui.RequestNameField
	var headers *ui.KVTable
	var getReq func() entity.KafkaRequest

	notify := func() {
		if applying || onChange == nil || nameField == nil || getReq == nil {
			return
		}
		onChange(nameField.Get(), getReq())
	}
	headers = ui.NewKVTable(nil, func([]ui.KVRow) { notify() })
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
	nameField = ui.NewRequestNameField(func(string) { notify() })
	topic.OnChanged(func(string) { notify() })
	key.OnChanged(func(string) { notify() })
	payload.OnChanged(func(string) { notify() })

	envHint := widget.NewLabel("Topic, key, headers and payload support {{var}} from the active environment.")
	envHint.TextStyle = fyne.TextStyle{Italic: true}

	produceBtn := widget.NewButton("Produce", func() {
		if onRun != nil {
			onRun(constants.KafkaMethodProduce, getReq())
		}
	})
	produceBtn.Importance = widget.HighImportance

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
			onRun(constants.KafkaMethodConsume, getReq())
		}
	})
	applyConsumeBtn()

	actions := container.NewBorder(nil, nil, container.NewHBox(produceBtn, consumeBtn), statusLabel, nil)
	requestPanel := container.NewBorder(
		container.NewVBox(
			header.Object,
			nameField.Object,
			widget.NewForm(
				widget.NewFormItem("Topic", topic),
				widget.NewFormItem("Key", key),
			),
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

	v := &RequestView{CanvasObject: split, Messages: messages, Topic: func() string { return topic.Text() }}
	v.Get = getReq
	v.SetDirty = header.SetDirty
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
	v.Set = func(req *entity.KafkaRequest, name string, consuming bool) {
		applying = true
		nameField.Set(name)
		statusLabel.SetText("")
		if req == nil {
			topic.SetText("")
			key.SetText("")
			payload.SetText("")
			headers.SetRows(nil)
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
			v.SetConsumeActive(consuming)
		}
		applying = false
	}
	return v
}
