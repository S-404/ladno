package rest

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	xWidget "fyne.io/x/fyne/widget"
	"strings"
)

func NewRequestInput(methods []string, requestString binding.String, onSend func(), window fyne.Window) fyne.CanvasObject {
	methodSelect := widget.NewSelect(methods, func(s string) {
		fmt.Printf("selected method: %s", s)
	})

	//requestInput := widget.NewEntryWithData(requestString)
	//requestInput.SetPlaceHolder("Request")
	requestButton := widget.NewButton("Send", onSend)
	requestButton.Resize(fyne.NewSize(300, requestButton.Size().Height))

	if len(methods) == 0 {
		methodSelect.Hide()
	} else {
		methodSelect.SetSelected(methods[0])
	}
	options := []string{
		"Apple", "Apricot", "Banana", "Cherry",
		"Grape", "Orange", "Peach", "Pear",
	}

	//entry := NewAutocompleteEntry(options, window)

	entry := xWidget.NewCompletionEntry(options)
	// When the use typed text, complete the list.
	entry.OnChanged = func(s string) {
		// completion start for text length >= 3
		if len(s) < 1 {
			entry.HideCompletion()
			return
		}
		var results []string
		for _, option := range options {
			if strings.Contains(strings.ToLower(option), strings.ToLower(s)) {
				results = append(results, option)
			}
		}

		// no results
		if len(results) == 0 {
			entry.HideCompletion()
			return
		}

		// then show them
		entry.SetOptions(results)
		entry.ShowCompletion()
	}

	border := container.NewBorder(
		nil,
		nil,
		methodSelect,  // Left
		requestButton, // Right
		entry,         // Center (растягивается)
	)

	return border
}
