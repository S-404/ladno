package wsui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/rest"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/service"
)

type RequestView struct {
	fyne.CanvasObject
	Set           func(req *entity.WsRequest, name string, connected bool)
	Get           func() entity.WsRequest
	SetConnecting func(connecting bool)
	SetConnected  func(connected bool)
	SetStatus     func(status string)
	SetDirty      func(dirty bool)
	Messages      *MessagesView
	FocusName     func()
}

func NewRequestView(
	onConnect func(req entity.WsRequest),
	onDisconnect func(),
	onSend func(req entity.WsRequest),
	onChange func(name string, req entity.WsRequest),
	onSave func(),
	messages *MessagesView,
) *RequestView {
	var applying = true
	var connected bool
	var connecting bool
	var header *ui.EntityHeader
	var connectBtn *widget.Button
	var sendBtn *widget.Button

	statusLabel := widget.NewLabel("")
	statusLabel.TextStyle = fyne.TextStyle{Italic: true}
	statusLabel.Hide()

	requestURL := binding.NewString()
	_ = requestURL.Set("")
	urlInput := ui.NewUrlInput(requestURL)
	urlInput.SetPlaceHolder("ws://host/path or wss://host/path")

	message := ui.NewEnvMultiLineInput()
	message.SetPlaceHolder("Message")
	message.SetMinRowsVisible(8)

	var headersView *rest.RequestHeadersView
	var paramsView *rest.RequestParamsView
	var getReq func() entity.WsRequest
	var refreshAutoHeaders func()
	var applyConnectBtn func()

	notify := func() {
		if applying || onChange == nil || header == nil || getReq == nil {
			return
		}
		onChange(header.GetName(), getReq())
	}

	header = ui.NewEntityHeader(theme.DocumentIcon(), "Request name", func(string) { notify() }, onSave)

	headersView = rest.NewRequestHeaders(nil, func([]ui.KVRow) { notify() })
	paramsView = rest.NewRequestParams(requestURL, notify)

	getReq = func() entity.WsRequest {
		urlVal, _ := requestURL.Get()
		rows := headersView.GetManual()
		vars := make([]entity.Variable, 0, len(rows))
		for _, r := range rows {
			if r.Key == "" {
				continue
			}
			vars = append(vars, entity.Variable{Key: r.Key, Value: r.Value, Type: "string"})
		}
		path := paramsView.GetPathParams()
		pathVars := make([]entity.Variable, 0, len(path))
		for k, v := range path {
			pathVars = append(pathVars, entity.Variable{Key: k, Value: v, Type: "string"})
		}
		return entity.WsRequest{
			URL:        urlVal,
			PathParams: pathVars,
			Headers:    vars,
			Message:    message.Text(),
		}
	}

	refreshAutoHeaders = func() {
		urlVal, _ := requestURL.Get()
		hs := service.HandshakeAutoHeaders(urlVal, "")
		rows := make([]ui.KVRow, 0, len(hs))
		for _, h := range hs {
			row := ui.KVRow{Enabled: true, Key: h.Key, Value: h.Value, Auto: true}
			switch strings.ToLower(h.Key) {
			case "host", "sec-websocket-key":
				row.Value = ""
				row.ValueHint = "calculated at runtime"
			}
			rows = append(rows, row)
		}
		headersView.SetAuto(rows)
	}

	applyConnectBtn = func() {
		if connecting {
			connectBtn.SetText("Connect")
			connectBtn.Importance = widget.MediumImportance
			connectBtn.Disable()
			sendBtn.Disable()
			return
		}
		connectBtn.Enable()
		if connected {
			connectBtn.SetText("Disconnect")
			connectBtn.Importance = widget.DangerImportance
			sendBtn.Enable()
			return
		}
		connectBtn.SetText("Connect")
		connectBtn.Importance = widget.MediumImportance
		sendBtn.Disable()
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

	sendBtn = widget.NewButton("Send", func() {
		if onSend != nil {
			onSend(getReq())
		}
	})
	sendBtn.Importance = widget.MediumImportance
	sendBtn.Disable()

	requestURL.AddListener(binding.NewDataListener(func() {
		refreshAutoHeaders()
		notify()
	}))
	message.OnChanged(func(string) { notify() })
	refreshAutoHeaders()

	headersPanel := headersView.Object
	paramsPanel := paramsView.Object
	messagePanel := container.NewBorder(
		nil,
		container.NewHBox(sendBtn),
		nil, nil,
		message,
	)
	requestTabs := container.NewAppTabs(
		container.NewTabItem("Params", paramsPanel),
		container.NewTabItem("Headers", headersPanel),
		container.NewTabItem("Message", messagePanel),
	)

	urlRow := container.NewBorder(nil, nil, nil, connectBtn, urlInput)
	requestPanel := container.NewBorder(
		container.NewVBox(header.Object, urlRow, statusLabel),
		nil, nil, nil,
		requestTabs,
	)
	split := container.NewVSplit(
		ui.NewMinSizeBox(fyne.NewSize(200, 80), container.NewPadded(requestPanel)),
		ui.NewMinSizeBox(fyne.NewSize(200, 80), messages.Object()),
	)
	split.SetOffset(0.55)

	v := &RequestView{CanvasObject: split, Messages: messages}
	v.Get = getReq
	v.SetDirty = header.SetDirty
	v.FocusName = header.FocusName
	v.SetStatus = func(status string) {
		statusLabel.SetText(status)
		if strings.TrimSpace(status) == "" {
			statusLabel.Hide()
		} else {
			statusLabel.Show()
		}
		statusLabel.Refresh()
	}
	v.SetConnecting = func(on bool) {
		connecting = on
		applyConnectBtn()
		connectBtn.Refresh()
		sendBtn.Refresh()
	}
	v.SetConnected = func(on bool) {
		connected = on
		if on {
			connecting = false
		}
		applyConnectBtn()
		connectBtn.Refresh()
		sendBtn.Refresh()
	}
	v.Set = func(req *entity.WsRequest, name string, isConnected bool) {
		applying = true
		header.SetName(name)
		if req == nil {
			_ = requestURL.Set("")
			message.SetText("")
			headersView.SetManual(nil)
			paramsView.SetPathParams(nil)
		} else {
			_ = requestURL.Set(req.URL)
			message.SetText(req.Message)
			rows := make([]ui.KVRow, 0, len(req.Headers))
			for _, h := range req.Headers {
				rows = append(rows, ui.KVRow{Enabled: true, Key: h.Key, Value: h.Value})
			}
			headersView.SetManual(rows)
			path := map[string]string{}
			for _, p := range req.PathParams {
				if p.Key != "" {
					path[p.Key] = p.Value
				}
			}
			paramsView.SetPathParams(path)
		}
		refreshAutoHeaders()
		applying = false
		v.SetConnecting(false)
		v.SetConnected(isConnected)
		if isConnected {
			v.SetStatus("Connected")
		} else {
			v.SetStatus("")
		}
	}
	applying = false
	return v
}
