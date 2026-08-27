package rest

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/ui"
)

// RequestParamsView — таб Params с path/query.
type RequestParamsView struct {
	Object        fyne.CanvasObject
	GetPathParams func() map[string]string
	SetPathParams func(vals map[string]string)
}

// NewRequestParams возвращает таб Params:
//   - Path Variables — только если в URL есть :param (без checkbox)
//   - Query Params — сразу после path
//
// Общий скролл, высота не ограничена (как логи).
func NewRequestParams(requestURL binding.String, onPathChange func()) *RequestParamsView {
	return newRequestParams(requestURL, onPathChange, true)
}

// NewRequestQueryParams — таб Params только с Query Params (без path variables).
func NewRequestQueryParams(requestURL binding.String) *RequestParamsView {
	return newRequestParams(requestURL, nil, false)
}

func newRequestParams(requestURL binding.String, onPathChange func(), includePath bool) *RequestParamsView {
	var syncing bool
	var pathParamNames []string

	pathTable := ui.NewKVTablePathVars(nil, func([]ui.KVRow) {
		if syncing || onPathChange == nil {
			return
		}
		onPathChange()
	})

	var queryRows []ui.KVRow
	queryTable := ui.NewKVTable(nil, func(rows []ui.KVRow) {
		if syncing || requestURL == nil {
			return
		}
		queryRows = rows
		syncing = true
		defer func() { syncing = false }()
		raw, _ := requestURL.Get()
		_ = requestURL.Set(applyQueryParams(raw, rows))
	})

	pathHeader := sectionLabel("Path Variables")
	pathSection := container.NewVBox(pathHeader, pathTable)
	pathSection.Hide()

	querySection := container.NewVBox(
		sectionLabel("Query Params"),
		queryTable,
	)

	var content *fyne.Container
	if includePath {
		content = container.NewVBox(pathSection, querySection)
	} else {
		content = container.NewVBox(querySection)
	}
	scroll := ui.NewListVScroll(content)

	updatePathVisibility := func(names []string) {
		if len(names) == 0 {
			pathSection.Hide()
			return
		}
		pathSection.Show()
	}

	rebuildPathRows := func(names []string, vals map[string]string) {
		pathParamNames = names
		newRows := make([]ui.KVRow, len(names))
		for i, name := range names {
			val := ""
			if vals != nil {
				val = vals[name]
			}
			newRows[i] = ui.KVRow{
				Enabled: true,
				Key:     name,
				Value:   val,
			}
		}
		syncing = true
		pathTable.SetRows(newRows)
		syncing = false
		updatePathVisibility(names)
	}

	if requestURL != nil {
		requestURL.AddListener(binding.NewDataListener(func() {
			if syncing {
				return
			}
			raw, _ := requestURL.Get()

			if includePath {
				newNames := extractPathParamNames(raw)
				if !equalStringSlices(newNames, pathParamNames) {
					// при ручном редактировании URL сохраняем уже введённые значения
					existingVals := map[string]string{}
					for _, r := range pathTable.GetRows() {
						existingVals[r.Key] = r.Value
					}
					rebuildPathRows(newNames, existingVals)
				}
			}

			newQueryRows := parseQueryOrdered(raw)
			merged := mergeQueryRows(queryRows, newQueryRows)
			if !equalKVRows(merged, queryRows) {
				queryRows = merged
				syncing = true
				queryTable.SetRows(merged)
				syncing = false
			}
		}))
	}

	return &RequestParamsView{
		Object: container.NewBorder(nil, nil, nil, nil, scroll),
		GetPathParams: func() map[string]string {
			vals := map[string]string{}
			if !includePath {
				return vals
			}
			for _, r := range pathTable.GetRows() {
				if r.Key != "" {
					vals[r.Key] = r.Value
				}
			}
			return vals
		},
		// SetPathParams полностью заменяет значения path variables (без наследования от предыдущего запроса).
		SetPathParams: func(vals map[string]string) {
			if !includePath {
				return
			}
			if vals == nil {
				vals = map[string]string{}
			}
			raw := ""
			if requestURL != nil {
				raw, _ = requestURL.Get()
			}
			rebuildPathRows(extractPathParamNames(raw), vals)
		},
	}
}

func skipAuthority(s string) int {
	idx := strings.Index(s, "://")
	if idx == -1 {
		return 0
	}
	i := idx + 3
	for i < len(s) && s[i] != '/' {
		i++
	}
	return i
}

func extractPathParamNames(rawURL string) []string {
	path, _, _ := strings.Cut(rawURL, "?")
	start := skipAuthority(path)

	var names []string
	seen := map[string]bool{}
	i := start
	for i < len(path) {
		if i+1 < len(path) && path[i] == '{' && path[i+1] == '{' {
			end := strings.Index(path[i:], "}}")
			if end == -1 {
				break
			}
			i += end + 2
			continue
		}
		if path[i] == ':' {
			j := i + 1
			for j < len(path) && isParamChar(path[j]) {
				j++
			}
			if j > i+1 {
				name := path[i+1 : j]
				if !seen[name] {
					seen[name] = true
					names = append(names, name)
				}
				i = j
				continue
			}
		}
		i++
	}
	return names
}

func applyQueryParams(rawURL string, rows []ui.KVRow) string {
	base, _, _ := strings.Cut(rawURL, "?")

	var parts []string
	for _, r := range rows {
		if r.Enabled && r.Key != "" {
			parts = append(parts, r.Key+"="+r.Value)
		}
	}

	if len(parts) == 0 {
		return base
	}
	return base + "?" + strings.Join(parts, "&")
}

func parseQueryOrdered(rawURL string) []ui.KVRow {
	_, query, found := strings.Cut(rawURL, "?")
	if !found || query == "" {
		return nil
	}

	var rows []ui.KVRow
	for _, part := range strings.Split(query, "&") {
		if part == "" {
			continue
		}
		k, v, _ := strings.Cut(part, "=")
		rows = append(rows, ui.KVRow{Enabled: true, Key: k, Value: v})
	}
	return rows
}

func mergeQueryRows(old, new []ui.KVRow) []ui.KVRow {
	newCount := map[string]int{}
	for _, r := range new {
		newCount[r.Key]++
	}
	oldCount := map[string]int{}
	for _, r := range old {
		oldCount[r.Key]++
	}

	newVals := map[string][]string{}
	for _, r := range new {
		newVals[r.Key] = append(newVals[r.Key], r.Value)
	}

	if equalCountMaps(newCount, oldCount) {
		usedIdx := map[string]int{}
		result := make([]ui.KVRow, len(old))
		for i, r := range old {
			idx := usedIdx[r.Key]
			val := r.Value
			if idx < len(newVals[r.Key]) {
				val = newVals[r.Key][idx]
			}
			result[i] = ui.KVRow{Enabled: r.Enabled, Key: r.Key, Value: val}
			usedIdx[r.Key]++
		}
		return result
	}

	used := map[string]int{}
	var result []ui.KVRow

	for _, r := range old {
		if newCount[r.Key] > used[r.Key] {
			idx := used[r.Key]
			val := r.Value
			if idx < len(newVals[r.Key]) {
				val = newVals[r.Key][idx]
			}
			result = append(result, ui.KVRow{Enabled: r.Enabled, Key: r.Key, Value: val})
			used[r.Key]++
		}
	}

	addedFromNew := map[string]bool{}
	for _, r := range new {
		if oldCount[r.Key] == 0 && !addedFromNew[r.Key] {
			result = append(result, r)
			addedFromNew[r.Key] = true
		}
	}

	return result
}

func sectionLabel(text string) fyne.CanvasObject {
	return widget.NewLabelWithStyle(text, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
}

func isParamChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_' || c == '-'
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalKVRows(a, b []ui.KVRow) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalCountMaps(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
