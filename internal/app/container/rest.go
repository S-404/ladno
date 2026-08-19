package container

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
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
	cookieStore := app.Store.Cookie
	requestURL := binding.NewString()
	_ = requestURL.Set("")

	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

	var headers []ui.KVRow
	var applying bool
	var requestInput *rest.RequestInputView
	var paramsView *rest.RequestParamsView
	var bodyView *rest.RequestBodyView
	var headersView *rest.RequestHeadersView
	var previewView *rest.PreviewView
	var authPanel *ui.AuthPanel
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

	var preparedReq func() entity.RestRequest
	var refreshAutoHeaders func()
	preparedReq = func() entity.RestRequest {
		req := buildReq()
		sel := currentSelection(selStore.GetSelection())
		if sel == nil || sel.Kind != entity.SelectionRequest || authPanel == nil {
			return req
		}
		return applyRESTAuth(app, req, sel.CollectionID, sel.ItemID, authPanel.Get())
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
			Name:         header.GetName(),
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
		applying = true
		refreshAutoHeaders()
		applying = false
	}

	send := func() {
		restStore.Send(preparedReq())
	}

	header = ui.NewEntityHeader(theme.DocumentIcon(), "Request name", func(string) {
		flushDraft(true)
	}, func() {
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
	headersView = rest.NewRequestHeaders(nil, func(rows []ui.KVRow) {
		headers = rows
		flushDraft(true)
	})
	bodyView = rest.NewRequestBody(rest.BodyState{Mode: rest.BodyModeRaw}, func(rest.BodyState) {
		flushDraft(true)
	})
	previewView = rest.NewPreviewView(func(showSecrets bool) string {
		return restStore.Preview(preparedReq(), showSecrets)
	})
	authPanel = ui.NewAuthPanel(ui.AuthPanelOptions{
		AllowInherited: true,
		OnChange:       func(entity.Auth) { flushDraft(true) },
	})

	refreshAutoHeaders = func() {
		if headersView == nil || authPanel == nil {
			return
		}
		sel := currentSelection(selStore.GetSelection())
		if sel == nil || sel.Kind != entity.SelectionRequest {
			headersView.SetAuto(nil)
			return
		}
		resolved := resolveRESTAuth(app, sel.CollectionID, sel.ItemID, authPanel.Get())
		auto := entity.AuthGeneratedHeaders(resolved)
		rows := make([]ui.KVRow, 0, len(auto)+1)
		for _, h := range auto {
			rows = append(rows, ui.KVRow{
				Enabled: true,
				Key:     h.Key,
				Value:   h.Value,
				Secret:  true,
			})
		}
		if cookieStore != nil && !ui.KVRowsHaveKey(headers, "Cookie") {
			urlVal, _ := requestURL.Get()
			if cv := cookieStore.CookieHeaderForURL(urlVal); cv != "" {
				rows = append(rows, ui.KVRow{
					Enabled: true,
					Key:     "Cookie",
					Value:   cv,
					Secret:  false,
				})
			}
		}
		headersView.SetAuto(rows)
	}

	requestURL.AddListener(binding.NewDataListener(func() {
		flushDraft(true)
	}))

	cookieJar := rest.NewCookieJarView(
		app.Window,
		func() []entity.Cookie { return cookieStore.List() },
		func() []string { return cookieStore.Domains() },
		func(c entity.Cookie) { cookieStore.Delete(c.Name, c.Domain, c.Path) },
		func() { cookieStore.Clear() },
		func(c entity.Cookie) {
			cookieStore.Update(c)
			refreshAutoHeaders()
		},
		func(prev, next entity.Cookie) {
			cookieStore.Replace(prev, next)
			refreshAutoHeaders()
		},
		func(domain string) { cookieStore.AddDomain(domain) },
		func(domain string) { cookieStore.DeleteDomain(domain) },
		func(domain string) {
			name := uniqueCookieName(cookieStore.List(), domain, "cookie")
			cookieStore.Add(entity.Cookie{
				Name:     name,
				Value:    "",
				Domain:   domain,
				Path:     "/",
				HostOnly: true,
			})
		},
	)
	cookiesDlg := ui.NewModal("Cookies", "Close", cookieJar.Object, app.Window)
	cookieStore.AddListener(func() {
		fyne.Do(func() {
			cookieJar.Refresh()
			refreshAutoHeaders()
		})
	})

	cookiesLink := widget.NewHyperlink("Cookies", nil)
	cookiesLink.OnTapped = func() {
		cookieJar.Refresh()
		if win := app.Window; win != nil {
			sz := win.Canvas().Size()
			cookiesDlg.Resize(fyne.NewSize(sz.Width*0.8, sz.Height*0.8))
		}
		cookiesDlg.Show()
	}

	requestTabs := container.NewAppTabs(
		container.NewTabItem("Params", paramsView.Object),
		container.NewTabItem("Headers", headersView.Object),
		container.NewTabItem("Auth", authPanel.CanvasObject),
		container.NewTabItem("Body", bodyView.Object),
		container.NewTabItem("Preview", previewView.Object),
	)
	requestTabs.OnSelected = func(tab *container.TabItem) {
		if tab != nil && tab.Text == "Preview" {
			previewView.Refresh()
		}
	}

	tabsWithCookies := container.NewBorder(
		nil, nil, nil,
		container.NewBorder(container.NewPadded(cookiesLink), nil, nil, nil, nil),
		requestTabs,
	)

	request := container.NewBorder(
		container.NewVBox(header.Object, requestInput.Object),
		nil, nil, nil,
		container.NewStack(tabsWithCookies),
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
		headersView.SetManual(headerRows)
		authPanel.Set(draft.Auth)
		refreshAutoHeaders()
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
		header.SetName(sel.Name)
		refreshAutoHeaders()
		applying = false
		header.SetDirty(drafts.IsRequestDirty(sel.ItemID))
		if sel.FocusName {
			sel.FocusName = false
			fyne.Do(header.FocusName)
		}
	}))

	drafts.AddDirtyListener(func() {
		sel := currentSelection(selStore.GetSelection())
		if sel == nil || sel.Kind != entity.SelectionRequest {
			return
		}
		fyne.Do(func() {
			header.SetDirty(drafts.IsRequestDirty(sel.ItemID))
			refreshAutoHeaders()
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

func uniqueCookieName(existing []entity.Cookie, domain, base string) string {
	used := map[string]bool{}
	for _, c := range existing {
		if strings.EqualFold(c.Domain, domain) && c.Path == "/" {
			used[c.Name] = true
		}
	}
	if !used[base] {
		return base
	}
	for i := 2; ; i++ {
		name := fmt.Sprintf("%s_%d", base, i)
		if !used[name] {
			return name
		}
	}
}
