package container

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/rest"
	"github.com/s-404/ladno/internal/app/entity/shared"
)

func RestContainer(app *shared.App) fyne.CanvasObject {
	requestString := binding.NewString()
	methods := []string{"POST", "GET", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

	requestInput := rest.NewRequestInput(methods, requestString, func() {
		value, _ := requestString.Get()
		fmt.Println("RestContainer requestinput", value)
	}, app.Window)

	requestTabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Params", nil, rest.NewRequestParams(nil)),
		container.NewTabItemWithIcon("Auth", nil, widget.NewLabel("auth content")),
		container.NewTabItemWithIcon("Headers", nil, widget.NewLabel("headers content")),
		container.NewTabItemWithIcon("Body", nil, widget.NewLabel("body content")),
		container.NewTabItemWithIcon("Script", nil, widget.NewLabel("script content")),
	)

	//request := container.NewStack(requestInput, requestTabs)

	request := container.NewBorder(
		requestInput,
		nil, nil, nil,
		container.NewStack(requestTabs),
	)

	responseLabel := widget.NewLabel("response")
	responsePayload := widget.NewLabel("response payload")
	response := container.NewVBox(responseLabel, responsePayload)

	split := container.NewVSplit(
		request,
		response,
	)

	split.SetOffset(.7)

	return split
}
