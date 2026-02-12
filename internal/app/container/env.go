package container

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/entity/shared"
)

func EnvContainer(app *shared.App) fyne.CanvasObject {
	return widget.NewLabel("env tab")
}
