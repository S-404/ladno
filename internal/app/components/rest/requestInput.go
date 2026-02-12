package rest

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/ui"
)

func NewRequestInput(methods []string, requestString binding.String, onSend func(), window fyne.Window) fyne.CanvasObject {
	input := ui.NewUrlInput(requestString)

	methodSelect := widget.NewSelect(methods, func(s string) {
		fmt.Printf("selected method: %s\n", s)
	})

	requestButton := widget.NewButton("Send", onSend)
	requestButton.Resize(fyne.NewSize(300, requestButton.Size().Height))

	if len(methods) == 0 {
		methodSelect.Hide()
	} else {
		methodSelect.SetSelected(methods[0])
	}

	border := container.NewBorder(
		nil,
		nil,
		methodSelect,  // Left
		requestButton, // Right
		input,         // Center
	)

	return border
}
