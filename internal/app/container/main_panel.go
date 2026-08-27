package container

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/collection"
	"github.com/s-404/ladno/internal/app/components/grpcui"
	"github.com/s-404/ladno/internal/app/components/kafkaui"
	"github.com/s-404/ladno/internal/app/components/messages"
	"github.com/s-404/ladno/internal/app/components/natsui"
	"github.com/s-404/ladno/internal/app/components/socketioui"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/components/wsui"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
	"github.com/s-404/ladno/internal/app/entity/shared"
	"github.com/s-404/ladno/internal/app/store"
)

func MainPanelContainer(app *shared.App) fyne.CanvasObject {
	selStore := app.Store.Selection
	drafts := app.Store.Draft
	natsStore := app.Store.Nats
	kafkaStore := app.Store.Kafka
	wsConnStore := app.Store.Ws
	sioConnStore := app.Store.SocketIO
	grpcStore := app.Store.Grpc
	envStore := app.Store.Env

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

	var grpcPanel *grpcui.RequestView
	var syncGrpcScriptDot func()
	var markGrpcScriptResult func(scriptErr string)
	grpcPanel = grpcui.NewRequestView(
		app.Window,
		func(req entity.GrpcRequest, auth entity.Auth, event entity.Event) {
			syncGrpcScriptDot()
			grpcStore.Send(entity.GrpcCall{
				Target:      req.Target,
				Method:      req.Method,
				Message:     req.Message,
				Metadata:    req.Metadata,
				Auth:        auth,
				ProtoFiles:  req.ProtoFiles,
				ActiveProto: req.ActiveProto,
				PreRequest:  event.PreRequest,
				PostRequest: event.PostRequest,
			})
		},
		func(name string, req entity.GrpcRequest, auth entity.Auth, event entity.Event) {
			putRequestDraft(func(d *entity.RequestDraft) {
				d.Name = name
				d.Request.Grpc = &req
				d.Request.Auth = auth
				d.Request.Event = event
			})
			syncGrpcScriptDot()
		},
		func() {
			sel := currentSelection(selStore.GetSelection())
			if sel != nil {
				drafts.SaveRequest(sel.CollectionID, sel.ItemID)
			}
		},
	)
	hasGrpcScripts := func() bool {
		if grpcPanel == nil || grpcPanel.GetEvent == nil {
			return false
		}
		ev := grpcPanel.GetEvent()
		return len(ev.PreRequest) > 0 || len(ev.PostRequest) > 0
	}
	syncGrpcScriptDot = func() {
		if grpcPanel == nil || grpcPanel.SetScriptIcon == nil {
			return
		}
		if hasGrpcScripts() {
			grpcPanel.SetScriptIcon(ui.DotGray)
		} else {
			grpcPanel.SetScriptIcon(nil)
		}
	}
	markGrpcScriptResult = func(scriptErr string) {
		if grpcPanel == nil {
			return
		}
		if grpcPanel.SetScriptError != nil {
			grpcPanel.SetScriptError(scriptErr)
		}
		if !hasGrpcScripts() {
			grpcPanel.SetScriptIcon(nil)
			return
		}
		if strings.TrimSpace(scriptErr) != "" {
			grpcPanel.SetScriptIcon(ui.DotRed)
			return
		}
		grpcPanel.SetScriptIcon(ui.DotGreen)
	}
	refreshGrpcScriptEnvKeys := func() {
		if grpcPanel != nil && grpcPanel.SetEnvKeys != nil {
			grpcPanel.SetEnvKeys(envStore.ActiveEnvKeys())
		}
	}
	grpcSending := grpcStore.GetIsSending()
	(*grpcSending).AddListener(binding.NewDataListener(func() {
		sending, _ := (*grpcSending).Get()
		if grpcPanel != nil && grpcPanel.SetSending != nil {
			grpcPanel.SetSending(sending)
		}
	}))
	grpcStore.GetResponse().AddListener(binding.NewDataListener(func() {
		val, err := grpcStore.GetResponse().Get()
		if err != nil || val == nil || grpcPanel == nil {
			return
		}
		resp, ok := val.(*entity.GrpcResponse)
		if !ok {
			return
		}
		grpcPanel.SetResponse(resp)
		markGrpcScriptResult(resp.ScriptError)
	}))

	var wsPanel *wsui.RequestView
	var wsCollectionID, wsItemID string
	var wsWasConnected bool
	refreshWsMessages := func() {
		if wsPanel == nil {
			return
		}
		wsPanel.Messages.SetItems(toMessageItems(wsConnStore.Messages(wsCollectionID, wsItemID)))
	}
	wsMessagesView := messages.NewView(
		"WebSocket messages",
		func(all bool) { refreshWsMessages() },
		func() { fyne.CurrentApp().Clipboard().SetContent(wsPanel.Messages.Text()) },
		func() {
			wsConnStore.ClearMessages(wsCollectionID, wsItemID)
			refreshWsMessages()
		},
	)
	wsConnStore.AddMessageListener(func() { fyne.Do(refreshWsMessages) })

	syncWsConnectionUI := func() {
		if wsPanel == nil {
			return
		}
		sel := currentSelection(selStore.GetSelection())
		if sel == nil || sel.Kind != entity.SelectionRequest {
			return
		}
		d, _ := drafts.GetRequestDraft(sel.ItemID)
		if d.Request.Kind() != constants.RequestKindWS {
			return
		}
		now := wsConnStore.IsConnected(sel.CollectionID, sel.ItemID)
		dropped := wsWasConnected && !now
		wsWasConnected = now
		wsPanel.SetConnected(now)
		if dropped {
			wsPanel.SetConnecting(false)
		}
	}
	wsConnStore.AddConnectionListener(func() {
		fyne.Do(syncWsConnectionUI)
	})

	wsPanel = wsui.NewRequestView(
		func(req entity.WsRequest) {
			sel := currentSelection(selStore.GetSelection())
			if sel == nil || sel.Kind != entity.SelectionRequest {
				return
			}
			wsCollectionID = sel.CollectionID
			wsItemID = sel.ItemID
			wsPanel.SetConnecting(true)
			wsConnStore.Connect(sel.CollectionID, sel.ItemID, req, func(ok bool, status string) {
				selNow := currentSelection(selStore.GetSelection())
				if selNow == nil || selNow.ItemID != sel.ItemID {
					return
				}
				wsPanel.SetConnecting(false)
				wsPanel.SetConnected(ok)
				wsWasConnected = ok
				refreshWsMessages()
			})
		},
		func() {
			sel := currentSelection(selStore.GetSelection())
			if sel == nil || sel.Kind != entity.SelectionRequest {
				return
			}
			wsConnStore.Disconnect(sel.CollectionID, sel.ItemID)
			wsPanel.SetConnecting(false)
			wsPanel.SetConnected(false)
			wsWasConnected = false
		},
		func(req entity.WsRequest) {
			sel := currentSelection(selStore.GetSelection())
			if sel == nil || sel.Kind != entity.SelectionRequest {
				return
			}
			wsCollectionID = sel.CollectionID
			wsItemID = sel.ItemID
			wsConnStore.Send(sel.CollectionID, sel.ItemID, req.Message, func(err error) {
				refreshWsMessages()
			})
		},
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
		wsMessagesView,
	)

	var sioPanel *socketioui.RequestView
	var sioCollectionID, sioItemID string
	var sioWasConnected bool
	refreshSioMessages := func() {
		if sioPanel == nil {
			return
		}
		sioPanel.Messages.SetItems(toMessageItems(sioConnStore.Messages(sioCollectionID, sioItemID)))
	}
	sioMessagesView := messages.NewView(
		"Socket.IO messages",
		func(all bool) { refreshSioMessages() },
		func() { fyne.CurrentApp().Clipboard().SetContent(sioPanel.Messages.Text()) },
		func() {
			sioConnStore.ClearMessages(sioCollectionID, sioItemID)
			refreshSioMessages()
		},
	)
	sioConnStore.AddMessageListener(func() { fyne.Do(refreshSioMessages) })

	syncSioConnectionUI := func() {
		if sioPanel == nil {
			return
		}
		sel := currentSelection(selStore.GetSelection())
		if sel == nil || sel.Kind != entity.SelectionRequest {
			return
		}
		d, _ := drafts.GetRequestDraft(sel.ItemID)
		if d.Request.Kind() != constants.RequestKindSocketIO {
			return
		}
		now := sioConnStore.IsConnected(sel.CollectionID, sel.ItemID)
		dropped := sioWasConnected && !now
		sioWasConnected = now
		sioPanel.SetConnected(now)
		if dropped {
			sioPanel.SetConnecting(false)
		}
	}
	sioConnStore.AddConnectionListener(func() {
		fyne.Do(syncSioConnectionUI)
	})

	sioPanel = socketioui.NewRequestView(
		func(req entity.SocketIORequest) {
			sel := currentSelection(selStore.GetSelection())
			if sel == nil || sel.Kind != entity.SelectionRequest {
				return
			}
			sioCollectionID = sel.CollectionID
			sioItemID = sel.ItemID
			sioPanel.SetConnecting(true)
			sioConnStore.Connect(sel.CollectionID, sel.ItemID, req, func(ok bool, status string) {
				selNow := currentSelection(selStore.GetSelection())
				if selNow == nil || selNow.ItemID != sel.ItemID {
					return
				}
				sioPanel.SetConnecting(false)
				sioPanel.SetConnected(ok)
				sioWasConnected = ok
				refreshSioMessages()
			})
		},
		func() {
			sel := currentSelection(selStore.GetSelection())
			if sel == nil || sel.Kind != entity.SelectionRequest {
				return
			}
			sioConnStore.Disconnect(sel.CollectionID, sel.ItemID)
			sioPanel.SetConnecting(false)
			sioPanel.SetConnected(false)
			sioWasConnected = false
		},
		func(req entity.SocketIORequest) {
			sel := currentSelection(selStore.GetSelection())
			if sel == nil || sel.Kind != entity.SelectionRequest {
				return
			}
			sioCollectionID = sel.CollectionID
			sioItemID = sel.ItemID
			sioConnStore.Emit(sel.CollectionID, sel.ItemID, req.Event, req.Payload, req.Namespace, func(err error) {
				refreshSioMessages()
			})
		},
		func(req entity.SocketIORequest) {
			sel := currentSelection(selStore.GetSelection())
			if sel == nil || sel.Kind != entity.SelectionRequest {
				return
			}
			sioCollectionID = sel.CollectionID
			sioItemID = sel.ItemID
			on := !sioConnStore.IsListening(sel.CollectionID, sel.ItemID, req.Event)
			sioConnStore.Listen(sel.CollectionID, sel.ItemID, req.Event, on, func(err error) {
				if err == nil {
					sioPanel.SetListening(sioConnStore.ListeningEvents(sel.CollectionID, sel.ItemID))
				}
				refreshSioMessages()
			})
		},
		func(name string, req entity.SocketIORequest) {
			putRequestDraft(func(d *entity.RequestDraft) {
				d.Name = name
				d.Request.Auth = req.Auth
				d.Request.SocketIO = &req
			})
		},
		func() {
			sel := currentSelection(selStore.GetSelection())
			if sel != nil {
				drafts.SaveRequest(sel.CollectionID, sel.ItemID)
			}
		},
		sioMessagesView,
	)

	var natsPanel *natsui.RequestView
	var natsCollectionID string
	refreshNatsMessages := func() {
		if natsPanel == nil {
			return
		}
		natsPanel.Messages.SetItems(toMessageItems(natsStore.Messages(natsCollectionID, natsPanel.Subject())))
	}
	natsMessagesView := messages.NewView(
		"Messages for selected subject",
		func(all bool) { refreshNatsMessages() },
		func() { fyne.CurrentApp().Clipboard().SetContent(natsPanel.Messages.Text()) },
		func() {
			natsStore.ClearMessages(natsCollectionID, natsPanel.Subject())
			refreshNatsMessages()
		},
	)
	natsStore.AddMessageListener(func() { fyne.Do(refreshNatsMessages) })

	var syncNatsScriptDot func()
	var markNatsScriptResult func(scriptErr string)

	natsPanel = natsui.NewRequestView(
		func(method constants.NatsMethod, req entity.NatsRequest, event entity.Event) {
			sel := currentSelection(selStore.GetSelection())
			if sel == nil || sel.Kind != entity.SelectionRequest {
				return
			}
			natsCollectionID = sel.CollectionID
			if method == constants.NatsMethodRequest {
				syncNatsScriptDot()
			}
			natsPanel.SetRunning(true)
			natsStore.Run(sel.CollectionID, sel.ItemID, method, req, event, func(err error, scriptErr string) {
				natsPanel.SetRunning(false)
				if method == constants.NatsMethodSubscribe && err == nil {
					natsPanel.SetSubActive(true)
				}
				if method == constants.NatsMethodRequest {
					markNatsScriptResult(scriptErr)
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
		func(name string, req entity.NatsRequest, event entity.Event) {
			putRequestDraft(func(d *entity.RequestDraft) {
				d.Name = name
				d.Request.Nats = &req
				d.Request.Event = event
			})
			syncNatsScriptDot()
		},
		func() {
			sel := currentSelection(selStore.GetSelection())
			if sel != nil {
				drafts.SaveRequest(sel.CollectionID, sel.ItemID)
			}
		},
		natsMessagesView,
	)

	hasNatsScripts := func() bool {
		if natsPanel == nil || natsPanel.GetEvent == nil {
			return false
		}
		ev := natsPanel.GetEvent()
		return len(ev.PreRequest) > 0 || len(ev.PostRequest) > 0
	}
	syncNatsScriptDot = func() {
		if natsPanel == nil || natsPanel.SetScriptIcon == nil {
			return
		}
		if hasNatsScripts() {
			natsPanel.SetScriptIcon(ui.DotGray)
		} else {
			natsPanel.SetScriptIcon(nil)
		}
	}
	markNatsScriptResult = func(scriptErr string) {
		if natsPanel == nil {
			return
		}
		if natsPanel.SetScriptError != nil {
			natsPanel.SetScriptError(scriptErr)
		}
		if !hasNatsScripts() {
			natsPanel.SetScriptIcon(nil)
			return
		}
		if strings.TrimSpace(scriptErr) != "" {
			natsPanel.SetScriptIcon(ui.DotRed)
			return
		}
		natsPanel.SetScriptIcon(ui.DotGreen)
	}
	refreshNatsScriptEnvKeys := func() {
		if natsPanel != nil && natsPanel.SetEnvKeys != nil {
			natsPanel.SetEnvKeys(envStore.ActiveEnvKeys())
		}
	}

	var kafkaPanel *kafkaui.RequestView
	var kafkaCollectionID string
	refreshKafkaMessages := func() {
		if kafkaPanel == nil {
			return
		}
		kafkaPanel.Messages.SetItems(toMessageItems(kafkaStore.Messages(kafkaCollectionID, kafkaPanel.Topic())))
	}
	kafkaMessagesView := messages.NewView(
		"Messages for selected topic",
		func(all bool) { refreshKafkaMessages() },
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
		refreshNatsScriptEnvKeys()
		refreshGrpcScriptEnvKeys()
	}))
	(*envStore.GetItems()).AddListener(binding.NewDataListener(func() {
		refreshNatsScriptEnvKeys()
		refreshGrpcScriptEnvKeys()
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
		sioPanel.CanvasObject,
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
			sioPanel.SetDirty(dirty)
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
			folderSettings.Set(sel.Name, sel.Auth, constants.IsHTTPCollection(sel.CollectionType))
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
			case constants.CollectionTypeNATS:
				natsCollectionID = sel.CollectionID
				natsPanel.Set(d.Request.Nats, d.Name, natsStore.IsSubscribed(sel.CollectionID, sel.ItemID), d.Request.Event)
				natsPanel.SetDirty(dirty)
				refreshNatsScriptEnvKeys()
				syncNatsScriptDot()
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
				switch d.Request.Kind() {
				case constants.RequestKindGRPC:
					grpcPanel.Set(d.Request.Grpc, d.Name, d.Request.Auth, d.Request.Event)
					grpcPanel.SetDirty(dirty)
					syncGrpcScriptDot()
					refreshGrpcScriptEnvKeys()
					show(4)
					if focusName {
						fyne.Do(grpcPanel.FocusName)
					}
				case constants.RequestKindSocketIO:
					sioCollectionID = sel.CollectionID
					sioItemID = sel.ItemID
					connected := sioConnStore.IsConnected(sel.CollectionID, sel.ItemID)
					sioWasConnected = connected
					sioReq := d.Request.SocketIO
					if sioReq != nil {
						cp := *sioReq
						cp.Auth = d.Request.Auth
						sioPanel.Set(&cp, d.Name, connected)
						sioPanel.SetListening(sioConnStore.ListeningEvents(sel.CollectionID, sel.ItemID))
					} else {
						sioPanel.Set(nil, d.Name, connected)
						sioPanel.SetListening(nil)
					}
					sioPanel.SetDirty(dirty)
					refreshSioMessages()
					show(8)
					if focusName {
						fyne.Do(sioPanel.FocusName)
					}
				case constants.RequestKindWS:
					wsCollectionID = sel.CollectionID
					wsItemID = sel.ItemID
					connected := wsConnStore.IsConnected(sel.CollectionID, sel.ItemID)
					wsWasConnected = connected
					wsPanel.Set(d.Request.Ws, d.Name, connected)
					wsPanel.SetDirty(dirty)
					refreshWsMessages()
					show(5)
					if focusName {
						fyne.Do(wsPanel.FocusName)
					}
				default:
					show(3)
					// REST focuses from RestContainer listener
				}
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

func toMessageItems(list []store.StreamMessage) []messages.Item {
	out := make([]messages.Item, len(list))
	for i, m := range list {
		out[i] = messages.Item{Time: m.Time, Dir: m.Dir, Body: m.Body}
	}
	return out
}
