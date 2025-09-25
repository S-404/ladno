package layout

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	appContainer "github.com/s-404/goose/internal/app/container"
	"github.com/s-404/goose/internal/app/entity/shared"
	"image/color"
)

func HeaderLayout(app *shared.App) fyne.CanvasObject {
	return container.NewBorder(
		nil,
		makeLine(),
		appContainer.WorkspaceContainer(app),
		nil,
		nil,
	)
}

func makeLine() fyne.CanvasObject {
	rect := canvas.NewRectangle(&color.NRGBA{128, 128, 128, 255})
	rect.SetMinSize(fyne.NewSize(1, 1))
	return rect
}
