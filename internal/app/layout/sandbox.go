package layout

import (
	"fyne.io/fyne/v2"
	appContainer "github.com/s-404/goose/internal/app/container"
	"github.com/s-404/goose/internal/app/entity/shared"
)

func SandboxLayout(app *shared.App) fyne.CanvasObject {
	return appContainer.SandboxContainer(app)
}
