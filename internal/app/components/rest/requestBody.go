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
	BodyModeRaw        BodyMode = "raw"
	BodyModeFormData   BodyMode = "form-data"
	BodyModeURLEncoded BodyMode = "x-www-form-urlencoded"
)

// BodyState хранит состояние таба Body
type BodyState struct {
	Mode           BodyMode
	RawText        string
	FormRows       []ui.KVRow
	URLEncodedRows []ui.KVRow
}

type RequestBodyView struct {
	Object fyne.CanvasObject
	Get    func() BodyState
	Set    func(state BodyState)
}

// NewRequestBody возвращает таб Body.
// onChange вызывается при любом изменении содержимого.
func NewRequestBody(initial BodyState, onChange func(state BodyState), win fyne.Window) *RequestBodyView {
	state := initial
	if state.Mode == "" {
		state.Mode = BodyModeRaw
	}
	var applying bool

	rawEntry := ui.NewEnvMultiLineInput()
	rawEntry.SetPlaceHolder("Paste JSON, XML or plain text…")
	rawEntry.SetText(state.RawText)
	rawEntry.SetMinRowsVisible(8)
	rawEntry.OnChanged(func(v string) {
		if applying {
			return
		}
		state.RawText = v
		if onChange != nil {
			onChange(state)
		}
	})

	formTable := ui.NewKVTableFormData(state.FormRows, func(rows []ui.KVRow) {
		if applying {
			return
		}
		state.FormRows = rows
		if onChange != nil {
			onChange(state)
		}
	}, win)

	urlencodedTable := ui.NewKVTable(state.URLEncodedRows, func(rows []ui.KVRow) {
		if applying {
			return
		}
		state.URLEncodedRows = rows
		if onChange != nil {
			onChange(state)
		}
	})

	stack := container.NewStack(rawEntry, formTable, urlencodedTable)

	var modeSelect *widget.Select

	applyMode := func(mode BodyMode, notify bool) {
		state.Mode = mode
		rawEntry.Hide()
		formTable.Hide()
		urlencodedTable.Hide()
		switch mode {
		case BodyModeFormData:
			formTable.Show()
		case BodyModeURLEncoded:
			urlencodedTable.Show()
		default:
			rawEntry.Show()
		}
		stack.Refresh()
		if notify && onChange != nil {
			onChange(state)
		}
	}

	applyMode(state.Mode, false)

	modes := []string{string(BodyModeRaw), string(BodyModeFormData), string(BodyModeURLEncoded)}
	modeSelect = widget.NewSelect(modes, func(s string) {
		if applying {
			return
		}
		applyMode(BodyMode(s), true)
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
			state.URLEncodedRows = urlencodedTable.GetRows()
			return state
		},
		Set: func(next BodyState) {
			if next.Mode == "" {
				next.Mode = BodyModeRaw
			}
			applying = true
			state = next
			rawEntry.SetText(state.RawText)
			formTable.SetRows(state.FormRows)
			urlencodedTable.SetRows(state.URLEncodedRows)
			modeSelect.SetSelected(string(state.Mode))
			applyMode(state.Mode, false)
			applying = false
		},
	}
}
