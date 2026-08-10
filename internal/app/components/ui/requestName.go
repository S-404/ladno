package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// RequestNameField — редактируемое имя запроса (сохранение при потере фокуса / Enter).
type RequestNameField struct {
	Object fyne.CanvasObject
	Set    func(name string)
	Get    func() string
}

func NewRequestNameField(onSave func(name string)) *RequestNameField {
	entry := &nameFocusEntry{}
	entry.ExtendBaseWidget(entry)
	entry.SetPlaceHolder("Request name")

	var applying bool
	save := func() {
		if applying || onSave == nil {
			return
		}
		onSave(entry.Text)
	}
	entry.onFocusLost = save
	entry.OnSubmitted = func(string) { save() }

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
