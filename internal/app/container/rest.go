package container

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/rest"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity/shared"
)

func RestContainer(app *shared.App) fyne.CanvasObject {
	requestURL := binding.NewString()
	methods := []string{"POST", "GET", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

	// --- request input (method + url + send) ---
	requestInput := rest.NewRequestInput(methods, requestURL, func() {
		value, _ := requestURL.Get()
		fmt.Println("RestContainer send:", value)
	}, app.Window)

	// --- request tabs ---
	requestTabs := container.NewAppTabs(
		container.NewTabItem("Params", rest.NewRequestParams(requestURL)),
		container.NewTabItem("Headers", rest.NewRequestHeaders(nil, func(rows []ui.KVRow) {
			fmt.Println("headers changed:", len(rows), "rows")
		})),
		container.NewTabItem("Auth", widget.NewLabel("auth content")),
		container.NewTabItem("Body", rest.NewRequestBody(rest.BodyState{}, func(state rest.BodyState) {
			fmt.Println("body changed, mode:", state.Mode)
		})),
		container.NewTabItem("Script", widget.NewLabel("script content")),
		container.NewTabItem("Preview", widget.NewLabel("preview content")),
	)

	request := container.NewBorder(
		requestInput,
		nil, nil, nil,
		container.NewStack(requestTabs),
	)

	// --- response panel ---
	responseLabel := widget.NewLabel("response")
	responsePayload := widget.NewLabel("response payload")
	response := container.NewVBox(responseLabel, responsePayload)

	split := container.NewVSplit(request, response)
	split.SetOffset(.7)

	return split
}
