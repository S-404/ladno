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
	natsStore := app.Store.Nats
	kafkaStore := app.Store.Kafka

	empty := container.NewCenter(widget.NewLabel("Select a collection or request"))

	var colSettings *collection.SettingsView
	colSettings = collection.NewSettingsView(collection.SettingsCallbacks{
		OnSave: func(save collection.SettingsSave) {
			sel := currentSelection(selStore.GetSelection())
			if sel == nil || sel.Kind != entity.SelectionCollection {
				return
			}
			selStore.UpdateCollection(sel.CollectionID, save.Name, save.Auth, save.Nats, save.Kafka)
		},
		OnConnect: func(save collection.SettingsSave) {
			sel := currentSelection(selStore.GetSelection())
			if sel == nil || sel.Kind != entity.SelectionCollection {
				return
			}
			selStore.UpdateCollection(sel.CollectionID, save.Name, save.Auth, save.Nats, save.Kafka)
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
	saveRequestName := func(name string) {
		sel := currentSelection(selStore.GetSelection())
		if sel == nil || sel.Kind != entity.SelectionRequest {
			return
		}
		selStore.UpdateRequestName(sel.CollectionID, sel.ItemID, name)
	}

	restPanel := RestContainer(app)
	grpcPanel := grpcui.NewRequestView(saveRequestAuth, saveRequestName)
	wsPanel := wsui.NewRequestView(saveRequestAuth, saveRequestName)

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
		func() {
			fyne.CurrentApp().Clipboard().SetContent(natsPanel.Messages.Text())
		},
		func() {
			natsStore.ClearMessages(natsCollectionID, natsPanel.Subject())
			refreshNatsMessages()
		},
	)
	natsStore.AddMessageListener(func() {
		fyne.Do(refreshNatsMessages)
	})

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
		saveRequestName,
		natsMessagesView,
	)

	var kafkaPanel *kafkaui.RequestView
	var kafkaCollectionID string
	refreshKafkaMessages := func() {
		if kafkaPanel == nil {
			return
		}
		text := kafkaStore.MessagesText(
			kafkaCollectionID,
			kafkaPanel.Topic(),
			kafkaPanel.Messages.Filter(),
			kafkaPanel.Messages.ShowAll(),
		)
		kafkaPanel.Messages.SetText(text)
	}
	kafkaMessagesView := kafkaui.NewMessagesView(
		func(all bool) { refreshKafkaMessages() },
		func(q string) { refreshKafkaMessages() },
		func() {
			fyne.CurrentApp().Clipboard().SetContent(kafkaPanel.Messages.Text())
		},
		func() {
			kafkaStore.ClearMessages(kafkaCollectionID, kafkaPanel.Topic())
			refreshKafkaMessages()
		},
	)
	kafkaStore.AddMessageListener(func() {
		fyne.Do(refreshKafkaMessages)
	})

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
		saveRequestName,
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
				natsCollectionID = sel.CollectionID
				natsPanel.Set(req, sel.Name, natsStore.IsSubscribed(sel.CollectionID, sel.ItemID))
				refreshNatsMessages()
				show(6)
			case constants.CollectionTypeKafka:
				var req *entity.KafkaRequest
				if sel.Request != nil {
					req = sel.Request.Kafka
				}
				kafkaCollectionID = sel.CollectionID
				kafkaPanel.Set(req, sel.Name, kafkaStore.IsConsuming(sel.CollectionID, sel.ItemID))
				refreshKafkaMessages()
				show(7)
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
