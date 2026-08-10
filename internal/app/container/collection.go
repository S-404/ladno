package container

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/collection"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
	"github.com/s-404/ladno/internal/app/entity/shared"
)

func CollectionContainer(app *shared.App) fyne.CanvasObject {
	wsStore := app.Store.Workspace
	selStore := app.Store.Selection
	restStore := app.Store.Rest
	natsStore := app.Store.Nats
	wsItem := wsStore.GetItem()

	tree := collection.NewTree(collection.SelectHandler{
		OnCollection: func(col entity.Collection) {
			col.Type = constants.NormalizeCollectionType(col.Type)
			var nats *entity.NatsConnection
			if col.Nats != nil {
				cp := *col.Nats
				nats = &cp
			}
			selStore.SetSelection(entity.Selection{
				Kind:           entity.SelectionCollection,
				CollectionID:   col.Id,
				CollectionType: col.Type,
				Name:           col.Name,
				Auth:           col.Auth,
				Nats:           nats,
			})
		},
		OnFolder: func(col entity.Collection, item entity.CollectionItem, path []string) {
			col.Type = constants.NormalizeCollectionType(col.Type)
			selStore.SetSelection(entity.Selection{
				Kind:           entity.SelectionFolder,
				CollectionID:   col.Id,
				CollectionType: col.Type,
				ItemID:         item.Id,
				Path:           path,
				Name:           item.Name,
				Auth:           item.Auth,
			})
		},
		OnRequest: func(col entity.Collection, item entity.CollectionItem, path []string) {
			col.Type = constants.NormalizeCollectionType(col.Type)
			auth := entity.Auth{Type: constants.AuthTypeInherited}
			if item.Request != nil {
				auth = item.Request.Auth
			}
			selStore.SetSelection(entity.Selection{
				Kind:           entity.SelectionRequest,
				CollectionID:   col.Id,
				CollectionType: col.Type,
				ItemID:         item.Id,
				Path:           path,
				Name:           item.Name,
				Auth:           auth,
				Request:        item.Request,
			})
			if col.Type == constants.CollectionTypeREST && item.Request != nil {
				restStore.SetDraft(draftFromCollectionItem(item))
			}
		},
	})

	refreshConnected := func() {
		tree.SetConnected(natsStore.ConnectedIDs())
	}
	natsStore.AddConnectionListener(func() {
		fyne.Do(refreshConnected)
	})

	wsItem.AddListener(binding.NewDataListener(func() {
		workspace := wsStore.GetWorkspaceDataItem(wsItem)
		if workspace == nil {
			tree.SetCollections(nil)
			selStore.ClearSelection()
			return
		}
		tree.SetCollections(workspace.Collections)
		refreshConnected()
	}))

	return container.NewBorder(
		widget.NewToolbar(),
		nil, nil, nil,
		container.NewScroll(tree),
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
		Auth:       req.Auth,
		BodyMode:   entity.RestBodyRaw,
	}
}
