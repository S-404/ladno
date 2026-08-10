package container

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity/shared"
)

func WorkspaceContainer(app *shared.App) fyne.CanvasObject {
	wsStore := app.Store.Workspace
	settings := app.Store.Settings
	wsList := wsStore.GetItems()
	list := widget.NewListWithData(*wsList,
		func() fyne.CanvasObject {
			return widget.NewLabel("Workspace")
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			ws := wsStore.GetWorkspaceListItemDataItem(item)
			if ws != nil {
				obj.(*widget.Label).SetText(fmt.Sprintf("%s", ws.Name))
			}
		},
	)

	btn := widget.NewButtonWithIcon("Select one", theme.MenuDropDownIcon(), nil)
	btn.IconPlacement = 1

	listLoader := ui.NewLoader(wsStore.GetIsFetching())
	dlgContent := container.NewStack(container.NewVBox(listLoader), list)
	dlg := ui.NewModal("Select Workspace", "Cancel", dlgContent, app.Window)

	btn.OnTapped = func() {
		dlg.Show()
		wsStore.FetchWorkspaceList()
	}

	list.OnSelected = func(id widget.ListItemID) {
		selected := wsStore.GetWorkspaceListItemByIndex(id)
		if selected == nil {
			return
		}
		settings.SetLastWorkspaceID(selected.Id)
		wsStore.FetchWorkspace(selected.Id)
		btn.SetText(selected.Name)
		dlg.Hide()
	}

	wsStore.GetItem().AddListener(binding.NewDataListener(func() {
		ws := wsStore.GetSelectedWorkspace()
		if ws == nil {
			btn.SetText("Select one")
			return
		}
		btn.SetText(ws.Name)
		settings.SetLastWorkspaceID(ws.Id)
	}))

	if id := settings.GetLastWorkspaceID(); id != "" {
		wsStore.FetchWorkspace(id)
	}

	return container.NewHBox(
		widget.NewLabel("Workspace:"),
		btn,
	)
}
