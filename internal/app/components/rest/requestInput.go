package rest

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/ui"
)

type RequestInputView struct {
	Object    fyne.CanvasObject
	GetMethod func() string
	SetMethod func(method string)
}

func NewRequestInput(methods []string, requestString binding.String, onSend func(), onMethodChange func(string)) *RequestInputView {
	input := ui.NewUrlInput(requestString)

	selected := ""
	if len(methods) > 0 {
		selected = methods[0]
	}
	applying := false

	methodSelect := widget.NewSelect(methods, func(s string) {
		selected = s
		if applying || onMethodChange == nil {
			return
		}
		onMethodChange(s)
	})

	requestButton := widget.NewButton("Send", onSend)

	if len(methods) == 0 {
		methodSelect.Hide()
	} else {
		applying = true
		methodSelect.SetSelected(selected)
		applying = false
	}

	border := container.NewBorder(
		nil,
		nil,
		widget.NewForm(widget.NewFormItem("Method", methodSelect)),
		requestButton,
		widget.NewForm(widget.NewFormItem("Url", input)),
	)

	return &RequestInputView{
		Object: border,
		GetMethod: func() string {
			return selected
		},
		SetMethod: func(method string) {
			selected = method
			applying = true
			methodSelect.SetSelected(method)
			applying = false
		},
	}
}
