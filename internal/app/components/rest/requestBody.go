package rest

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/ui"
)

// BodyMode — тип тела запроса
type BodyMode string

const (
	BodyModeRaw      BodyMode = "raw"
	BodyModeFormData BodyMode = "form-data"
)

// BodyState хранит состояние таба Body
type BodyState struct {
	Mode     BodyMode
	RawText  string
	FormRows []ui.KVRow
}

// NewRequestBody возвращает таб Body.
// onChange вызывается при любом изменении содержимого.
func NewRequestBody(initial BodyState, onChange func(state BodyState)) fyne.CanvasObject {
	state := initial
	if state.Mode == "" {
		state.Mode = BodyModeRaw
	}

	// --- raw editor ---
	rawBinding := binding.NewString()
	rawBinding.Set(state.RawText)
	rawEntry := widget.NewMultiLineEntry()
	rawEntry.Bind(rawBinding)
	rawEntry.SetPlaceHolder("Paste JSON, XML or plain text…")
	rawEntry.TextStyle = fyne.TextStyle{Monospace: true}
	rawEntry.Wrapping = fyne.TextWrapOff
	rawBinding.AddListener(binding.NewDataListener(func() {
		v, _ := rawBinding.Get()
		state.RawText = v
		if onChange != nil {
			onChange(state)
		}
	}))

	// --- form-data editor ---
	formTable := ui.NewKVTable(state.FormRows, func(rows []ui.KVRow) {
		state.FormRows = rows
		if onChange != nil {
			onChange(state)
		}
	})

	// --- content stack ---
	rawScroll := container.NewScroll(rawEntry)
	rawScroll.SetMinSize(fyne.NewSize(0, 200))

	// Показываем нужный редактор
	stack := container.NewStack(rawScroll, formTable)

	applyMode := func(mode BodyMode) {
		state.Mode = mode
		switch mode {
		case BodyModeRaw:
			formTable.Hide()
			rawScroll.Show()
		case BodyModeFormData:
			rawScroll.Hide()
			formTable.Show()
		}
		stack.Refresh()
		if onChange != nil {
			onChange(state)
		}
	}

	// Инициализация видимости
	applyMode(state.Mode)

	// --- mode selector ---
	modes := []string{string(BodyModeRaw), string(BodyModeFormData)}
	modeSelect := widget.NewSelect(modes, func(s string) {
		applyMode(BodyMode(s))
	})
	modeSelect.SetSelected(string(state.Mode))

	toolbar := container.NewHBox(
		widget.NewLabel("Body:"),
		modeSelect,
	)

	return container.NewBorder(
		toolbar,
		nil, nil, nil,
		stack,
	)
}
