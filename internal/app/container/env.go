package container

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/entity/shared"
)

// EnvContainer — переменные окружения для подстановки в запросы ({{var}}).
func EnvContainer(app *shared.App) fyne.CanvasObject {
	return widget.NewLabel("envs")
}
