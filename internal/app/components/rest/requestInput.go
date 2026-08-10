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

func NewRequestInput(methods []string, requestString binding.String, onSend func()) *RequestInputView {
	input := ui.NewUrlInput(requestString)

	selected := ""
	if len(methods) > 0 {
		selected = methods[0]
	}

	methodSelect := widget.NewSelect(methods, func(s string) {
		selected = s
	})

	requestButton := widget.NewButton("Send", onSend)

	if len(methods) == 0 {
		methodSelect.Hide()
	} else {
		methodSelect.SetSelected(selected)
	}

	border := container.NewBorder(
		nil,
		nil,
		methodSelect,
		requestButton,
		input,
	)

	return &RequestInputView{
		Object: border,
		GetMethod: func() string {
			return selected
		},
		SetMethod: func(method string) {
			selected = method
			methodSelect.SetSelected(method)
		},
	}
}
