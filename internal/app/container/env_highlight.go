package container

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/shared"
	"github.com/s-404/ladno/internal/app/utils"
)

// BindEnvHighlights keeps {{var}} known-keys and envs-tab used-keys in sync.
func BindEnvHighlights(app *shared.App, tabs *container.AppTabs, onUsedKeys func(keys []string)) {
	refreshKnown := func() {
		ui.SetKnownEnvKeys(activeEnvListedKeys(app))
	}
	refreshUsed := func() {
		if onUsedKeys == nil {
			return
		}
		if tabs != nil {
			if tab := tabs.Selected(); tab == nil || tab.Text != "envs" {
				onUsedKeys(nil)
				return
			}
		}
		onUsedKeys(usedEnvKeysFromSelection(app))
	}
	refreshAll := func() {
		refreshKnown()
		refreshUsed()
	}

	refreshAll()

	app.Store.Env.GetActiveID().AddListener(binding.NewDataListener(func() {
		fyne.Do(refreshAll)
	}))
	(*app.Store.Env.GetItems()).AddListener(binding.NewDataListener(func() {
		fyne.Do(refreshAll)
	}))
	app.Store.Selection.GetSelection().AddListener(binding.NewDataListener(func() {
		fyne.Do(refreshUsed)
	}))
	app.Store.Draft.AddDirtyListener(func() {
		fyne.Do(refreshAll)
	})
	if tabs != nil {
		prev := tabs.OnSelected
		tabs.OnSelected = func(tab *container.TabItem) {
			if prev != nil {
				prev(tab)
			}
			fyne.Do(refreshUsed)
		}
	}
}

func activeEnvListedKeys(app *shared.App) []string {
	activeID, _ := app.Store.Env.GetActiveID().Get()
	if activeID == "" {
		return nil
	}
	if d, ok := app.Store.Draft.GetEnvDraft(activeID); ok {
		return envVariableKeys(d.Variables)
	}
	env := app.Store.Env.GetEnvByID(activeID)
	if env == nil {
		return nil
	}
	return envVariableKeys(env.Variables)
}

func envVariableKeys(vars []entity.EnvVariable) []string {
	out := make([]string, 0, len(vars))
	seen := map[string]struct{}{}
	for _, v := range vars {
		if v.Key == "" {
			continue
		}
		if _, ok := seen[v.Key]; ok {
			continue
		}
		seen[v.Key] = struct{}{}
		out = append(out, v.Key)
	}
	return out
}

func usedEnvKeysFromSelection(app *shared.App) []string {
	sel := currentSelection(app.Store.Selection.GetSelection())
	if sel == nil {
		return nil
	}
	drafts := app.Store.Draft
	switch sel.Kind {
	case entity.SelectionRequest:
		if d, ok := drafts.GetRequestDraft(sel.ItemID); ok {
			return utils.CollectItemRequestEnvKeys(d.Request)
		}
		return nil
	case entity.SelectionCollection:
		if d, ok := drafts.GetCollectionDraft(sel.CollectionID); ok {
			return utils.CollectCollectionEnvKeys(d.Nats, d.Kafka, d.Auth)
		}
		return nil
	case entity.SelectionFolder:
		if d, ok := drafts.GetFolderDraft(sel.ItemID); ok {
			return utils.CollectCollectionEnvKeys(nil, nil, d.Auth)
		}
		return nil
	default:
		return nil
	}
}
