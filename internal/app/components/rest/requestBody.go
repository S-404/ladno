package rest

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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

type RequestBodyView struct {
	Object fyne.CanvasObject
	Get    func() BodyState
	Set    func(state BodyState)
}

// NewRequestBody возвращает таб Body.
// onChange вызывается при любом изменении содержимого.
func NewRequestBody(initial BodyState, onChange func(state BodyState)) *RequestBodyView {
	state := initial
	if state.Mode == "" {
		state.Mode = BodyModeRaw
	}

	rawEntry := ui.NewEnvMultiLineInput()
	rawEntry.SetPlaceHolder("Paste JSON, XML or plain text…")
	rawEntry.SetText(state.RawText)
	rawEntry.SetMinRowsVisible(8)
	rawEntry.OnChanged(func(v string) {
		state.RawText = v
		if onChange != nil {
			onChange(state)
		}
	})

	formTable := ui.NewKVTable(state.FormRows, func(rows []ui.KVRow) {
		state.FormRows = rows
		if onChange != nil {
			onChange(state)
		}
	})

	stack := container.NewStack(rawEntry, formTable)

	var modeSelect *widget.Select

	applyMode := func(mode BodyMode) {
		state.Mode = mode
		switch mode {
		case BodyModeRaw:
			formTable.Hide()
			rawEntry.Show()
		case BodyModeFormData:
			rawEntry.Hide()
			formTable.Show()
		}
		stack.Refresh()
		if onChange != nil {
			onChange(state)
		}
	}

	applyMode(state.Mode)

	modes := []string{string(BodyModeRaw), string(BodyModeFormData)}
	modeSelect = widget.NewSelect(modes, func(s string) {
		applyMode(BodyMode(s))
	})
	modeSelect.SetSelected(string(state.Mode))

	toolbar := container.NewHBox(
		widget.NewLabel("Body:"),
		modeSelect,
	)

	return &RequestBodyView{
		Object: container.NewBorder(toolbar, nil, nil, nil, stack),
		Get: func() BodyState {
			state.RawText = rawEntry.Text()
			state.FormRows = formTable.GetRows()
			return state
		},
		Set: func(next BodyState) {
			if next.Mode == "" {
				next.Mode = BodyModeRaw
			}
			state = next
			rawEntry.SetText(state.RawText)
			formTable.SetRows(state.FormRows)
			modeSelect.SetSelected(string(state.Mode))
			applyMode(state.Mode)
		},
	}
}
