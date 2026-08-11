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

	var syncList func()
	envList := ui.NewEnvList(
		func(id string) {
			envStore.Select(id)
		},
		func(id string, toIndex int) {
			envStore.MoveEnv(id, toIndex)
			// UntypedList list-listeners skip same-length reorder; refresh explicitly.
			syncList()
		},
	)

	syncList = func() {
		raw, err := (*items).Get()
		if err != nil {
			return
		}
		listItems := make([]ui.EnvListItem, 0, len(raw))
		for _, item := range raw {
			env, ok := item.(*entity.Env)
			if !ok || env == nil {
				continue
			}
			listItems = append(listItems, ui.EnvListItem{ID: env.Id, Name: env.Name})
		}
		selID := ""
		if sel := envStore.GetSelected(); sel != nil {
			if v, err := sel.Get(); err == nil {
				if env, ok := v.(*entity.Env); ok && env != nil {
					selID = env.Id
				}
			}
		}
		activeID, _ := envStore.GetActiveID().Get()
		envList.SetItems(listItems, selID, activeID)
	}

	applySelected := func() {
		val, err := envStore.GetSelected().Get()
		if err != nil || val == nil {
			applying = true
			nameEntry.SetText("")
			varsTable.SetRows(nil)
			nameEntry.Disable()
			applying = false
			syncList()
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
		syncList()
	}

	envStore.GetSelected().AddListener(binding.NewDataListener(applySelected))
	envStore.GetActiveID().AddListener(binding.NewDataListener(syncList))
	(*items).AddListener(binding.NewDataListener(syncList))

	newBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
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

	cloneBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		envStore.CloneSelected()
	})

	deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
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

	toolbar := container.NewHBox(newBtn, cloneBtn, deleteBtn)
	loader := ui.NewLoader(envStore.GetIsFetching())

	left := container.NewBorder(
		toolbar,
		nil, nil, nil,
		container.NewStack(container.NewVBox(loader), envList),
	)

	editor := container.NewBorder(
		container.NewBorder(nil, nil, widget.NewLabel("Name"), nil, nameEntry),
		nil, nil, nil,
		varsTable,
	)

	split := container.NewHSplit(
		ui.NewMinSizeBox(fyne.NewSize(140, 80), left),
		ui.NewMinSizeBox(fyne.NewSize(200, 80), editor),
	)
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
