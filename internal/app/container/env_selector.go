package container

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/shared"
)

func EnvSelectorContainer(app *shared.App) fyne.CanvasObject {
	envStore := app.Store.Env
	items := envStore.GetItems()

	btn := widget.NewButtonWithIcon("No env", theme.MenuDropDownIcon(), nil)
	btn.IconPlacement = 1

	list := widget.NewListWithData(*items,
		func() fyne.CanvasObject {
			return widget.NewLabel("env")
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			env := envStore.GetEnvDataItem(item)
			if env != nil {
				obj.(*widget.Label).SetText(env.Name)
			}
		},
	)

	dlgContent := container.NewStack(list)
	dlg := ui.NewModal("Select Environment", "Cancel", dlgContent, app.Window)

	refreshLabel := func() {
		activeID, _ := envStore.GetActiveID().Get()
		if activeID == "" {
			btn.SetText("No env")
			return
		}
		all, _ := (*items).Get()
		for _, item := range all {
			env, ok := item.(*entity.Env)
			if ok && env != nil && env.Id == activeID {
				btn.SetText(env.Name)
				return
			}
		}
		btn.SetText("No env")
	}

	btn.OnTapped = func() {
		envStore.FetchList()
		dlg.Show()
	}

	list.OnSelected = func(id widget.ListItemID) {
		env := envStore.GetItemByIndex(id)
		if env == nil {
			return
		}
		envStore.SetActive(env.Id)
		refreshLabel()
		dlg.Hide()
		list.UnselectAll()
	}

	envStore.GetActiveID().AddListener(binding.NewDataListener(refreshLabel))
	(*items).AddListener(binding.NewDataListener(refreshLabel))

	envStore.FetchList()

	return container.NewHBox(
		widget.NewLabel("Env:"),
		btn,
	)
}
