package container

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/shared"
)

func WorkspaceContainer(app *shared.App) fyne.CanvasObject {
	wsStore := app.Store.Workspace
	settings := app.Store.Settings
	wsList := wsStore.GetItems()

	var all []entity.WorkspaceLightWeight
	var filtered []entity.WorkspaceLightWeight
	var query string

	applyFilter := func() {
		q := strings.ToLower(strings.TrimSpace(query))
		if q == "" {
			filtered = append([]entity.WorkspaceLightWeight(nil), all...)
		} else {
			filtered = filtered[:0]
			for _, ws := range all {
				if strings.Contains(strings.ToLower(ws.Name), q) {
					filtered = append(filtered, ws)
				}
			}
		}
	}

	syncFromStore := func() {
		items, err := (*wsList).Get()
		if err != nil {
			all = nil
			filtered = nil
			return
		}
		all = make([]entity.WorkspaceLightWeight, 0, len(items))
		for _, item := range items {
			switch v := item.(type) {
			case entity.WorkspaceLightWeight:
				all = append(all, v)
			case *entity.WorkspaceLightWeight:
				if v != nil {
					all = append(all, *v)
				}
			}
		}
		applyFilter()
	}

	list := widget.NewList(
		func() int { return len(filtered) },
		func() fyne.CanvasObject {
			return widget.NewLabel("Workspace")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(filtered) {
				return
			}
			obj.(*widget.Label).SetText(filtered[id].Name)
		},
	)

	btn := widget.NewButtonWithIcon("Select one", theme.MenuDropDownIcon(), nil)
	btn.IconPlacement = 1

	search := widget.NewEntry()
	search.SetPlaceHolder("Search…")
	search.OnChanged = func(q string) {
		query = q
		applyFilter()
		list.UnselectAll()
		list.Refresh()
	}

	addBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), nil)
	addBtn.Importance = widget.LowImportance

	listLoader := ui.NewLoader(wsStore.GetIsFetching())
	toolbar := container.NewBorder(nil, nil, nil, addBtn, search)
	dlgContent := container.NewBorder(
		toolbar,
		nil, nil, nil,
		container.NewStack(container.NewVBox(listLoader), list),
	)
	dlg := ui.NewModal("Select Workspace", "Cancel", dlgContent, app.Window)

	selectWorkspace := func(selected *entity.WorkspaceLightWeight) {
		if selected == nil {
			return
		}
		settings.SetLastWorkspaceID(selected.Id)
		wsStore.FetchWorkspace(selected.Id)
		btn.SetText(selected.Name)
		dlg.Hide()
		list.UnselectAll()
	}

	addBtn.OnTapped = func() {
		entry := widget.NewEntry()
		entry.SetText("New Workspace")
		entry.SetPlaceHolder("Workspace name")
		dialog.ShowForm("New workspace", "Confirm", "Cancel", []*widget.FormItem{
			widget.NewFormItem("Name", entry),
		}, func(ok bool) {
			if !ok {
				return
			}
			name := strings.TrimSpace(entry.Text)
			if name == "" {
				name = "New Workspace"
			}
			wsStore.Create(name, func(ws *entity.Workspace, err error) {
				if err != nil || ws == nil {
					return
				}
				settings.SetLastWorkspaceID(ws.Id)
				_ = wsStore.GetItem().Set(ws)
				btn.SetText(ws.Name)
				dlg.Hide()
				list.UnselectAll()
			})
		}, app.Window)
	}

	btn.OnTapped = func() {
		query = ""
		search.SetText("")
		dlg.Show()
		wsStore.FetchWorkspaceList()
	}

	list.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(filtered) {
			return
		}
		selected := filtered[id]
		selectWorkspace(&selected)
	}

	(*wsList).AddListener(binding.NewDataListener(func() {
		syncFromStore()
		list.Refresh()
	}))

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
