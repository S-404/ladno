package container

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/shared"
)

// BindGlobalSaveShortcut registers Ctrl+S for both canvas (no focus) and focused
// text inputs (via ui.SetGlobalSaveHandler). Fyne does not fall through from a
// focused Entry to canvas shortcuts.
// tabs selects which context to save (collections vs envs).
func BindGlobalSaveShortcut(app *shared.App, tabs *container.AppTabs) {
	sc := &desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierControl}
	save := func() { saveForActiveTab(app, tabs) }
	ui.SetGlobalSaveHandler(save)
	app.Window.Canvas().AddShortcut(sc, func(fyne.Shortcut) { save() })
}

func saveForActiveTab(app *shared.App, tabs *container.AppTabs) {
	if tabs != nil {
		if tab := tabs.Selected(); tab != nil {
			switch tab.Text {
			case "envs":
				saveSelectedEnv(app)
				return
			case "settings":
				return
			}
		}
	}
	saveCollectionSelection(app)
}

func saveCollectionSelection(app *shared.App) {
	drafts := app.Store.Draft
	sel := currentSelection(app.Store.Selection.GetSelection())
	if sel == nil {
		return
	}
	switch sel.Kind {
	case entity.SelectionRequest:
		if drafts.IsRequestDirty(sel.ItemID) {
			drafts.SaveRequest(sel.CollectionID, sel.ItemID)
		}
	case entity.SelectionFolder:
		if drafts.IsFolderDirty(sel.ItemID) {
			drafts.SaveFolder(sel.CollectionID, sel.ItemID)
		}
	case entity.SelectionCollection:
		if drafts.IsCollectionDirty(sel.CollectionID) {
			drafts.SaveCollection(sel.CollectionID)
		}
	}
}

func saveSelectedEnv(app *shared.App) {
	drafts := app.Store.Draft
	val, err := app.Store.Env.GetSelected().Get()
	if err != nil || val == nil {
		return
	}
	env, ok := val.(*entity.Env)
	if !ok || env == nil {
		return
	}
	if drafts.IsEnvDirty(env.Id) {
		drafts.SaveEnv(env.Id)
	}
}
