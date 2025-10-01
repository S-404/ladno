package container

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/goose/internal/app/components/ui"
	"github.com/s-404/goose/internal/app/entity/shared"
)

func WorkspaceContainer(app *shared.App) fyne.CanvasObject {
	wsList := app.Store.Workspace.GetItems()
	list := widget.NewListWithData(*wsList,
		func() fyne.CanvasObject {
			return widget.NewLabel("Workspace")
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			ws := app.Store.Workspace.GetWorkspaceListItemDataItem(item)
			if ws != nil {
				obj.(*widget.Label).SetText(fmt.Sprintf("%s", ws.Name))
			}
		},
	)

	btn := widget.NewButtonWithIcon("Select one", theme.MenuDropDownIcon(), nil)
	btn.IconPlacement = 1

	listLoader := ui.NewLoader(app.Store.Workspace.GetIsFetching())
	dlgContent := container.NewStack(container.NewVBox(listLoader), list)
	dlg := ui.NewModal("Select Workspace", "Cancel", dlgContent, app.Window)

	btn.OnTapped = func() {
		dlg.Show()
		app.Store.Workspace.FetchWorkspaceList()
	}

	list.OnSelected = func(id widget.ListItemID) {
		selected := app.Store.Workspace.GetWorkspaceListItemByIndex(id)
		app.Store.Workspace.FetchWorkspace(selected.Id)
		btn.Text = selected.Name
		btn.Refresh()
		dlg.Hide()
	}

	return container.NewHBox(
		widget.NewLabel("Workspace:"),
		btn,
	)
}
