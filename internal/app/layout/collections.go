package layout

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"github.com/s-404/ladno/internal/app/components/ui"
	appContainer "github.com/s-404/ladno/internal/app/container"
	"github.com/s-404/ladno/internal/app/entity/shared"
)

func CollectionsLayout(app *shared.App) fyne.CanvasObject {
	left := appContainer.CollectionContainer(app)
	right := appContainer.RestContainer(app)

	split := container.NewHSplit(
		ui.NewMinSizeBox(fyne.NewSize(120, 80), left),
		ui.NewMinSizeBox(fyne.NewSize(200, 80), right),
	)
	split.SetOffset(0.22)

	return split
}
