package layout

import (
	"fyne.io/fyne/v2"
	appContainer "github.com/s-404/ladno/internal/app/container"
	"github.com/s-404/ladno/internal/app/entity/shared"
)

func EnvsLayout(app *shared.App) (fyne.CanvasObject, func(keys []string)) {
	return appContainer.EnvContainer(app)
}
