package layout

import (
	"fyne.io/fyne/v2"
	appContainer "github.com/s-404/goose/internal/app/container"
	"github.com/s-404/goose/internal/app/entity/shared"
)

func EnvsLayout(app *shared.App) fyne.CanvasObject {
	return appContainer.EnvContainer(app)
}
