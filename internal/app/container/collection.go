package container

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
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
	win := app.Window

	showMenu := func(menu *fyne.Menu, pos fyne.Position) {
		widget.ShowPopUpMenuAtPosition(menu, win.Canvas(), pos)
	}

	var tree *collection.Tree

	selectCollection := func(col entity.Collection) {
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
	}
	selectFolder := func(col entity.Collection, item entity.CollectionItem, path []string) {
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
	}
	selectRequest := func(col entity.Collection, item entity.CollectionItem, path []string) {
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
	}

	createCollection := func(colType constants.CollectionType) {
		log.Printf("[collections] UI createCollection type=%s", colType)
		id, ok := selStore.CreateCollection(colType)
		if !ok {
			log.Printf("[collections] UI createCollection failed")
			return
		}
		uid := tree.CollectionUID(id)
		fyne.Do(func() {
			tree.SelectUID(uid)
			ws := wsStore.GetSelectedWorkspace()
			if ws == nil {
				return
			}
			for i := range ws.Collections {
				if ws.Collections[i].Id == id {
					selectCollection(ws.Collections[i])
					return
				}
			}
		})
	}

	addRequest := func(collectionID, parentItemID, parentUID string) {
		log.Printf("[collections] UI addRequest col=%s parentItem=%s parentUID=%s",
			collectionID, parentItemID, parentUID)
		itemID, path, ok := selStore.AddRequest(collectionID, parentItemID)
		if !ok {
			log.Printf("[collections] UI addRequest failed")
			return
		}
		uid := tree.ItemUID(itemID)
		fyne.Do(func() {
			tree.OpenUID(parentUID)
			tree.SelectUID(uid)
			ws := wsStore.GetSelectedWorkspace()
			if ws == nil {
				return
			}
			for i := range ws.Collections {
				if ws.Collections[i].Id != collectionID {
					continue
				}
				col := ws.Collections[i]
				item := findCollectionItem(col.Items, itemID)
				if item == nil {
					return
				}
				selectRequest(col, *item, path)
				return
			}
		})
	}

	addFolder := func(collectionID, parentItemID, parentUID string) {
		log.Printf("[collections] UI addFolder col=%s parentItem=%s parentUID=%s",
			collectionID, parentItemID, parentUID)
		itemID, path, ok := selStore.AddFolder(collectionID, parentItemID)
		if !ok {
			log.Printf("[collections] UI addFolder failed")
			return
		}
		uid := tree.ItemUID(itemID)
		fyne.Do(func() {
			tree.OpenUID(parentUID)
			tree.SelectUID(uid)
			ws := wsStore.GetSelectedWorkspace()
			if ws == nil {
				return
			}
			for i := range ws.Collections {
				if ws.Collections[i].Id != collectionID {
					continue
				}
				col := ws.Collections[i]
				item := findCollectionItem(col.Items, itemID)
				if item == nil {
					return
				}
				selectFolder(col, *item, path)
				return
			}
		})
	}

	tree = collection.NewTree(
		collection.SelectHandler{
			OnCollection: selectCollection,
			OnFolder:     selectFolder,
			OnRequest:    selectRequest,
		},
		collection.ContextHandler{
			OnCollection: func(col entity.Collection, pos fyne.Position) {
				colUID := tree.CollectionUID(col.Id)
				showMenu(fyne.NewMenu("",
					fyne.NewMenuItem("Add request", func() { addRequest(col.Id, "", colUID) }),
					fyne.NewMenuItem("Add folder", func() { addFolder(col.Id, "", colUID) }),
					fyne.NewMenuItemSeparator(),
					fyne.NewMenuItem("Delete", func() {
						dialog.ShowConfirm("Delete collection", "Delete \""+col.Name+"\"?", func(ok bool) {
							if !ok {
								return
							}
							natsStore.Disconnect(col.Id)
							selStore.DeleteCollection(col.Id)
						}, win)
					}),
				), pos)
			},
			OnFolder: func(col entity.Collection, item entity.CollectionItem, path []string, pos fyne.Position) {
				itemUID := tree.ItemUID(item.Id)
				showMenu(fyne.NewMenu("",
					fyne.NewMenuItem("Add request", func() { addRequest(col.Id, item.Id, itemUID) }),
					fyne.NewMenuItemSeparator(),
					fyne.NewMenuItem("Delete", func() {
						dialog.ShowConfirm("Delete folder", "Delete \""+item.Name+"\"?", func(ok bool) {
							if ok {
								selStore.DeleteItem(col.Id, item.Id)
							}
						}, win)
					}),
				), pos)
			},
			OnRequest: func(col entity.Collection, item entity.CollectionItem, path []string, pos fyne.Position) {
				parentUID := tree.CollectionUID(col.Id)
				if len(path) > 1 {
					parentUID = tree.ItemUID(path[len(path)-2])
				}
				showMenu(fyne.NewMenu("",
					fyne.NewMenuItem("Duplicate", func() {
						itemID, newPath, ok := selStore.DuplicateRequest(col.Id, item.Id)
						if !ok {
							return
						}
						uid := tree.ItemUID(itemID)
						fyne.Do(func() {
							tree.OpenUID(parentUID)
							tree.SelectUID(uid)
							ws := wsStore.GetSelectedWorkspace()
							if ws == nil {
								return
							}
							for i := range ws.Collections {
								if ws.Collections[i].Id != col.Id {
									continue
								}
								c := ws.Collections[i]
								dup := findCollectionItem(c.Items, itemID)
								if dup == nil {
									return
								}
								selectRequest(c, *dup, newPath)
								return
							}
						})
					}),
					fyne.NewMenuItemSeparator(),
					fyne.NewMenuItem("Delete", func() {
						dialog.ShowConfirm("Delete request", "Delete \""+item.Name+"\"?", func(ok bool) {
							if ok {
								if col.Type == constants.CollectionTypeNATS {
									natsStore.Unsubscribe(col.Id, item.Id)
								}
								selStore.DeleteItem(col.Id, item.Id)
							}
						}, win)
					}),
				), pos)
			},
		},
	)

	refreshConnected := func() {
		tree.SetConnected(natsStore.ConnectedIDs())
	}
	natsStore.AddConnectionListener(func() {
		fyne.Do(refreshConnected)
	})

	addMenu := fyne.NewMenu("",
		fyne.NewMenuItem("REST collection", func() { createCollection(constants.CollectionTypeREST) }),
		fyne.NewMenuItem("gRPC collection", func() { createCollection(constants.CollectionTypeGRPC) }),
		fyne.NewMenuItem("NATS collection", func() { createCollection(constants.CollectionTypeNATS) }),
		fyne.NewMenuItem("WS collection", func() { createCollection(constants.CollectionTypeWS) }),
	)
	var addBtn *widget.Button
	addBtn = widget.NewButtonWithIcon("Add", theme.ContentAddIcon(), func() {
		widget.ShowPopUpMenuAtRelativePosition(addMenu, win.Canvas(), fyne.NewPos(0, addBtn.Size().Height), addBtn)
	})
	addBtn.Importance = widget.LowImportance
	addBtn.Hide()

	search := widget.NewEntry()
	search.SetPlaceHolder("Search…")
	search.OnChanged = func(q string) {
		tree.SetFilter(q)
	}
	search.Hide()

	toolbarRight := addBtn
	toolbar := container.NewBorder(nil, nil, nil, toolbarRight, search)

	syncToolbar := func() {
		if wsStore.GetSelectedWorkspace() == nil {
			addBtn.Hide()
			search.Hide()
			search.SetText("")
			tree.SetFilter("")
		} else {
			addBtn.Show()
			search.Show()
		}
		toolbar.Refresh()
	}
	wsItem.AddListener(binding.NewDataListener(func() {
		workspace := wsStore.GetWorkspaceDataItem(wsItem)
		if workspace == nil {
			log.Printf("[collections] workspace listener: nil")
			tree.SetCollections(nil)
			selStore.ClearSelection()
			syncToolbar()
			return
		}
		log.Printf("[collections] workspace listener: refresh tree cols=%d", len(workspace.Collections))
		tree.SetCollections(workspace.Collections)
		refreshConnected()
		syncToolbar()
	}))

	return container.NewBorder(
		toolbar,
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

func findCollectionItem(items []entity.CollectionItem, id string) *entity.CollectionItem {
	for i := range items {
		if items[i].Id == id {
			return &items[i]
		}
		if found := findCollectionItem(items[i].Item, id); found != nil {
			return found
		}
	}
	return nil
}
