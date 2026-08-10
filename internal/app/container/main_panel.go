package container

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/collection"
	"github.com/s-404/ladno/internal/app/components/grpcui"
	"github.com/s-404/ladno/internal/app/components/natsui"
	"github.com/s-404/ladno/internal/app/components/wsui"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
	"github.com/s-404/ladno/internal/app/entity/shared"
)

func MainPanelContainer(app *shared.App) fyne.CanvasObject {
	selStore := app.Store.Selection

	empty := container.NewCenter(widget.NewLabel("Select a collection or request"))

	var colSettings *collection.SettingsView
	colSettings = collection.NewSettingsView(collection.SettingsCallbacks{
		OnSave: func(save collection.SettingsSave) {
			sel := currentSelection(selStore.GetSelection())
			if sel == nil || sel.Kind != entity.SelectionCollection {
				return
			}
			selStore.UpdateCollection(sel.CollectionID, save.Name, save.Auth, save.Nats)
		},
		OnConnect: func(save collection.SettingsSave) {
			sel := currentSelection(selStore.GetSelection())
			if sel == nil || sel.Kind != entity.SelectionCollection {
				return
			}
			// Persist connection fields, then stub connect until NATS client exists.
			selStore.UpdateCollection(sel.CollectionID, save.Name, save.Auth, save.Nats)
			colSettings.SetConnectStatus("Connect is not implemented yet")
		},
	})

	folderSettings := collection.NewFolderView(func(name string, auth entity.Auth) {
		sel := currentSelection(selStore.GetSelection())
		if sel == nil || sel.Kind != entity.SelectionFolder {
			return
		}
		selStore.UpdateFolder(sel.CollectionID, sel.ItemID, name, auth)
	})

	saveRequestAuth := func(auth entity.Auth) {
		sel := currentSelection(selStore.GetSelection())
		if sel == nil || sel.Kind != entity.SelectionRequest {
			return
		}
		selStore.UpdateRequestAuth(sel.CollectionID, sel.ItemID, auth)
	}

	restPanel := RestContainer(app)
	grpcPanel := grpcui.NewRequestView(saveRequestAuth)
	wsPanel := wsui.NewRequestView(saveRequestAuth)
	natsPanel := natsui.NewRequestView(saveRequestAuth)

	panels := []fyne.CanvasObject{
		empty,
		colSettings.CanvasObject,
		folderSettings.CanvasObject,
		restPanel,
		grpcPanel.CanvasObject,
		wsPanel.CanvasObject,
		natsPanel.CanvasObject,
	}
	stack := container.NewStack(panels...)

	show := func(idx int) {
		for i, p := range panels {
			if i == idx {
				p.Show()
			} else {
				p.Hide()
			}
		}
		stack.Refresh()
	}
	show(0)

	selStore.GetSelection().AddListener(binding.NewDataListener(func() {
		sel := currentSelection(selStore.GetSelection())
		if sel == nil {
			show(0)
			return
		}
		switch sel.Kind {
		case entity.SelectionCollection:
			colSettings.Set(sel.Name, sel.Auth, sel.Nats, sel.CollectionType)
			show(1)
		case entity.SelectionFolder:
			folderSettings.Set(sel.Name, sel.Auth)
			show(2)
		case entity.SelectionRequest:
			switch constants.NormalizeCollectionType(sel.CollectionType) {
			case constants.CollectionTypeGRPC:
				var req *entity.GrpcRequest
				if sel.Request != nil {
					req = sel.Request.Grpc
				}
				grpcPanel.Set(req, sel.Name, sel.Auth)
				show(4)
			case constants.CollectionTypeWS:
				var req *entity.WsRequest
				if sel.Request != nil {
					req = sel.Request.Ws
				}
				wsPanel.Set(req, sel.Name, sel.Auth)
				show(5)
			case constants.CollectionTypeNATS:
				var req *entity.NatsRequest
				if sel.Request != nil {
					req = sel.Request.Nats
				}
				natsPanel.Set(req, sel.Name, sel.Auth)
				show(6)
			default:
				show(3)
			}
		default:
			show(0)
		}
	}))

	return stack
}

func currentSelection(b binding.Untyped) *entity.Selection {
	val, err := b.Get()
	if err != nil || val == nil {
		return nil
	}
	sel, ok := val.(*entity.Selection)
	if !ok {
		return nil
	}
	return sel
}
