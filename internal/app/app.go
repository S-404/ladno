package app

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity/shared"

	appContainer "github.com/s-404/ladno/internal/app/container"
	"github.com/s-404/ladno/internal/app/layout"
	"github.com/s-404/ladno/internal/app/repository"
	"github.com/s-404/ladno/internal/app/service"
	"github.com/s-404/ladno/internal/app/store"
)

func Init(window fyne.Window) {
	newRepository := repository.NewRepository()
	newService := service.NewService(newRepository)
	newStore := store.NewStore(newService)
	newStore.Settings.ApplyTheme()

	app := shared.App{
		Store:  *newStore,
		Window: window,
	}

	envs, setEnvUsedKeys := layout.EnvsLayout(&app)
	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("collections", theme.HomeIcon(), layout.CollectionsLayout(&app)),
		container.NewTabItemWithIcon("envs", theme.StorageIcon(), envs),
		container.NewTabItemWithIcon("settings", theme.SettingsIcon(), layout.SettingsLayout(&app)),
	)
	tabs.SetTabLocation(container.TabLocationLeading)

	header := layout.HeaderLayout(&app)
	logs := layout.LogsLayout(&app)

	body := container.NewVSplit(
		ui.NewMinSizeBox(fyne.NewSize(200, 100), tabs),
		ui.NewMinSizeBox(fyne.NewSize(200, 64), logs),
	)
	body.SetOffset(0.75)

	app.Window.SetContent(container.NewBorder(header, nil, nil, nil, body))
	appContainer.BindGlobalSaveShortcut(&app, tabs)
	appContainer.BindEnvHighlights(&app, tabs, setEnvUsedKeys)
}
