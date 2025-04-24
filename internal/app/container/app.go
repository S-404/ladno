package container

import (
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"github.com/s-404/goose/internal/app/entity/shared"
)

func App(app *shared.App) {
	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("sandbox", theme.BrokenImageIcon(), SandboxContainer(app)),
		container.NewTabItemWithIcon("envs", theme.StorageIcon(), EnvContainer(app)),
		container.NewTabItemWithIcon("settings", theme.SettingsIcon(), SettingsContainer(app)),
	)
	tabs.SetTabLocation(container.TabLocationLeading)

	header := HeaderContainer(app)

	app.Window.SetContent(container.NewAdaptiveGrid(1, container.NewBorder(header, nil, nil, nil, tabs)))
}
