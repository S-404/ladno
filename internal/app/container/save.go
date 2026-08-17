package container

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/shared"
)

// BindGlobalSaveShortcut registers Ctrl+S for both canvas (no focus) and focused
// text inputs (via ui.SetGlobalSaveHandler). Fyne does not fall through from a
// focused Entry to canvas shortcuts.
func BindGlobalSaveShortcut(app *shared.App) {
	sc := &desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierControl}
	save := func() { saveCurrentSelection(app) }
	ui.SetGlobalSaveHandler(save)
	app.Window.Canvas().AddShortcut(sc, func(fyne.Shortcut) { save() })
}

func saveCurrentSelection(app *shared.App) {
	drafts := app.Store.Draft
	sel := currentSelection(app.Store.Selection.GetSelection())
	if sel != nil {
		switch sel.Kind {
		case entity.SelectionRequest:
			if drafts.IsRequestDirty(sel.ItemID) {
				drafts.SaveRequest(sel.CollectionID, sel.ItemID)
			}
			return
		case entity.SelectionFolder:
			if drafts.IsFolderDirty(sel.ItemID) {
				drafts.SaveFolder(sel.CollectionID, sel.ItemID)
			}
			return
		case entity.SelectionCollection:
			if drafts.IsCollectionDirty(sel.CollectionID) {
				drafts.SaveCollection(sel.CollectionID)
			}
			return
		}
	}
	// Env tab: save selected env if dirty.
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
