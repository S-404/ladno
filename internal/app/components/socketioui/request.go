package socketioui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/messages"
	"github.com/s-404/ladno/internal/app/components/rest"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
	"github.com/s-404/ladno/internal/app/service"
)

type RequestView struct {
	fyne.CanvasObject
	Set           func(req *entity.SocketIORequest, name string, connected bool)
	Get           func() entity.SocketIORequest
	SetConnecting func(connecting bool)
	SetConnected  func(connected bool)
	SetListening  func(events []string)
	SetDirty      func(dirty bool)
	Messages      *messages.View
	FocusName     func()
}

func NewRequestView(
	onConnect func(req entity.SocketIORequest),
	onDisconnect func(),
	onEmit func(req entity.SocketIORequest),
	onListen func(req entity.SocketIORequest),
	onChange func(name string, req entity.SocketIORequest),
	onSave func(),
	msgPane *messages.View,
) *RequestView {
	var applying = true
	var connected bool
	var connecting bool
	var header *ui.EntityHeader
	var connectBtn *widget.Button
	var emitBtn *widget.Button
	var listenBtn *widget.Button

	requestURL := binding.NewString()
	_ = requestURL.Set("")
	urlInput := ui.NewUrlInput(requestURL)
	urlInput.SetPlaceHolder("http://host or http://host/namespace")
	eventEntry := widget.NewEntry()
	eventEntry.SetPlaceHolder("event name")
	payload := ui.NewEnvMultiLineInput()
	payload.SetPlaceHolder(`{"text":"hello"}`)
	payload.SetMinRowsVisible(8)

	var headersView *rest.RequestHeadersView
	var paramsView *rest.RequestParamsView
	var authPanel *ui.AuthPanel
	var getReq func() entity.SocketIORequest
	var refreshAutoHeaders func()
	var applyConnectBtn func()
	var applyListenBtn func()
	var listeningEvents = map[string]bool{}

	notify := func() {
		if applying || onChange == nil || header == nil || getReq == nil {
			return
		}
		onChange(header.GetName(), getReq())
	}

	header = ui.NewEntityHeader(theme.DocumentIcon(), "Request name", func(string) { notify() }, onSave)
	headersView = rest.NewRequestHeaders(nil, func([]ui.KVRow) { notify() })
	paramsView = rest.NewRequestQueryParams(requestURL)
	authPanel = ui.NewAuthPanel(ui.AuthPanelOptions{
		AllowInherited:   true,
		DisableBasic:     true,
		AllowJSON:        true,
		APIKeyHeaderOnly: true,
		OnChange: func(entity.Auth) {
			if refreshAutoHeaders != nil {
				refreshAutoHeaders()
			}
			notify()
		},
	})

	rowsToVars := func(rows []ui.KVRow) []entity.Variable {
		vars := make([]entity.Variable, 0, len(rows))
		for _, r := range rows {
			if r.Key == "" {
				continue
			}
			vars = append(vars, entity.Variable{Key: r.Key, Value: r.Value, Type: "string"})
		}
		return vars
	}

	getReq = func() entity.SocketIORequest {
		urlVal, _ := requestURL.Get()
		auth := entity.Auth{Type: constants.AuthTypeNoAuth}
		if authPanel != nil {
			auth = authPanel.Get()
		}
		return entity.SocketIORequest{
			URL:      urlVal,
			Headers:  rowsToVars(headersView.GetManual()),
			Auth:     auth,
			AuthJSON: entity.SocketIOAuthJSON(auth, ""),
			Event:    strings.TrimSpace(eventEntry.Text),
			Payload:  payload.Text(),
		}
	}

	refreshAutoHeaders = func() {
		urlVal, _ := requestURL.Get()
		hs := service.HandshakeAutoHeaders(urlVal, "")
		rows := make([]ui.KVRow, 0, len(hs)+4)
		for _, h := range hs {
			row := ui.KVRow{Enabled: true, Key: h.Key, Value: h.Value, Auto: true}
			switch strings.ToLower(h.Key) {
			case "host", "sec-websocket-key":
				row.Value = ""
				row.ValueHint = "calculated at runtime"
			}
			rows = append(rows, row)
		}
		if authPanel != nil {
			for _, h := range entity.AuthGeneratedHeaders(authPanel.Get()) {
				rows = append(rows, ui.KVRow{
					Enabled: true,
					Key:     h.Key,
					Value:   h.Value,
					Auto:    true,
					Secret:  true,
				})
			}
		}
		headersView.SetAuto(rows)
	}

	applyConnectBtn = func() {
		if connecting {
			connectBtn.SetText("Connect")
			connectBtn.Importance = widget.MediumImportance
			connectBtn.Disable()
			emitBtn.Disable()
			listenBtn.Disable()
			return
		}
		connectBtn.Enable()
		if connected {
			connectBtn.SetText("Disconnect")
			connectBtn.Importance = widget.DangerImportance
			emitBtn.Enable()
			listenBtn.Enable()
			applyListenBtn()
			return
		}
		connectBtn.SetText("Connect")
		connectBtn.Importance = widget.MediumImportance
		emitBtn.Disable()
		listenBtn.Disable()
		applyListenBtn()
	}

	applyListenBtn = func() {
		if listenBtn == nil {
			return
		}
		ev := strings.TrimSpace(eventEntry.Text)
		if listeningEvents[ev] && ev != "" {
			listenBtn.SetText("Stop")
			listenBtn.Importance = widget.DangerImportance
			return
		}
		listenBtn.SetText("Listen")
		listenBtn.Importance = widget.MediumImportance
	}

	connectBtn = widget.NewButton("Connect", func() {
		if connecting {
			return
		}
		if connected {
			if onDisconnect != nil {
				onDisconnect()
			}
			return
		}
		if onConnect != nil {
			onConnect(getReq())
		}
	})
	connectBtn.Importance = widget.MediumImportance

	emitBtn = widget.NewButton("Emit", func() {
		if onEmit != nil {
			onEmit(getReq())
		}
	})
	emitBtn.Importance = widget.MediumImportance
	emitBtn.Disable()

	listenBtn = widget.NewButton("Listen", func() {
		if onListen != nil {
			onListen(getReq())
		}
	})
	listenBtn.Importance = widget.MediumImportance
	listenBtn.Disable()

	requestURL.AddListener(binding.NewDataListener(func() {
		refreshAutoHeaders()
		notify()
	}))
	eventEntry.OnChanged = func(string) {
		applyListenBtn()
		if listenBtn != nil {
			listenBtn.Refresh()
		}
		notify()
	}
	payload.OnChanged(func(string) { notify() })
	refreshAutoHeaders()

	eventPanel := container.NewBorder(
		widget.NewForm(widget.NewFormItem("Event", eventEntry)),
		container.NewHBox(emitBtn, listenBtn),
		nil, nil,
		payload,
	)
	requestTabs := container.NewAppTabs(
		container.NewTabItem("Params", paramsView.Object),
		container.NewTabItem("Headers", headersView.Object),
		container.NewTabItem("Auth", authPanel.CanvasObject),
		container.NewTabItem("Event", eventPanel),
	)

	urlRow := container.NewBorder(nil, nil, nil, connectBtn, urlInput)
	requestPanel := container.NewBorder(
		container.NewVBox(header.Object, urlRow),
		nil, nil, nil,
		requestTabs,
	)
	split := container.NewVSplit(
		ui.NewMinSizeBox(fyne.NewSize(200, 80), container.NewPadded(requestPanel)),
		ui.NewMinSizeBox(fyne.NewSize(200, 80), msgPane.Object()),
	)
	split.SetOffset(0.55)

	v := &RequestView{CanvasObject: split, Messages: msgPane}
	v.Get = getReq
	v.SetDirty = header.SetDirty
	v.FocusName = header.FocusName
	v.SetConnecting = func(on bool) {
		connecting = on
		applyConnectBtn()
		connectBtn.Refresh()
		emitBtn.Refresh()
		listenBtn.Refresh()
	}
	v.SetConnected = func(on bool) {
		connected = on
		if on {
			connecting = false
		} else {
			listeningEvents = map[string]bool{}
		}
		applyConnectBtn()
		connectBtn.Refresh()
		emitBtn.Refresh()
		listenBtn.Refresh()
	}
	v.SetListening = func(events []string) {
		listeningEvents = map[string]bool{}
		for _, ev := range events {
			ev = strings.TrimSpace(ev)
			if ev != "" {
				listeningEvents[ev] = true
			}
		}
		applyListenBtn()
		listenBtn.Refresh()
	}
	v.Set = func(req *entity.SocketIORequest, name string, isConnected bool) {
		applying = true
		header.SetName(name)
		if req == nil {
			_ = requestURL.Set("")
			eventEntry.SetText("")
			payload.SetText("")
			authPanel.Set(entity.Auth{Type: constants.AuthTypeNoAuth})
			headersView.SetManual(nil)
		} else {
			urlVal := service.MergeLegacySocketIOURL(req.URL, req.Namespace)
			urlVal = service.MergeSocketIOQuery(urlVal, req.Query)
			_ = requestURL.Set(urlVal)
			eventEntry.SetText(req.Event)
			payload.SetText(req.Payload)
			authPanel.Set(entity.EffectiveSocketIOAuth(req.Auth, req.AuthJSON))
			rows := make([]ui.KVRow, 0, len(req.Headers))
			for _, h := range req.Headers {
				rows = append(rows, ui.KVRow{Enabled: true, Key: h.Key, Value: h.Value})
			}
			headersView.SetManual(rows)
		}
		refreshAutoHeaders()
		applying = false
		v.SetConnecting(false)
		v.SetConnected(isConnected)
	}
	applying = false
	return v
}
