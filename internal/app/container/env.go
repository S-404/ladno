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
	drafts := app.Store.Draft
	items := envStore.GetItems()

	var applying bool
	var header *ui.EntityHeader
	var varsTable *ui.KVTable
	nameEntry := ui.NewEntry()
	nameEntry.SetPlaceHolder("Environment name")

	header = ui.NewTitledEntityHeader("Env variables", func() {
		sel := selectedEnv(envStore)
		if sel == nil {
			return
		}
		flushEnvDraft(app, nameEntry.Text, varsTable.GetRows(), true)
		drafts.SaveEnv(sel.Id)
		header.SetDirty(false)
	})

	varsTable = ui.NewKVTableEnv(nil, func(rows []ui.KVRow) {
		if applying {
			return
		}
		flushEnvDraft(app, nameEntry.Text, rows, true)
		if sel := selectedEnv(envStore); sel != nil {
			header.SetDirty(drafts.IsEnvDirty(sel.Id))
		}
	})

	nameEntry.OnChanged = func(string) {
		if applying {
			return
		}
		flushEnvDraft(app, nameEntry.Text, varsTable.GetRows(), true)
		if sel := selectedEnv(envStore); sel != nil {
			header.SetDirty(drafts.IsEnvDirty(sel.Id))
		}
	}

	var syncList func()
	envList := ui.NewEnvList(
		func(id string) { envStore.Select(id) },
		func(id string, toIndex int) {
			envStore.MoveEnv(id, toIndex)
			syncList()
		},
	)
	envList.SetDirtyCheck(drafts.IsEnvDirty)

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
			name := env.Name
			if d, ok := drafts.GetEnvDraft(env.Id); ok {
				name = d.Name
			}
			listItems = append(listItems, ui.EnvListItem{ID: env.Id, Name: name})
		}
		selID := ""
		if sel := selectedEnv(envStore); sel != nil {
			selID = sel.Id
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
			header.SetDirty(false)
			syncList()
			return
		}
		env, ok := val.(*entity.Env)
		if !ok || env == nil {
			return
		}
		d := drafts.EnsureEnvDraft(env)
		applying = true
		nameEntry.Enable()
		nameEntry.SetText(d.Name)
		varsTable.SetRows(envVariablesToKVRows(d.Variables))
		applying = false
		header.SetDirty(drafts.IsEnvDirty(env.Id))
		syncList()
	}

	envStore.GetSelected().AddListener(binding.NewDataListener(applySelected))
	envStore.GetActiveID().AddListener(binding.NewDataListener(syncList))
	(*items).AddListener(binding.NewDataListener(syncList))
	drafts.AddDirtyListener(func() {
		fyne.Do(func() {
			syncList()
			if sel := selectedEnv(envStore); sel != nil {
				header.SetDirty(drafts.IsEnvDirty(sel.Id))
			}
		})
	})

	newBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		envStore.Create(entity.DefaultNewEnvName)
	})
	cloneBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		envStore.CloneSelected()
	})
	deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		env := selectedEnv(envStore)
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
		toolbar, nil, nil, nil,
		container.NewStack(container.NewVBox(loader), envList),
	)
	editor := container.NewBorder(
		container.NewVBox(
			header.Object,
			container.NewBorder(nil, nil, widget.NewLabel("Name"), nil, nameEntry),
		),
		nil, nil, nil,
		container.NewVScroll(varsTable),
	)
	split := container.NewHSplit(
		ui.NewMinSizeBox(fyne.NewSize(140, 80), left),
		ui.NewMinSizeBox(fyne.NewSize(200, 80), editor),
	)
	split.SetOffset(0.28)
	envStore.FetchList()
	return split
}

func selectedEnv(envStore interface {
	GetSelected() binding.Untyped
}) *entity.Env {
	val, err := envStore.GetSelected().Get()
	if err != nil || val == nil {
		return nil
	}
	env, _ := val.(*entity.Env)
	return env
}

func flushEnvDraft(app *shared.App, name string, rows []ui.KVRow, markDirty bool) {
	sel := selectedEnv(app.Store.Env)
	if sel == nil {
		return
	}
	if rows == nil {
		// keep existing draft vars if rows not provided
		if d, ok := app.Store.Draft.GetEnvDraft(sel.Id); ok {
			app.Store.Draft.PutEnvDraft(sel.Id, entity.EnvDraft{Name: name, Variables: d.Variables}, markDirty)
			return
		}
	}
	app.Store.Draft.PutEnvDraft(sel.Id, entity.EnvDraft{
		Name:      name,
		Variables: kvRowsToEnvVariables(rows),
	}, markDirty)
}

func envVariablesToKVRows(vars []entity.EnvVariable) []ui.KVRow {
	rows := make([]ui.KVRow, 0, len(vars))
	for _, v := range vars {
		rows = append(rows, ui.KVRow{Enabled: v.Enabled, Key: v.Key, Value: v.Value, Secret: v.IsSecret})
	}
	return rows
}

func kvRowsToEnvVariables(rows []ui.KVRow) []entity.EnvVariable {
	out := make([]entity.EnvVariable, 0, len(rows))
	for _, r := range rows {
		if r.Key == "" && r.Value == "" {
			continue
		}
		out = append(out, entity.EnvVariable{Key: r.Key, Value: r.Value, Enabled: r.Enabled, IsSecret: r.Secret})
	}
	return out
}
