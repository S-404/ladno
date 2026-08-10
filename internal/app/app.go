package app

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"github.com/s-404/ladno/internal/app/entity/shared"

	"github.com/s-404/ladno/internal/app/layout"
	"github.com/s-404/ladno/internal/app/repository"
	"github.com/s-404/ladno/internal/app/service"
	"github.com/s-404/ladno/internal/app/store"
)

func Init(window fyne.Window) {
	newRepository := repository.NewRepository()
	newService := service.NewService(newRepository)
	newStore := store.NewStore(newService)
	app := shared.App{
		Store:  *newStore,
		Window: window,
	}

	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("collections", theme.HomeIcon(), layout.CollectionsLayout(&app)),
		container.NewTabItemWithIcon("envs", theme.StorageIcon(), layout.EnvsLayout(&app)),
		container.NewTabItemWithIcon("settings", theme.SettingsIcon(), layout.SettingsLayout(&app)),
	)
	tabs.SetTabLocation(container.TabLocationLeading)

	header := layout.HeaderLayout(&app)

	app.Window.SetContent(container.NewBorder(header, nil, nil, nil, tabs))
}
