package layout

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	appContainer "github.com/s-404/ladno/internal/app/container"
	"github.com/s-404/ladno/internal/app/entity/shared"
)

func CollectionsLayout(app *shared.App) fyne.CanvasObject {
	left := appContainer.CollectionContainer(app)
	right := appContainer.RestContainer(app)

	splitContainer := container.NewHSplit(left, right)
	splitContainer.SetOffset(.2)

	return splitContainer
}
