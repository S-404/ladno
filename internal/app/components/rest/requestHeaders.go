package rest

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/ui"
)

// RequestHeadersView — headers + auto-generated из Auth в одном списке.
type RequestHeadersView struct {
	Object    fyne.CanvasObject
	SetManual func(rows []ui.KVRow)
	GetManual func() []ui.KVRow
	SetAuto   func(rows []ui.KVRow)
}

// NewRequestHeaders возвращает таб Headers.
func NewRequestHeaders(initial []ui.KVRow, onChange func(rows []ui.KVRow)) *RequestHeadersView {
	manual := cloneRows(initial)
	var auto []ui.KVRow
	showAuto := true
	var syncing bool

	warnHint := widget.NewLabel("⚠ Duplicate of auto-generated — will be overwritten by Auth on send.")
	warnHint.Importance = widget.WarningImportance
	warnHint.Wrapping = fyne.TextWrapWord
	warnHint.Hide()

	autoKeys := func() []string {
		keys := make([]string, 0, len(auto))
		for _, r := range auto {
			if strings.TrimSpace(r.Key) != "" {
				keys = append(keys, r.Key)
			}
		}
		return keys
	}

	markWarns := func() {
		keys := autoKeys()
		any := false
		for i := range manual {
			manual[i].Warn = ui.KeyConflicts(manual[i].Key, keys)
			if manual[i].Warn {
				any = true
			}
		}
		if any {
			warnHint.Show()
		} else {
			warnHint.Hide()
		}
	}

	var table *ui.KVTable
	rebuild := func() {
		syncing = true
		markWarns()
		rows := make([]ui.KVRow, 0, len(auto)+len(manual))
		if showAuto {
			for _, r := range auto {
				r.Auto = true
				r.Enabled = true
				r.Warn = false
				rows = append(rows, r)
			}
		}
		for _, r := range manual {
			r.Auto = false
			rows = append(rows, r)
		}
		table.SetRows(rows)
		table.SetConflictKeys(autoKeys())
		syncing = false
	}

	table = ui.NewKVTable(nil, func(rows []ui.KVRow) {
		if syncing {
			return
		}
		next := make([]ui.KVRow, 0, len(rows))
		for _, r := range rows {
			if r.Auto {
				continue
			}
			r.Auto = false
			next = append(next, r)
		}
		manual = next
		markWarns()
		if anyWarn(manual) {
			warnHint.Show()
		} else {
			warnHint.Hide()
		}
		table.SetConflictKeys(autoKeys())
		if onChange != nil {
			onChange(cloneRows(manual))
		}
	})

	showCheck := widget.NewCheck("Show auto-generated", func(v bool) {
		showAuto = v
		rebuild()
	})
	showCheck.Checked = true

	content := container.NewVBox(showCheck, warnHint, table)
	scroll := container.NewVScroll(content)
	rebuild()

	v := &RequestHeadersView{Object: scroll}
	v.SetManual = func(rows []ui.KVRow) {
		manual = cloneRows(rows)
		rebuild()
	}
	v.GetManual = func() []ui.KVRow {
		return cloneRows(manual)
	}
	v.SetAuto = func(rows []ui.KVRow) {
		auto = cloneRows(rows)
		for i := range auto {
			auto[i].Auto = true
			auto[i].Enabled = true
		}
		showCheck.Enable()
		if len(auto) == 0 {
			showCheck.Disable()
		}
		rebuild()
	}
	return v
}

func anyWarn(rows []ui.KVRow) bool {
	for _, r := range rows {
		if r.Warn {
			return true
		}
	}
	return false
}

func cloneRows(rows []ui.KVRow) []ui.KVRow {
	if rows == nil {
		return []ui.KVRow{}
	}
	out := make([]ui.KVRow, len(rows))
	copy(out, rows)
	return out
}
