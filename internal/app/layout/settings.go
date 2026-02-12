package layout

import (
	"fyne.io/fyne/v2"
	appContainer "github.com/s-404/ladno/internal/app/container"
	"github.com/s-404/ladno/internal/app/entity/shared"
)

func SettingsLayout(app *shared.App) fyne.CanvasObject {
	return appContainer.SettingsContainer(app)
}
