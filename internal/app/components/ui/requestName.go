package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// RequestNameField — редактируемое имя запроса.
type RequestNameField struct {
	Object fyne.CanvasObject
	Set    func(name string)
	Get    func() string
}

func NewRequestNameField(onChange func(name string)) *RequestNameField {
	entry := &nameFocusEntry{}
	entry.ExtendBaseWidget(entry)
	entry.SetPlaceHolder("Request name")

	var applying bool
	notify := func() {
		if applying || onChange == nil {
			return
		}
		onChange(entry.Text)
	}
	entry.onFocusLost = notify
	entry.OnSubmitted = func(string) { notify() }
	entry.OnChanged = func(string) { notify() }

	form := widget.NewForm(widget.NewFormItem("Name", entry))
	return &RequestNameField{
		Object: container.NewVBox(form),
		Set: func(name string) {
			applying = true
			entry.SetText(name)
			applying = false
		},
		Get: func() string { return entry.Text },
	}
}

type nameFocusEntry struct {
	widget.Entry
	onFocusLost func()
}

func (e *nameFocusEntry) FocusLost() {
	e.Entry.FocusLost()
	if e.onFocusLost != nil {
		e.onFocusLost()
	}
}

func (e *nameFocusEntry) TypedShortcut(s fyne.Shortcut) {
	if isSaveShortcut(s) {
		triggerGlobalSave()
		return
	}
	e.Entry.TypedShortcut(s)
}
