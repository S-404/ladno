package shared

import (
	"fyne.io/fyne/v2"
	"github.com/s-404/goose/internal/app/store"
)

type App struct {
	Store  store.Store
	Window fyne.Window
}
