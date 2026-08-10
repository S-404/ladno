package container

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/rest"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/shared"
)

func CollectionContainer(app *shared.App) fyne.CanvasObject {
	wsStore := app.Store.Workspace
	restStore := app.Store.Rest
	wsItem := wsStore.GetItem()

	onSelectCollectionItem := func(item entity.CollectionItem) {
		if item.Request == nil {
			return
		}
		restStore.SetDraft(draftFromCollectionItem(item))
	}

	collectionTree := rest.NewCollectionTree(onSelectCollectionItem)

	wsItem.AddListener(binding.NewDataListener(func() {
		workspace := wsStore.GetWorkspaceDataItem(wsItem)
		if workspace == nil {
			collectionTree.SetCollections(nil)
			return
		}
		collectionTree.SetCollections(workspace.Collections)
	}))

	return container.NewBorder(
		widget.NewToolbar(),
		nil, nil, nil,
		container.NewScroll(collectionTree),
	)
}

func draftFromCollectionItem(item entity.CollectionItem) entity.RestDraft {
	req := item.Request
	pathParams := map[string]string{}
	for _, v := range req.Url.Variable {
		if v.Key != "" {
			pathParams[v.Key] = v.Value
		}
	}
	return entity.RestDraft{
		Method:     string(req.Method),
		URL:        req.Url.Raw,
		PathParams: pathParams,
		Headers:    req.Header,
		BodyMode:   entity.RestBodyRaw,
	}
}
