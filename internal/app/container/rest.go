package container

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"github.com/s-404/ladno/internal/app/components/rest"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
	"github.com/s-404/ladno/internal/app/entity/shared"
)

func RestContainer(app *shared.App) fyne.CanvasObject {
	restStore := app.Store.Rest
	selStore := app.Store.Selection
	drafts := app.Store.Draft
	envStore := app.Store.Env
	requestURL := binding.NewString()
	_ = requestURL.Set("")

	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

	var headers []ui.KVRow
	var applying bool
	var requestInput *rest.RequestInputView
	var paramsView *rest.RequestParamsView
	var bodyView *rest.RequestBodyView
	var headersTable *ui.KVTable
	var previewView *rest.PreviewView
	var authPanel *ui.AuthPanel
	var nameField *ui.RequestNameField
	var header *ui.EntityHeader
	responseView := rest.NewResponseView()

	buildReq := func() entity.RestRequest {
		urlVal, _ := requestURL.Get()
		body := bodyView.Get()
		req := entity.RestRequest{
			Method:     requestInput.GetMethod(),
			URL:        urlVal,
			PathParams: paramsView.GetPathParams(),
			Headers:    kvRowsToVariables(headers),
			BodyMode:   entity.RestBodyMode(body.Mode),
			RawBody:    body.RawText,
			FormData:   kvRowsToVariables(body.FormRows),
		}
		if req.BodyMode == "" {
			req.BodyMode = entity.RestBodyRaw
		}
		return req
	}

	flushDraft := func(markDirty bool) {
		if applying {
			return
		}
		sel := currentSelection(selStore.GetSelection())
		if sel == nil || sel.Kind != entity.SelectionRequest {
			return
		}
		if constants.NormalizeCollectionType(sel.CollectionType) != constants.CollectionTypeREST {
			return
		}
		rr := buildReq()
		pathVars := make([]entity.Variable, 0, len(rr.PathParams))
		for k, v := range rr.PathParams {
			pathVars = append(pathVars, entity.Variable{Key: k, Value: v})
		}
		d := entity.RequestDraft{
			CollectionID: sel.CollectionID,
			Name:         nameField.Get(),
			Request: entity.ItemRequest{
				Method:   constants.RequestMethod(rr.Method),
				Header:   rr.Headers,
				Auth:     authPanel.Get(),
				Url:      entity.RequestUrl{Raw: rr.URL, Variable: pathVars},
				BodyMode: rr.BodyMode,
				Body:     rr.RawBody,
				FormData: rr.FormData,
			},
		}
		drafts.PutRequestDraft(sel.ItemID, d, markDirty)
		header.SetDirty(drafts.IsRequestDirty(sel.ItemID))
	}

	send := func() {
		restStore.Send(buildReq())
	}

	header = ui.NewEntityHeader("HTTP request", func() {
		sel := currentSelection(selStore.GetSelection())
		if sel == nil || sel.Kind != entity.SelectionRequest {
			return
		}
		flushDraft(true)
		drafts.SaveRequest(sel.CollectionID, sel.ItemID)
		header.SetDirty(false)
	})

	requestInput = rest.NewRequestInput(methods, requestURL, send, func(string) {
		flushDraft(true)
	})
	paramsView = rest.NewRequestParams(requestURL)
	headersTable = rest.NewRequestHeaders(nil, func(rows []ui.KVRow) {
		headers = rows
		flushDraft(true)
	})
	bodyView = rest.NewRequestBody(rest.BodyState{Mode: rest.BodyModeRaw}, func(rest.BodyState) {
		flushDraft(true)
	})
	previewView = rest.NewPreviewView(func() string {
		return restStore.Preview(buildReq())
	})
	authPanel = ui.NewAuthPanel(ui.AuthPanelOptions{
		AllowInherited: true,
		OnChange:       func(entity.Auth) { flushDraft(true) },
	})
	nameField = ui.NewRequestNameField(func(string) { flushDraft(true) })

	requestURL.AddListener(binding.NewDataListener(func() {
		flushDraft(true)
	}))

	requestTabs := container.NewAppTabs(
		container.NewTabItem("Params", paramsView.Object),
		container.NewTabItem("Headers", headersTable),
		container.NewTabItem("Auth", authPanel.CanvasObject),
		container.NewTabItem("Body", bodyView.Object),
		container.NewTabItem("Preview", previewView.Object),
	)
	requestTabs.OnSelected = func(tab *container.TabItem) {
		if tab != nil && tab.Text == "Preview" {
			previewView.Refresh()
		}
	}

	request := container.NewBorder(
		container.NewVBox(header.Object, nameField.Object, requestInput.Object),
		nil, nil, nil,
		container.NewStack(requestTabs),
	)

	split := container.NewVSplit(
		ui.NewMinSizeBox(fyne.NewSize(200, 80), request),
		ui.NewMinSizeBox(fyne.NewSize(200, 80), responseView.Object()),
	)
	split.SetOffset(0.55)

	isSending := restStore.GetIsSending()
	(*isSending).AddListener(binding.NewDataListener(func() {
		sending, _ := (*isSending).Get()
		if sending {
			responseView.SetLoading()
		}
	}))

	restStore.GetResponse().AddListener(binding.NewDataListener(func() {
		val, err := restStore.GetResponse().Get()
		if err != nil || val == nil {
			return
		}
		resp, ok := val.(*entity.RestResponse)
		if !ok {
			return
		}
		responseView.SetResponse(resp)
	}))

	restStore.GetDraft().AddListener(binding.NewDataListener(func() {
		val, err := restStore.GetDraft().Get()
		if err != nil || val == nil {
			return
		}
		draft, ok := val.(*entity.RestDraft)
		if !ok || draft == nil {
			return
		}
		applying = true
		if draft.Method != "" {
			requestInput.SetMethod(draft.Method)
		}
		_ = requestURL.Set(draft.URL)
		headerRows := variablesToKVRows(draft.Headers)
		headers = headerRows
		headersTable.SetRows(headerRows)
		authPanel.Set(draft.Auth)
		mode := rest.BodyModeRaw
		if draft.BodyMode == entity.RestBodyFormData {
			mode = rest.BodyModeFormData
		}
		bodyView.Set(rest.BodyState{
			Mode:     mode,
			RawText:  draft.RawBody,
			FormRows: variablesToKVRows(draft.FormData),
		})
		paramsView.SetPathParams(draft.PathParams)
		applying = false
		if requestTabs.Selected() != nil && requestTabs.Selected().Text == "Preview" {
			previewView.Refresh()
		}
	}))

	selStore.GetSelection().AddListener(binding.NewDataListener(func() {
		sel := currentSelection(selStore.GetSelection())
		if sel == nil || sel.Kind != entity.SelectionRequest {
			return
		}
		if constants.NormalizeCollectionType(sel.CollectionType) != constants.CollectionTypeREST {
			return
		}
		applying = true
		nameField.Set(sel.Name)
		applying = false
		header.SetDirty(drafts.IsRequestDirty(sel.ItemID))
	}))

	drafts.AddDirtyListener(func() {
		sel := currentSelection(selStore.GetSelection())
		if sel == nil || sel.Kind != entity.SelectionRequest {
			return
		}
		fyne.Do(func() {
			header.SetDirty(drafts.IsRequestDirty(sel.ItemID))
		})
	})

	envStore.GetActiveID().AddListener(binding.NewDataListener(func() {
		if requestTabs.Selected() != nil && requestTabs.Selected().Text == "Preview" {
			previewView.Refresh()
		}
	}))

	return split
}

func kvRowsToVariables(rows []ui.KVRow) []entity.Variable {
	out := make([]entity.Variable, 0, len(rows))
	for _, r := range rows {
		if !r.Enabled || r.Key == "" {
			continue
		}
		out = append(out, entity.Variable{Key: r.Key, Value: r.Value})
	}
	return out
}

func variablesToKVRows(vars []entity.Variable) []ui.KVRow {
	rows := make([]ui.KVRow, 0, len(vars))
	for _, v := range vars {
		rows = append(rows, ui.KVRow{Enabled: true, Key: v.Key, Value: v.Value})
	}
	return rows
}
