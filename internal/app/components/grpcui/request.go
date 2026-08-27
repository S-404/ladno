package grpcui

import (
	"io"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/scripttab"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
	"github.com/s-404/ladno/internal/app/service"
)

type RequestView struct {
	fyne.CanvasObject
	Set            func(req *entity.GrpcRequest, name string, auth entity.Auth, event entity.Event)
	Get            func() entity.GrpcRequest
	GetAuth        func() entity.Auth
	GetEvent       func() entity.Event
	SetDirty       func(dirty bool)
	SetSending     func(sending bool)
	SetResponse    func(resp *entity.GrpcResponse)
	SetEnvKeys     func(keys []string)
	SetScriptError func(msg string)
	SetScriptIcon  func(icon fyne.Resource)
	FocusName      func()
}

func NewRequestView(
	win fyne.Window,
	onSend func(req entity.GrpcRequest, auth entity.Auth, event entity.Event),
	onChange func(name string, req entity.GrpcRequest, auth entity.Auth, event entity.Event),
	onSave func(),
) *RequestView {
	var applying bool
	var header *ui.EntityHeader
	var protoFiles []entity.GrpcProtoFile
	var fileNames []string
	var methodOptions []string

	target := ui.NewEnvInput()
	target.SetPlaceHolder("localhost:50051 or {{grpcHost}}:{{grpcPort}}")
	message := ui.NewEnvMultiLineInput()
	message.SetPlaceHolder(`{"id":"1"}`)
	message.SetMinRowsVisible(8)

	var meta *ui.KVTable
	var authPanel *ui.AuthPanel
	var scriptView *scripttab.ScriptView
	var scriptTab *container.TabItem
	var requestTabs *container.AppTabs
	var getReq func() entity.GrpcRequest
	var fileSelect *widget.Select
	var methodBar *widget.Select
	var protoHint *widget.Label
	var sendBtn *widget.Button
	var responseView *ResponseView

	notify := func() {
		if applying || onChange == nil || header == nil || getReq == nil || authPanel == nil || scriptView == nil {
			return
		}
		onChange(header.GetName(), getReq(), authPanel.Get(), scriptView.Get())
	}

	header = ui.NewEntityHeader(theme.DocumentIcon(), "Request name", func(string) { notify() }, onSave)
	meta = ui.NewKVTable(nil, func([]ui.KVRow) { notify() })
	scriptView = scripttab.NewScriptView(func(entity.Event) { notify() })
	authPanel = ui.NewAuthPanel(ui.AuthPanelOptions{
		AllowInherited: true,
		DisableAPIKey:  true,
		OnChange:       func(entity.Auth) { notify() },
	})
	responseView = NewResponseView()

	getReq = func() entity.GrpcRequest {
		rows := meta.GetRows()
		vars := make([]entity.Variable, 0, len(rows))
		for _, r := range rows {
			if r.Key == "" {
				continue
			}
			vars = append(vars, entity.Variable{Key: r.Key, Value: r.Value, Type: "string"})
		}
		active := ""
		if fileSelect != nil {
			active = fileSelect.Selected
		}
		method := ""
		if methodBar != nil {
			method = methodBar.Selected
		}
		return entity.GrpcRequest{
			Target:      target.Text(),
			Method:      method,
			ProtoFiles:  append([]entity.GrpcProtoFile(nil), protoFiles...),
			ActiveProto: active,
			Metadata:    vars,
			Message:     message.Text(),
		}
	}

	refreshMethods := func() {
		active := ""
		if fileSelect != nil {
			active = fileSelect.Selected
		}
		src := ""
		for _, f := range protoFiles {
			if f.Name == active {
				src = f.Content
				break
			}
		}
		methodOptions = service.ProtoMethodNames(src)
		current := ""
		if methodBar != nil {
			current = methodBar.Selected
		}
		opts := withCurrentMethod(methodOptions, current)
		setSelectOptions(methodBar, opts, current)
		if protoHint == nil {
			return
		}
		if len(protoFiles) == 0 {
			protoHint.SetText("Import a .proto file to list methods.")
			return
		}
		if src == "" {
			protoHint.SetText("Select an imported file.")
			return
		}
		if len(methodOptions) == 0 {
			protoHint.SetText("No rpc methods found in this file.")
			return
		}
		protoHint.SetText("")
	}

	fileSelect = widget.NewSelect(nil, func(string) {
		if applying {
			return
		}
		refreshMethods()
		notify()
	})
	fileSelect.PlaceHolder = "Imported files"

	methodBar = widget.NewSelect(nil, func(string) {
		if applying {
			return
		}
		notify()
	})
	methodBar.PlaceHolder = "Method"

	importBtn := widget.NewButton("Import .proto", func() {
		if win == nil {
			return
		}
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			uri := reader.URI()
			data, readErr := io.ReadAll(reader)
			_ = reader.Close()
			if readErr != nil {
				return
			}
			path := ""
			name := "imported.proto"
			if uri != nil {
				path = uri.Path()
				if path == "" {
					path = strings.TrimPrefix(uri.String(), "file://")
				}
				if base := filepath.Base(path); base != "" && base != "." {
					name = base
				}
			}
			replaced := false
			for i, f := range protoFiles {
				if f.Name == name {
					protoFiles[i] = entity.GrpcProtoFile{Name: name, Path: path, Content: string(data)}
					replaced = true
					break
				}
			}
			if !replaced {
				protoFiles = append(protoFiles, entity.GrpcProtoFile{Name: name, Path: path, Content: string(data)})
			}
			fileNames = protoFileNames(protoFiles)
			applying = true
			fileSelect.SetOptions(fileNames)
			fileSelect.SetSelected(name)
			applying = false
			refreshMethods()
			notify()
		}, win)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".proto"}))
		fd.Show()
		fd.Resize(fyne.NewSize(800, 600))
	})

	protoHint = widget.NewLabel("Import a .proto file to list methods.")
	protoHint.TextStyle = fyne.TextStyle{Italic: true}
	protoHint.Wrapping = fyne.TextWrapWord

	servicePanel := container.NewPadded(container.NewVBox(
		importBtn,
		widget.NewForm(
			widget.NewFormItem("Imported files", fileSelect),
		),
		protoHint,
	))
	metaPanel := container.NewBorder(nil, nil, nil, nil, ui.NewListVScroll(container.NewVBox(meta)))
	messagePanel := container.NewBorder(nil, nil, nil, nil, message)
	scriptTab = container.NewTabItem("Script", scriptView.Object)
	requestTabs = container.NewAppTabs(
		container.NewTabItem("Service", servicePanel),
		container.NewTabItem("Auth", authPanel.CanvasObject),
		container.NewTabItem("Metadata", metaPanel),
		container.NewTabItem("Message", messagePanel),
		scriptTab,
	)

	sendBtn = widget.NewButton("Send", func() {
		if onSend != nil {
			onSend(getReq(), authPanel.Get(), scriptView.Get())
		}
	})
	sendBtn.Importance = widget.MediumImportance

	urlRow := container.NewBorder(nil, nil, nil, sendBtn,
		container.NewGridWithColumns(2, target, methodBar),
	)
	requestPanel := container.NewBorder(
		container.NewVBox(header.Object, urlRow),
		nil, nil, nil,
		requestTabs,
	)
	split := container.NewVSplit(
		ui.NewMinSizeBox(fyne.NewSize(200, 80), container.NewPadded(requestPanel)),
		ui.NewMinSizeBox(fyne.NewSize(200, 80), responseView.Object()),
	)
	split.SetOffset(0.55)

	target.OnChanged(func(string) { notify() })
	message.OnChanged(func(string) { notify() })

	v := &RequestView{CanvasObject: split}
	v.Get = getReq
	v.GetAuth = authPanel.Get
	v.GetEvent = scriptView.Get
	v.SetDirty = header.SetDirty
	v.FocusName = header.FocusName
	v.SetEnvKeys = scriptView.SetEnvKeys
	v.SetScriptError = scriptView.SetError
	v.SetScriptIcon = func(icon fyne.Resource) {
		scriptTab.Icon = icon
		scriptTab.Text = "Script"
		requestTabs.Refresh()
	}
	v.SetSending = func(sending bool) {
		if sending {
			sendBtn.Disable()
			responseView.SetLoading()
			return
		}
		sendBtn.Enable()
	}
	v.SetResponse = responseView.SetResponse
	v.Set = func(req *entity.GrpcRequest, name string, auth entity.Auth, event entity.Event) {
		applying = true
		header.SetName(name)
		if req == nil {
			target.SetText("")
			message.SetText("")
			meta.SetRows(nil)
			protoFiles = nil
			fileNames = nil
			fileSelect.SetOptions(nil)
			fileSelect.ClearSelected()
			setSelectOptions(methodBar, nil, "")
			authPanel.Set(entity.Auth{Type: constants.AuthTypeNoAuth})
			scriptView.Set(entity.Event{})
		} else {
			target.SetText(req.Target)
			message.SetText(req.Message)
			rows := make([]ui.KVRow, 0, len(req.Metadata))
			for _, m := range req.Metadata {
				rows = append(rows, ui.KVRow{Enabled: true, Key: m.Key, Value: m.Value})
			}
			meta.SetRows(rows)
			protoFiles = append([]entity.GrpcProtoFile(nil), req.ProtoFiles...)
			fileNames = protoFileNames(protoFiles)
			fileSelect.SetOptions(fileNames)
			active := req.ActiveProto
			if active == "" && len(fileNames) > 0 {
				active = fileNames[0]
			}
			if active != "" {
				fileSelect.SetSelected(active)
			} else {
				fileSelect.ClearSelected()
			}
			methodOptions = nil
			src := ""
			for _, f := range protoFiles {
				if f.Name == active {
					src = f.Content
					break
				}
			}
			methodOptions = service.ProtoMethodNames(src)
			opts := withCurrentMethod(methodOptions, req.Method)
			setSelectOptions(methodBar, opts, req.Method)
			authPanel.Set(auth)
			scriptView.Set(event)
		}
		scriptView.SetError("")
		refreshMethods()
		applying = false
	}
	return v
}

func protoFileNames(files []entity.GrpcProtoFile) []string {
	out := make([]string, 0, len(files))
	for _, f := range files {
		if f.Name != "" {
			out = append(out, f.Name)
		}
	}
	return out
}

func withCurrentMethod(methods []string, current string) []string {
	current = strings.TrimSpace(current)
	if current == "" {
		return methods
	}
	for _, m := range methods {
		if m == current {
			return methods
		}
	}
	return append([]string{current}, methods...)
}

func setSelectOptions(sel *widget.Select, opts []string, current string) {
	if sel == nil {
		return
	}
	sel.SetOptions(opts)
	if current != "" {
		sel.SetSelected(current)
		return
	}
	if len(opts) > 0 {
		sel.SetSelected(opts[0])
		return
	}
	sel.ClearSelected()
}
