package container

import (
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

func EnvContainer(app *shared.App) fyne.CanvasObject {
	envStore := app.Store.Env
	items := envStore.GetItems()

	var applying bool
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Environment name")

	var varsTable *ui.KVTable
	varsTable = ui.NewKVTable(nil, func(rows []ui.KVRow) {
		if applying {
			return
		}
		selVal, err := envStore.GetSelected().Get()
		if err != nil || selVal == nil {
			return
		}
		envStore.SaveSelected(nameEntry.Text, kvRowsToEnvVariables(rows))
	})

	nameEntry.OnChanged = func(string) {
		if applying {
			return
		}
		selVal, err := envStore.GetSelected().Get()
		if err != nil || selVal == nil {
			return
		}
		envStore.SaveSelected(nameEntry.Text, kvRowsToEnvVariables(varsTable.GetRows()))
	}

	list := widget.NewListWithData(*items,
		func() fyne.CanvasObject {
			return widget.NewLabel("env")
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			env := envStore.GetEnvDataItem(item)
			if env == nil {
				return
			}
			label := obj.(*widget.Label)
			activeID, _ := envStore.GetActiveID().Get()
			if env.Id == activeID {
				label.SetText("● " + env.Name)
			} else {
				label.SetText(env.Name)
			}
		},
	)

	list.OnSelected = func(id widget.ListItemID) {
		env := envStore.GetItemByIndex(id)
		if env == nil {
			return
		}
		envStore.Select(env.Id)
	}

	applySelected := func() {
		val, err := envStore.GetSelected().Get()
		if err != nil || val == nil {
			applying = true
			nameEntry.SetText("")
			varsTable.SetRows(nil)
			nameEntry.Disable()
			applying = false
			return
		}
		env, ok := val.(*entity.Env)
		if !ok || env == nil {
			return
		}
		applying = true
		nameEntry.Enable()
		nameEntry.SetText(env.Name)
		varsTable.SetRows(envVariablesToKVRows(env.Variables))
		applying = false
	}

	envStore.GetSelected().AddListener(binding.NewDataListener(applySelected))
	envStore.GetActiveID().AddListener(binding.NewDataListener(func() {
		list.Refresh()
	}))
	(*items).AddListener(binding.NewDataListener(func() {
		list.Refresh()
	}))

	newBtn := widget.NewButtonWithIcon("New", theme.ContentAddIcon(), func() {
		entry := widget.NewEntry()
		entry.SetText("New Environment")
		dialog.ShowForm("New environment", "Create", "Cancel", []*widget.FormItem{
			widget.NewFormItem("Name", entry),
		}, func(ok bool) {
			if !ok {
				return
			}
			name := entry.Text
			if name == "" {
				name = "New Environment"
			}
			envStore.Create(name)
		}, app.Window)
	})

	cloneBtn := widget.NewButtonWithIcon("Clone", theme.ContentCopyIcon(), func() {
		envStore.CloneSelected()
	})

	deleteBtn := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
		val, _ := envStore.GetSelected().Get()
		env, _ := val.(*entity.Env)
		if env == nil {
			return
		}
		dialog.ShowConfirm("Delete environment", "Delete \""+env.Name+"\"?", func(ok bool) {
			if ok {
				envStore.DeleteSelected()
			}
		}, app.Window)
	})

	useBtn := widget.NewButtonWithIcon("Use", theme.ConfirmIcon(), func() {
		val, _ := envStore.GetSelected().Get()
		env, _ := val.(*entity.Env)
		if env == nil {
			return
		}
		envStore.SetActive(env.Id)
	})

	toolbar := container.NewHBox(newBtn, cloneBtn, deleteBtn, useBtn)
	loader := ui.NewLoader(envStore.GetIsFetching())

	left := container.NewBorder(
		toolbar,
		nil, nil, nil,
		container.NewStack(container.NewVBox(loader), list),
	)

	editor := container.NewBorder(
		container.NewBorder(nil, nil, widget.NewLabel("Name"), nil, nameEntry),
		nil, nil, nil,
		varsTable,
	)

	split := container.NewHSplit(left, editor)
	split.SetOffset(0.28)

	envStore.FetchList()
	return split
}

func envVariablesToKVRows(vars []entity.EnvVariable) []ui.KVRow {
	rows := make([]ui.KVRow, 0, len(vars))
	for _, v := range vars {
		rows = append(rows, ui.KVRow{
			Enabled: v.Enabled,
			Key:     v.Key,
			Value:   v.Value,
		})
	}
	return rows
}

func kvRowsToEnvVariables(rows []ui.KVRow) []entity.EnvVariable {
	out := make([]entity.EnvVariable, 0, len(rows))
	for _, r := range rows {
		if r.Key == "" && r.Value == "" {
			continue
		}
		out = append(out, entity.EnvVariable{
			Key:     r.Key,
			Value:   r.Value,
			Enabled: r.Enabled,
		})
	}
	return out
}
