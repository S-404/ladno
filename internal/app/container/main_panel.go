package container

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/collection"
	"github.com/s-404/ladno/internal/app/components/grpcui"
	"github.com/s-404/ladno/internal/app/components/kafkaui"
	"github.com/s-404/ladno/internal/app/components/natsui"
	"github.com/s-404/ladno/internal/app/components/wsui"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
	"github.com/s-404/ladno/internal/app/entity/shared"
)

func MainPanelContainer(app *shared.App) fyne.CanvasObject {
	selStore := app.Store.Selection
	drafts := app.Store.Draft
	natsStore := app.Store.Nats
	kafkaStore := app.Store.Kafka

	empty := container.NewCenter(widget.NewLabel("Select a collection or request"))

	var colSettings *collection.SettingsView
	colSettings = collection.NewSettingsView(collection.SettingsCallbacks{
		OnChange: func(save collection.SettingsSave) {
			sel := currentSelection(selStore.GetSelection())
			if sel == nil || sel.Kind != entity.SelectionCollection {
				return
			}
			drafts.PutCollectionDraft(sel.CollectionID, entity.CollectionDraft{
				Name: save.Name, Auth: save.Auth, Nats: save.Nats, Kafka: save.Kafka,
			}, true)
			colSettings.SetDirty(true)
		},
		OnSave: func(save collection.SettingsSave) {
			sel := currentSelection(selStore.GetSelection())
			if sel == nil || sel.Kind != entity.SelectionCollection {
				return
			}
			drafts.PutCollectionDraft(sel.CollectionID, entity.CollectionDraft{
				Name: save.Name, Auth: save.Auth, Nats: save.Nats, Kafka: save.Kafka,
			}, true)
			drafts.SaveCollection(sel.CollectionID)
			colSettings.SetDirty(false)
		},
		OnConnect: func(save collection.SettingsSave) {
			sel := currentSelection(selStore.GetSelection())
			if sel == nil || sel.Kind != entity.SelectionCollection {
				return
			}
			// Connect uses current draft values without forcing disk save.
			drafts.PutCollectionDraft(sel.CollectionID, entity.CollectionDraft{
				Name: save.Name, Auth: save.Auth, Nats: save.Nats, Kafka: save.Kafka,
			}, drafts.IsCollectionDirty(sel.CollectionID))
			switch constants.NormalizeCollectionType(sel.CollectionType) {
			case constants.CollectionTypeNATS:
				if save.Nats == nil {
					colSettings.SetConnectStatus("Host/port required")
					return
				}
				colSettings.SetConnectStatus("Connecting…")
				natsStore.Connect(sel.CollectionID, save.Name, *save.Nats, func(ok bool, status string) {
					colSettings.SetConnectStatus(status)
					colSettings.SetConnected(ok)
				})
			case constants.CollectionTypeKafka:
				if save.Kafka == nil || save.Kafka.Brokers == "" {
					colSettings.SetConnectStatus("Brokers required")
					return
				}
				colSettings.SetConnectStatus("Connecting…")
				kafkaStore.Connect(sel.CollectionID, save.Name, *save.Kafka, func(ok bool, status string) {
					colSettings.SetConnectStatus(status)
					colSettings.SetConnected(ok)
				})
			}
		},
		OnDisconnect: func() {
			sel := currentSelection(selStore.GetSelection())
			if sel == nil || sel.Kind != entity.SelectionCollection {
				return
			}
			switch constants.NormalizeCollectionType(sel.CollectionType) {
			case constants.CollectionTypeNATS:
				natsStore.Disconnect(sel.CollectionID)
			case constants.CollectionTypeKafka:
				kafkaStore.Disconnect(sel.CollectionID)
			}
			colSettings.SetConnected(false)
			colSettings.SetConnectStatus("Disconnected")
		},
	})

	var folderSettings *collection.FolderView
	folderSettings = collection.NewFolderView(
		func(name string, auth entity.Auth) {
			sel := currentSelection(selStore.GetSelection())
			if sel == nil || sel.Kind != entity.SelectionFolder {
				return
			}
			drafts.PutFolderDraft(sel.ItemID, entity.FolderDraft{
				CollectionID: sel.CollectionID, Name: name, Auth: auth,
			}, true)
			folderSettings.SetDirty(true)
		},
		func(name string, auth entity.Auth) {
			sel := currentSelection(selStore.GetSelection())
			if sel == nil || sel.Kind != entity.SelectionFolder {
				return
			}
			drafts.PutFolderDraft(sel.ItemID, entity.FolderDraft{
				CollectionID: sel.CollectionID, Name: name, Auth: auth,
			}, true)
			drafts.SaveFolder(sel.CollectionID, sel.ItemID)
			folderSettings.SetDirty(false)
		},
	)

	putRequestDraft := func(mutate func(*entity.RequestDraft)) {
		sel := currentSelection(selStore.GetSelection())
		if sel == nil || sel.Kind != entity.SelectionRequest {
			return
		}
		d, ok := drafts.GetRequestDraft(sel.ItemID)
		if !ok {
			d = drafts.EnsureRequestDraft(sel.CollectionID, sel.ItemID, sel.Name, sel.Request)
		}
		mutate(&d)
		d.CollectionID = sel.CollectionID
		drafts.PutRequestDraft(sel.ItemID, d, true)
	}

	restPanel := RestContainer(app)

	grpcPanel := grpcui.NewRequestView(
		func(name string, req entity.GrpcRequest) {
			putRequestDraft(func(d *entity.RequestDraft) {
				d.Name = name
				d.Request.Grpc = &req
			})
		},
		func() {
			sel := currentSelection(selStore.GetSelection())
			if sel != nil {
				drafts.SaveRequest(sel.CollectionID, sel.ItemID)
			}
		},
	)
	wsPanel := wsui.NewRequestView(
		func(name string, req entity.WsRequest) {
			putRequestDraft(func(d *entity.RequestDraft) {
				d.Name = name
				d.Request.Ws = &req
			})
		},
		func() {
			sel := currentSelection(selStore.GetSelection())
			if sel != nil {
				drafts.SaveRequest(sel.CollectionID, sel.ItemID)
			}
		},
	)

	var natsPanel *natsui.RequestView
	var natsCollectionID string
	refreshNatsMessages := func() {
		if natsPanel == nil {
			return
		}
		text := natsStore.MessagesText(natsCollectionID, natsPanel.Subject(), natsPanel.Messages.ShowAll())
		natsPanel.Messages.SetText(text)
	}
	natsMessagesView := natsui.NewMessagesView(
		func(all bool) { refreshNatsMessages() },
		func() { fyne.CurrentApp().Clipboard().SetContent(natsPanel.Messages.Text()) },
		func() {
			natsStore.ClearMessages(natsCollectionID, natsPanel.Subject())
			refreshNatsMessages()
		},
	)
	natsStore.AddMessageListener(func() { fyne.Do(refreshNatsMessages) })

	natsPanel = natsui.NewRequestView(
		func(method constants.NatsMethod, req entity.NatsRequest) {
			sel := currentSelection(selStore.GetSelection())
			if sel == nil || sel.Kind != entity.SelectionRequest {
				return
			}
			natsCollectionID = sel.CollectionID
			natsPanel.SetRunning(true)
			natsStore.Run(sel.CollectionID, sel.ItemID, method, req, func(err error) {
				natsPanel.SetRunning(false)
				if method == constants.NatsMethodSubscribe && err == nil {
					natsPanel.SetSubActive(true)
				}
				refreshNatsMessages()
			})
		},
		func() {
			sel := currentSelection(selStore.GetSelection())
			if sel == nil || sel.Kind != entity.SelectionRequest {
				return
			}
			natsStore.Unsubscribe(sel.CollectionID, sel.ItemID)
			natsPanel.SetSubActive(false)
		},
		func(name string, req entity.NatsRequest) {
			putRequestDraft(func(d *entity.RequestDraft) {
				d.Name = name
				d.Request.Nats = &req
			})
		},
		func() {
			sel := currentSelection(selStore.GetSelection())
			if sel != nil {
				drafts.SaveRequest(sel.CollectionID, sel.ItemID)
			}
		},
		natsMessagesView,
	)

	var kafkaPanel *kafkaui.RequestView
	var kafkaCollectionID string
	refreshKafkaMessages := func() {
		if kafkaPanel == nil {
			return
		}
		text := kafkaStore.MessagesText(kafkaCollectionID, kafkaPanel.Topic(), kafkaPanel.Messages.Filter(), kafkaPanel.Messages.ShowAll())
		kafkaPanel.Messages.SetText(text)
	}
	kafkaMessagesView := kafkaui.NewMessagesView(
		func(all bool) { refreshKafkaMessages() },
		func(q string) { refreshKafkaMessages() },
		func() { fyne.CurrentApp().Clipboard().SetContent(kafkaPanel.Messages.Text()) },
		func() {
			kafkaStore.ClearMessages(kafkaCollectionID, kafkaPanel.Topic())
			refreshKafkaMessages()
		},
	)
	kafkaStore.AddMessageListener(func() { fyne.Do(refreshKafkaMessages) })

	kafkaPanel = kafkaui.NewRequestView(
		func(method constants.KafkaMethod, req entity.KafkaRequest) {
			sel := currentSelection(selStore.GetSelection())
			if sel == nil || sel.Kind != entity.SelectionRequest {
				return
			}
			kafkaCollectionID = sel.CollectionID
			kafkaPanel.SetRunning(true)
			kafkaStore.Run(sel.CollectionID, sel.ItemID, method, req, func(err error) {
				kafkaPanel.SetRunning(false)
				if method == constants.KafkaMethodConsume && err == nil {
					kafkaPanel.SetConsumeActive(true)
				}
				refreshKafkaMessages()
			})
		},
		func() {
			sel := currentSelection(selStore.GetSelection())
			if sel == nil || sel.Kind != entity.SelectionRequest {
				return
			}
			kafkaStore.StopConsume(sel.CollectionID, sel.ItemID)
			kafkaPanel.SetConsumeActive(false)
		},
		func(name string, req entity.KafkaRequest) {
			putRequestDraft(func(d *entity.RequestDraft) {
				d.Name = name
				d.Request.Kafka = &req
			})
		},
		func() {
			sel := currentSelection(selStore.GetSelection())
			if sel != nil {
				drafts.SaveRequest(sel.CollectionID, sel.ItemID)
			}
		},
		kafkaMessagesView,
	)

	app.Store.Env.GetActiveID().AddListener(binding.NewDataListener(func() {
		refreshNatsMessages()
		refreshKafkaMessages()
	}))

	panels := []fyne.CanvasObject{
		empty,
		colSettings.CanvasObject,
		folderSettings.CanvasObject,
		restPanel,
		grpcPanel.CanvasObject,
		wsPanel.CanvasObject,
		natsPanel.CanvasObject,
		kafkaPanel.CanvasObject,
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

	syncDirtyHeaders := func(sel *entity.Selection) {
		if sel == nil {
			return
		}
		switch sel.Kind {
		case entity.SelectionCollection:
			colSettings.SetDirty(drafts.IsCollectionDirty(sel.CollectionID))
		case entity.SelectionFolder:
			folderSettings.SetDirty(drafts.IsFolderDirty(sel.ItemID))
		case entity.SelectionRequest:
			dirty := drafts.IsRequestDirty(sel.ItemID)
			grpcPanel.SetDirty(dirty)
			wsPanel.SetDirty(dirty)
			natsPanel.SetDirty(dirty)
			kafkaPanel.SetDirty(dirty)
		}
	}

	drafts.AddDirtyListener(func() {
		fyne.Do(func() {
			syncDirtyHeaders(currentSelection(selStore.GetSelection()))
		})
	})

	selStore.GetSelection().AddListener(binding.NewDataListener(func() {
		sel := currentSelection(selStore.GetSelection())
		if sel == nil {
			show(0)
			return
		}
		switch sel.Kind {
		case entity.SelectionCollection:
			colType := constants.NormalizeCollectionType(sel.CollectionType)
			connected := false
			switch colType {
			case constants.CollectionTypeNATS:
				connected = natsStore.IsConnected(sel.CollectionID)
			case constants.CollectionTypeKafka:
				connected = kafkaStore.IsConnected(sel.CollectionID)
			}
			colSettings.Set(sel.Name, sel.Auth, sel.Nats, sel.Kafka, sel.CollectionType, connected)
			colSettings.SetDirty(drafts.IsCollectionDirty(sel.CollectionID))
			show(1)
			if sel.FocusName {
				sel.FocusName = false
				fyne.Do(colSettings.FocusName)
			}
		case entity.SelectionFolder:
			folderSettings.Set(sel.Name, sel.Auth, constants.NormalizeCollectionType(sel.CollectionType) == constants.CollectionTypeREST)
			folderSettings.SetDirty(drafts.IsFolderDirty(sel.ItemID))
			show(2)
			if sel.FocusName {
				sel.FocusName = false
				fyne.Do(folderSettings.FocusName)
			}
		case entity.SelectionRequest:
			d, _ := drafts.GetRequestDraft(sel.ItemID)
			dirty := drafts.IsRequestDirty(sel.ItemID)
			focusName := sel.FocusName
			if focusName {
				sel.FocusName = false
			}
			switch constants.NormalizeCollectionType(sel.CollectionType) {
			case constants.CollectionTypeGRPC:
				grpcPanel.Set(d.Request.Grpc, d.Name)
				grpcPanel.SetDirty(dirty)
				show(4)
				if focusName {
					fyne.Do(grpcPanel.FocusName)
				}
			case constants.CollectionTypeWS:
				wsPanel.Set(d.Request.Ws, d.Name)
				wsPanel.SetDirty(dirty)
				show(5)
				if focusName {
					fyne.Do(wsPanel.FocusName)
				}
			case constants.CollectionTypeNATS:
				natsCollectionID = sel.CollectionID
				natsPanel.Set(d.Request.Nats, d.Name, natsStore.IsSubscribed(sel.CollectionID, sel.ItemID))
				natsPanel.SetDirty(dirty)
				refreshNatsMessages()
				show(6)
				if focusName {
					fyne.Do(natsPanel.FocusName)
				}
			case constants.CollectionTypeKafka:
				kafkaCollectionID = sel.CollectionID
				kafkaPanel.Set(d.Request.Kafka, d.Name, kafkaStore.IsConsuming(sel.CollectionID, sel.ItemID))
				kafkaPanel.SetDirty(dirty)
				refreshKafkaMessages()
				show(7)
				if focusName {
					fyne.Do(kafkaPanel.FocusName)
				}
			default:
				show(3)
				// REST focuses from RestContainer listener
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
