package app

import (
	"fyne.io/fyne/v2"
	"github.com/s-404/goose/internal/app/container"
	"github.com/s-404/goose/internal/app/entity/shared"
	"github.com/s-404/goose/internal/app/repository"
	"github.com/s-404/goose/internal/app/service"
	"github.com/s-404/goose/internal/app/store"
)

func Init(window fyne.Window) {
	newRepository := repository.NewRepository()
	newService := service.NewService(newRepository)
	newStore := store.NewStore(newService)
	app := shared.App{
		Store:  *newStore,
		Window: window,
	}

	container.App(&app)
}
