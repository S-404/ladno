package container

import (
	"fmt"
	"fyne.io/fyne/v2/canvas"
	"github.com/s-404/goose/internal/app/entity/shared"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func HeaderContainer(app *shared.App) fyne.CanvasObject {
	b := container.NewHBox(
		widget.NewButton("Foo", func() { fmt.Println("foo") }),
	)

	return container.NewBorder(nil, makeCell(), nil, nil, b)
}

func makeCell() fyne.CanvasObject {
	rect := canvas.NewRectangle(&color.NRGBA{128, 128, 128, 255})
	rect.SetMinSize(fyne.NewSize(1, 1))
	return rect
}
