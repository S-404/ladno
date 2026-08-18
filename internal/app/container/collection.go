package container

import (
	"fmt"
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
	kafkaStore := app.Store.Kafka
	wsItem := wsStore.GetItem()
	win := app.Window

	showMenu := func(menu *fyne.Menu, pos fyne.Position) {
		widget.ShowPopUpMenuAtPosition(menu, win.Canvas(), pos)
	}

	var tree *collection.Tree

	drafts := app.Store.Draft

	selectCollection := func(col entity.Collection, focusName bool) {
		col.Type = constants.NormalizeCollectionType(col.Type)
		d := drafts.EnsureCollectionDraft(col)
		var nats *entity.NatsConnection
		if d.Nats != nil {
			cp := *d.Nats
			nats = &cp
		}
		var kafka *entity.KafkaConnection
		if d.Kafka != nil {
			cp := *d.Kafka
			kafka = &cp
		}
		selStore.SetSelection(entity.Selection{
			Kind:           entity.SelectionCollection,
			CollectionID:   col.Id,
			CollectionType: col.Type,
			Name:           d.Name,
			Auth:           d.Auth,
			Nats:           nats,
			Kafka:          kafka,
			FocusName:      focusName,
		})
	}
	selectFolder := func(col entity.Collection, item entity.CollectionItem, path []string, focusName bool) {
		col.Type = constants.NormalizeCollectionType(col.Type)
		d := drafts.EnsureFolderDraft(col.Id, item.Id, item.Name, item.Auth)
		selStore.SetSelection(entity.Selection{
			Kind:           entity.SelectionFolder,
			CollectionID:   col.Id,
			CollectionType: col.Type,
			ItemID:         item.Id,
			Path:           path,
			Name:           d.Name,
			Auth:           d.Auth,
			FocusName:      focusName,
		})
	}
	selectRequest := func(col entity.Collection, item entity.CollectionItem, path []string, focusName bool) {
		col.Type = constants.NormalizeCollectionType(col.Type)
		d := drafts.EnsureRequestDraft(col.Id, item.Id, item.Name, item.Request)
		req := d.Request
		selStore.SetSelection(entity.Selection{
			Kind:           entity.SelectionRequest,
			CollectionID:   col.Id,
			CollectionType: col.Type,
			ItemID:         item.Id,
			Path:           path,
			Name:           d.Name,
			Auth:           req.Auth,
			Request:        &req,
			FocusName:      focusName,
		})
		if col.Type == constants.CollectionTypeREST {
			restStore.SetDraft(restDraftFromRequestDraft(d))
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
					selectCollection(ws.Collections[i], true)
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
				selectRequest(col, *item, path, true)
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
				selectFolder(col, *item, path, true)
				return
			}
		})
	}

	tree = collection.NewTree(
		collection.SelectHandler{
			OnCollection: func(col entity.Collection) { selectCollection(col, false) },
			OnFolder: func(col entity.Collection, item entity.CollectionItem, path []string) {
				selectFolder(col, item, path, false)
			},
			OnRequest: func(col entity.Collection, item entity.CollectionItem, path []string) {
				selectRequest(col, item, path, false)
			},
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
							kafkaStore.Disconnect(col.Id)
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
								selectRequest(c, *dup, newPath, true)
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
		collection.ReorderHandler{
			OnCollection: func(collectionID string, steps int) {
				selStore.MoveCollection(collectionID, steps)
			},
			OnItem: func(collectionID, itemID string, steps int) {
				selStore.MoveItem(collectionID, itemID, steps)
			},
			OnRelocate: func(fromCollectionID, itemID, toCollectionID, toParentItemID string, toIndex int) {
				if !selStore.RelocateItem(fromCollectionID, itemID, toCollectionID, toParentItemID, toIndex) {
					return
				}
				tree.RevealItem(toCollectionID, toParentItemID, itemID)
			},
		},
	)
	tree.SetDirtyResolver(collection.DirtyResolver{
		IsDirty: func(collectionID, itemID string, isCollection, isFolder bool) bool {
			if isCollection {
				return drafts.IsCollectionDirty(collectionID)
			}
			if isFolder {
				return drafts.IsFolderDirty(itemID)
			}
			return drafts.IsRequestDirty(itemID)
		},
		ResolveLabel: func(collectionID, itemID string, isCollection, isFolder bool, fallback string) string {
			if isCollection {
				return drafts.CollectionDisplayName(collectionID, fallback)
			}
			if isFolder {
				return drafts.FolderDisplayName(itemID, fallback)
			}
			d, ok := drafts.GetRequestDraft(itemID)
			if !ok {
				return fallback
			}
			if colType := collectionTypeByID(wsStore, collectionID); colType == constants.CollectionTypeREST {
				return fmt.Sprintf("[%s] %s", d.Request.Method, d.Name)
			}
			return d.Name
		},
	})
	drafts.AddDirtyListener(func() {
		fyne.Do(tree.RefreshDirty)
	})

	refreshConnected := func() {
		ids := natsStore.ConnectedIDs()
		for id, ok := range kafkaStore.ConnectedIDs() {
			if ok {
				ids[id] = true
			}
		}
		tree.SetConnected(ids)
	}
	natsStore.AddConnectionListener(func() {
		fyne.Do(refreshConnected)
	})
	kafkaStore.AddConnectionListener(func() {
		fyne.Do(refreshConnected)
	})

	addMenu := fyne.NewMenu("",
		fyne.NewMenuItem("REST collection", func() { createCollection(constants.CollectionTypeREST) }),
		fyne.NewMenuItem("gRPC collection", func() { createCollection(constants.CollectionTypeGRPC) }),
		fyne.NewMenuItem("NATS collection", func() { createCollection(constants.CollectionTypeNATS) }),
		fyne.NewMenuItem("Kafka collection", func() { createCollection(constants.CollectionTypeKafka) }),
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

func restDraftFromRequestDraft(d entity.RequestDraft) entity.RestDraft {
	req := d.Request
	pathParams := map[string]string{}
	for _, v := range req.Url.Variable {
		if v.Key != "" {
			pathParams[v.Key] = v.Value
		}
	}
	bodyMode := req.BodyMode
	if bodyMode == "" {
		bodyMode = entity.RestBodyRaw
	}
	return entity.RestDraft{
		Method:     string(req.Method),
		URL:        req.Url.Raw,
		PathParams: pathParams,
		Headers:    req.Header,
		Auth:       req.Auth,
		BodyMode:   bodyMode,
		RawBody:    req.Body,
		FormData:   req.FormData,
	}
}

func collectionTypeByID(wsStore interface {
	GetSelectedWorkspace() *entity.Workspace
}, collectionID string) constants.CollectionType {
	ws := wsStore.GetSelectedWorkspace()
	if ws == nil {
		return ""
	}
	for i := range ws.Collections {
		if ws.Collections[i].Id == collectionID {
			return constants.NormalizeCollectionType(ws.Collections[i].Type)
		}
	}
	return ""
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
