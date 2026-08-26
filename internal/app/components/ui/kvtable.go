package ui

import (
	"image/color"
	"path/filepath"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/entity/constants"
)

// KVRow — одна строка таблицы
type KVRow struct {
	Enabled bool
	Key     string
	Value   string
	// Type — для form-data: "text" | "file" (пусто = text).
	Type string
	// Auto — auto-generated (read-only, no delete/reorder).
	Auto bool
	// Secret — mask value (password field).
	Secret bool
	// Warn — highlight overwrite conflict (e.g. duplicates auto-generated header).
	Warn bool
}

// KVTableOptions — поведение таблицы.
type KVTableOptions struct {
	ShowAdd       bool
	ShowCheck     bool
	ShowDelete    bool
	KeyReadOnly   bool
	ValueReadOnly bool
	Reorderable   bool
	// ShowType — колонка Type (text/file) для form-data.
	ShowType bool
	// ShowSecret — кнопка isSecret: маскирует value (для env variables).
	ShowSecret bool
	// Window — для file open dialog (нужен при ShowType).
	Window fyne.Window
}

// KVTable — таблица key/value без внутреннего скролла (растёт по контенту).
type KVTable struct {
	widget.BaseWidget

	mu       sync.Mutex
	rows     []KVRow
	onChange func(rows []KVRow)
	opts     KVTableOptions
	// conflictKeys — keys that conflict with auto-generated (case-insensitive).
	conflictKeys []string
	usedKeys     map[string]struct{}

	root       *fyne.Container
	rowsBox    *fyne.Container
	rowWidgets []*kvDragRow

	dragActive    bool
	dragSourceIdx int
	dropInsertIdx int
	dropLineIdx   int
	dropBefore    bool
	ghost         DragGhost
}

func NewKVTable(initial []KVRow, onChange func(rows []KVRow)) *KVTable {
	return newKVTable(initial, onChange, KVTableOptions{
		ShowAdd:     true,
		ShowCheck:   true,
		ShowDelete:  true,
		Reorderable: true,
	})
}

// NewKVTableEnv — env variables: enabled + key/value + isSecret toggle.
func NewKVTableEnv(initial []KVRow, onChange func(rows []KVRow)) *KVTable {
	return newKVTable(initial, onChange, KVTableOptions{
		ShowAdd:     true,
		ShowCheck:   true,
		ShowDelete:  true,
		Reorderable: true,
		ShowSecret:  true,
	})
}

// NewKVTableFormData — form-data: Key / Type / Value, с выбором файла.
func NewKVTableFormData(initial []KVRow, onChange func(rows []KVRow), win fyne.Window) *KVTable {
	return newKVTable(initial, onChange, KVTableOptions{
		ShowAdd:     true,
		ShowCheck:   true,
		ShowDelete:  true,
		Reorderable: true,
		ShowType:    true,
		Window:      win,
	})
}

// NewKVTablePathVars — path variables: без checkbox/delete/add, key read-only.
func NewKVTablePathVars(initial []KVRow, onChange func(rows []KVRow)) *KVTable {
	return newKVTable(initial, onChange, KVTableOptions{
		KeyReadOnly: true,
	})
}

func NewKVTableReadOnly(initial []KVRow, onChange func(rows []KVRow)) *KVTable {
	return newKVTable(initial, onChange, KVTableOptions{
		ShowCheck:   true,
		ShowDelete:  true,
		Reorderable: true,
	})
}

func newKVTable(initial []KVRow, onChange func(rows []KVRow), opts KVTableOptions) *KVTable {
	if initial == nil {
		initial = []KVRow{}
	}
	rowsBox := container.NewVBox()
	t := &KVTable{
		rows:          initial,
		onChange:      onChange,
		opts:          opts,
		root:          container.NewVBox(),
		rowsBox:       rowsBox,
		dragSourceIdx: -1,
		dropInsertIdx: -1,
		dropLineIdx:   -1,
	}
	t.ExtendBaseWidget(t)
	t.rebuild()
	return t
}

func (t *KVTable) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.root)
}

func (t *KVTable) SetRows(rows []KVRow) {
	if rows == nil {
		rows = []KVRow{}
	}
	t.mu.Lock()
	t.rows = rows
	t.mu.Unlock()
	t.rebuild()
	t.Refresh()
}

func (t *KVTable) GetRows() []KVRow {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]KVRow, len(t.rows))
	copy(out, t.rows)
	return out
}

// SetConflictKeys marks non-auto rows whose Key matches (case-insensitive) for Warn.
func (t *KVTable) SetConflictKeys(keys []string) {
	t.mu.Lock()
	t.conflictKeys = append([]string{}, keys...)
	changed := false
	for i := range t.rows {
		if t.rows[i].Auto {
			continue
		}
		w := KeyConflicts(t.rows[i].Key, t.conflictKeys)
		if t.rows[i].Warn != w {
			t.rows[i].Warn = w
			changed = true
		}
	}
	t.mu.Unlock()
	if changed {
		t.rebuild()
		t.Refresh()
	}
}

// SetUsedKeys highlights rows whose Key is referenced as {{key}} in the selected request.
func (t *KVTable) SetUsedKeys(keys []string) {
	next := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		if k == "" {
			continue
		}
		next[k] = struct{}{}
	}
	t.mu.Lock()
	changed := len(next) != len(t.usedKeys)
	if !changed {
		for k := range next {
			if _, ok := t.usedKeys[k]; !ok {
				changed = true
				break
			}
		}
	}
	t.usedKeys = next
	t.mu.Unlock()
	if changed {
		t.rebuild()
		t.Refresh()
	}
}

func (t *KVTable) notify() {
	if t.onChange == nil {
		return
	}
	t.mu.Lock()
	cp := make([]KVRow, len(t.rows))
	copy(cp, t.rows)
	t.mu.Unlock()
	t.onChange(cp)
}

func (t *KVTable) addRow() {
	t.mu.Lock()
	row := KVRow{Enabled: true}
	if t.opts.ShowType {
		row.Type = "text"
	}
	t.rows = append(t.rows, row)
	t.mu.Unlock()
	t.rebuild()
	t.Refresh()
	t.notify()
}

func (t *KVTable) deleteRow(idx int) {
	t.mu.Lock()
	if idx < 0 || idx >= len(t.rows) || t.rows[idx].Auto {
		t.mu.Unlock()
		return
	}
	t.rows = append(t.rows[:idx], t.rows[idx+1:]...)
	t.mu.Unlock()
	t.rebuild()
	t.Refresh()
	t.notify()
}

func (t *KVTable) moveRow(from, to int) {
	t.mu.Lock()
	if from < 0 || to < 0 || from >= len(t.rows) || to >= len(t.rows) || from == to {
		t.mu.Unlock()
		return
	}
	if t.rows[from].Auto || t.rows[to].Auto {
		t.mu.Unlock()
		return
	}
	item := t.rows[from]
	t.rows = append(t.rows[:from], t.rows[from+1:]...)
	t.rows = append(t.rows[:to], append([]KVRow{item}, t.rows[to:]...)...)
	t.mu.Unlock()
	t.rebuild()
	t.Refresh()
	t.notify()
}

func (t *KVTable) rebuild() {
	var addAction fyne.CanvasObject = widget.NewLabel("")
	if t.opts.ShowAdd {
		addBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
			t.addRow()
		})
		addBtn.Importance = widget.LowImportance
		addAction = addBtn
	}

	t.mu.Lock()
	n := len(t.rows)
	t.mu.Unlock()

	t.rowWidgets = make([]*kvDragRow, 0, n)
	objs := make([]fyne.CanvasObject, 0, n)
	for i := 0; i < n; i++ {
		idx := i
		row := t.makeRow(idx)
		objs = append(objs, row)
	}
	t.rowsBox.Objects = objs
	t.rowsBox.Refresh()

	t.root.Objects = []fyne.CanvasObject{
		container.NewBorder(
			kvHeader(addAction, t.opts),
			nil, nil, nil,
			t.rowsBox,
		),
	}
	t.root.Refresh()
	t.refreshDropIndicators()
}

func (t *KVTable) makeRow(idx int) fyne.CanvasObject {
	t.mu.Lock()
	row := t.rows[idx]
	_, used := t.usedKeys[row.Key]
	t.mu.Unlock()

	var keyObj fyne.CanvasObject
	if t.opts.ShowSecret {
		keyEntry := NewEntry()
		keyEntry.SetPlaceHolder("Key")
		keyEntry.SetText(row.Key)
		if t.opts.KeyReadOnly || row.Auto {
			keyEntry.Disable()
		} else {
			keyEntry.OnChanged = func(v string) {
				t.mu.Lock()
				if idx >= len(t.rows) {
					t.mu.Unlock()
					return
				}
				old := t.rows[idx].Key
				t.rows[idx].Key = v
				_, wasUsed := t.usedKeys[old]
				_, nowUsed := t.usedKeys[v]
				t.mu.Unlock()
				if wasUsed != nowUsed {
					t.rebuild()
					t.Refresh()
				}
				t.notify()
			}
		}
		keyObj = keyEntry
		if used && row.Key != "" {
			keyObj = withUsedAccent(keyEntry)
		}
	} else {
		keyEntry := NewEnvInput()
		keyEntry.SetPlaceHolder("Key")
		keyEntry.SetText(row.Key)
		if used && row.Key != "" {
			keyEntry.SetUsedAccent(true)
		}
		if t.opts.KeyReadOnly || row.Auto {
			keyEntry.Disable()
		} else {
			keyEntry.OnChanged(func(v string) {
				t.mu.Lock()
				if idx >= len(t.rows) {
					t.mu.Unlock()
					return
				}
				prev := t.rows[idx].Key
				prevWarn := t.rows[idx].Warn
				prevSecret := t.rows[idx].Secret
				t.rows[idx].Key = v
				t.rows[idx].Secret = IsSecretHeaderKey(v)
				t.rows[idx].Warn = !t.rows[idx].Auto && KeyConflicts(v, t.conflictKeys)
				flip := t.rows[idx].Secret != prevSecret || prevWarn != t.rows[idx].Warn ||
					IsSecretHeaderKey(prev) != IsSecretHeaderKey(v)
				t.mu.Unlock()
				if flip {
					t.rebuild()
					t.Refresh()
				}
				t.notify()
			})
		}
		keyObj = keyEntry
	}

	var valObj fyne.CanvasObject
	rowType := constants.NormalizeFormDataType(row.Type)
	maskValue := row.Secret
	if !t.opts.ShowSecret {
		maskValue = maskValue || IsSecretHeaderKey(row.Key)
	}
	if t.opts.ShowType && rowType == constants.FormDataTypeFile {
		valObj = t.makeFileValue(idx, row)
	} else if maskValue || t.opts.ShowSecret {
		valEntry := NewEntry()
		valEntry.Password = maskValue
		valEntry.SetPlaceHolder("Value")
		valEntry.SetText(row.Value)
		if t.opts.ValueReadOnly || row.Auto {
			valEntry.Disable()
		} else {
			valEntry.OnChanged = func(v string) {
				t.mu.Lock()
				if idx < len(t.rows) {
					t.rows[idx].Value = v
				}
				t.mu.Unlock()
				t.notify()
			}
		}
		valObj = valEntry
	} else {
		valEntry := NewEnvInput()
		valEntry.SetPlaceHolder("Value")
		valEntry.SetText(row.Value)
		if t.opts.ValueReadOnly || row.Auto {
			valEntry.Disable()
		} else {
			valEntry.OnChanged(func(v string) {
				t.mu.Lock()
				if idx < len(t.rows) {
					t.rows[idx].Value = v
				}
				t.mu.Unlock()
				t.notify()
			})
		}
		valObj = valEntry
	}

	var center fyne.CanvasObject
	if t.opts.ShowType {
		typeSelect := widget.NewSelect([]string{constants.FormDataTypeText, constants.FormDataTypeFile}, nil)
		typeSelect.PlaceHolder = "text"
		typeSelect.SetSelected(rowType)
		typeSelect.OnChanged = func(s string) {
			next := constants.NormalizeFormDataType(s)
			t.mu.Lock()
			if idx >= len(t.rows) {
				t.mu.Unlock()
				return
			}
			prev := constants.NormalizeFormDataType(t.rows[idx].Type)
			if prev == next {
				t.mu.Unlock()
				return
			}
			t.rows[idx].Type = next
			t.rows[idx].Value = ""
			t.mu.Unlock()
			t.rebuild()
			t.Refresh()
			t.notify()
		}
		center = container.New(&formDataColsLayout{}, keyObj, typeSelect, valObj)
	} else {
		center = container.NewGridWithColumns(2, keyObj, valObj)
	}

	var leftParts []fyne.CanvasObject
	if t.opts.Reorderable {
		if row.Auto {
			leftParts = append(leftParts, NewMinSizeBox(fyne.NewSize(20, 1), widget.NewLabel("")))
		} else {
			leftParts = append(leftParts, newKVDragHandle(t, idx))
		}
	}
	if t.opts.ShowCheck {
		check := widget.NewCheck("", func(v bool) {
			if row.Auto {
				return
			}
			t.mu.Lock()
			if idx < len(t.rows) {
				t.rows[idx].Enabled = v
			}
			t.mu.Unlock()
			t.notify()
		})
		check.Checked = row.Enabled
		if row.Auto {
			check.Disable()
		}
		leftParts = append(leftParts, check)
	}

	var left fyne.CanvasObject
	if len(leftParts) == 1 {
		left = leftParts[0]
	} else if len(leftParts) > 1 {
		left = container.NewHBox(leftParts...)
	}

	var right fyne.CanvasObject
	if t.opts.ShowSecret || t.opts.ShowDelete {
		var parts []fyne.CanvasObject
		if row.Warn && !row.Auto {
			parts = append(parts, widget.NewIcon(theme.WarningIcon()))
		}
		if t.opts.ShowSecret && !row.Auto {
			icon := theme.VisibilityIcon()
			if row.Secret {
				icon = theme.VisibilityOffIcon()
			}
			secretBtn := widget.NewButtonWithIcon("", icon, func() {
				t.mu.Lock()
				if idx < len(t.rows) {
					t.rows[idx].Secret = !t.rows[idx].Secret
				}
				t.mu.Unlock()
				t.rebuild()
				t.Refresh()
				t.notify()
			})
			secretBtn.Importance = widget.LowImportance
			parts = append(parts, secretBtn)
		}
		if t.opts.ShowDelete {
			if row.Auto {
				parts = append(parts, widget.NewLabel(""))
			} else {
				parts = append(parts, widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
					t.deleteRow(idx)
				}))
			}
		}
		if len(parts) == 1 {
			right = parts[0]
		} else if len(parts) > 1 {
			right = container.NewHBox(parts...)
		}
	}

	var body fyne.CanvasObject
	if left == nil && right == nil {
		body = center
	} else {
		body = container.NewBorder(nil, nil, left, right, center)
	}

	if !t.opts.Reorderable || row.Auto {
		return body
	}

	dragRow := newKVDragRow(t, idx, body)
	t.rowWidgets = append(t.rowWidgets, dragRow)
	return dragRow
}

func (t *KVTable) beginDrag(idx int) {
	t.mu.Lock()
	if idx < 0 || idx >= len(t.rows) || t.rows[idx].Auto {
		t.mu.Unlock()
		return
	}
	label := t.rows[idx].Key
	if label == "" {
		label = t.rows[idx].Value
	}
	if label == "" {
		label = "row"
	}
	t.dragActive = true
	t.dragSourceIdx = idx
	t.dropInsertIdx = -1
	t.dropLineIdx = -1
	t.mu.Unlock()
	t.ghost.Show(t, label, theme.ListIcon())
	t.refreshDropIndicators()
}

func (t *KVTable) updateDrag(abs fyne.Position) {
	t.mu.Lock()
	active := t.dragActive
	source := t.dragSourceIdx
	t.mu.Unlock()
	if !active || source < 0 {
		return
	}
	t.ghost.Move(abs)

	insertIdx, lineIdx, before, ok := t.resolveDrop(source, abs)
	t.mu.Lock()
	same := t.dropInsertIdx == insertIdx && t.dropLineIdx == lineIdx && t.dropBefore == before
	if ok {
		t.dropInsertIdx = insertIdx
		t.dropLineIdx = lineIdx
		t.dropBefore = before
	} else {
		t.dropInsertIdx = -1
		t.dropLineIdx = -1
	}
	t.mu.Unlock()
	if !same {
		t.refreshDropIndicators()
	}
}

func (t *KVTable) resolveDrop(source int, abs fyne.Position) (insertIdx, lineIdx int, before, ok bool) {
	rows := t.rowWidgets
	if len(rows) == 0 {
		return -1, -1, false, false
	}
	drv := fyne.CurrentApp().Driver()
	insertIdx = len(rows)
	lineIdx = -1
	for i, row := range rows {
		if row == nil {
			continue
		}
		pos := drv.AbsolutePositionForObject(row)
		h := row.Size().Height
		if h < 1 {
			h = 28
		}
		if abs.Y < pos.Y+h/2 {
			insertIdx = i
			lineIdx = i
			before = true
			break
		}
		lineIdx = i
		before = false
		insertIdx = i + 1
	}

	adjusted := insertIdx
	if source >= 0 && source < insertIdx {
		adjusted--
	}
	if source >= 0 && adjusted == source {
		return -1, -1, false, false
	}
	return adjusted, lineIdx, before, true
}

func (t *KVTable) endDrag() {
	t.mu.Lock()
	active := t.dragActive
	from := t.dragSourceIdx
	to := t.dropInsertIdx
	t.dragActive = false
	t.dragSourceIdx = -1
	t.dropInsertIdx = -1
	t.dropLineIdx = -1
	t.mu.Unlock()

	t.ghost.Hide()
	t.refreshDropIndicators()

	if !active || from < 0 || to < 0 {
		return
	}
	t.moveRow(from, to)
}

func (t *KVTable) refreshDropIndicators() {
	t.mu.Lock()
	source := t.dragSourceIdx
	lineIdx := t.dropLineIdx
	before := t.dropBefore
	active := t.dragActive
	t.mu.Unlock()

	for i, row := range t.rowWidgets {
		if row == nil {
			continue
		}
		row.SetDraggingSource(active && i == source)
		ind := dropNone
		if active && lineIdx >= 0 && i == lineIdx {
			if before {
				ind = dropBeforeIndicator
			} else {
				ind = dropAfterIndicator
			}
		}
		row.SetDropIndicator(ind)
	}
}

func (t *KVTable) MinSize() fyne.Size {
	s := fyne.NewSize(120, 32)
	if t.root != nil {
		s = t.root.MinSize()
	}
	// VScroll sizes content to max(min, viewport); keep width from exploding.
	const maxW float32 = 200
	if s.Width > maxW {
		s.Width = maxW
	}
	return s
}

type kvDragRow struct {
	widget.BaseWidget
	table *KVTable
	idx   int

	body       fyne.CanvasObject
	topLine    *canvas.Rectangle
	bottomLine *canvas.Rectangle
	root       *fyne.Container
}

func newKVDragRow(table *KVTable, idx int, body fyne.CanvasObject) *kvDragRow {
	topLine := canvas.NewRectangle(color.Transparent)
	topLine.SetMinSize(fyne.NewSize(1, 2))
	topLine.Hide()
	bottomLine := canvas.NewRectangle(color.Transparent)
	bottomLine.SetMinSize(fyne.NewSize(1, 2))
	bottomLine.Hide()
	root := container.NewBorder(topLine, bottomLine, nil, nil, body)
	r := &kvDragRow{
		table:      table,
		idx:        idx,
		body:       body,
		topLine:    topLine,
		bottomLine: bottomLine,
		root:       root,
	}
	r.ExtendBaseWidget(r)
	return r
}

func (r *kvDragRow) SetDraggingSource(on bool) {
	_ = on
}

func (r *kvDragRow) SetDropIndicator(where dropIndicator) {
	r.topLine.Hide()
	r.bottomLine.Hide()
	switch where {
	case dropBeforeIndicator:
		r.topLine.FillColor = ColorInsert
		r.topLine.Show()
	case dropAfterIndicator:
		r.bottomLine.FillColor = ColorInsert
		r.bottomLine.Show()
	}
	r.topLine.Refresh()
	r.bottomLine.Refresh()
}

func (r *kvDragRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(r.root)
}

func (r *kvDragRow) MinSize() fyne.Size {
	return r.root.MinSize()
}

// kvDragHandle — захват только за ручку, чтобы Entry не перехватывал drag.
type kvDragHandle struct {
	widget.BaseWidget
	table *KVTable
	idx   int
	hold  HoldDrag
	icon  *widget.Icon
}

func newKVDragHandle(table *KVTable, idx int) *kvDragHandle {
	icon := widget.NewIcon(theme.MenuIcon())
	h := &kvDragHandle{table: table, idx: idx, icon: icon}
	h.ExtendBaseWidget(h)
	return h
}

func (h *kvDragHandle) MouseDown(e *desktop.MouseEvent) {
	if e.Button != desktop.MouseButtonPrimary || h.table == nil {
		return
	}
	h.hold.MouseDown()
}

func (h *kvDragHandle) MouseUp(*desktop.MouseEvent) {
	h.hold.MouseUp()
}

func (h *kvDragHandle) Dragged(e *fyne.DragEvent) {
	if h.table == nil {
		return
	}
	started, active := h.hold.Dragged(e.Dragged.DX, e.Dragged.DY)
	if started {
		h.table.beginDrag(h.idx)
	}
	if active {
		h.table.updateDrag(e.AbsolutePosition)
	}
}

func (h *kvDragHandle) DragEnd() {
	if h.hold.DragEnd() && h.table != nil {
		h.table.endDrag()
	}
}

func (h *kvDragHandle) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(h.icon)
}

func (h *kvDragHandle) MinSize() fyne.Size {
	return fyne.NewSize(20, 20)
}

func (h *kvDragHandle) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

var _ fyne.Draggable = (*kvDragHandle)(nil)
var _ desktop.Mouseable = (*kvDragHandle)(nil)
var _ desktop.Cursorable = (*kvDragHandle)(nil)

// formDataColsLayout — Key | narrow Type | Value (Type ~ width of "text"/"file").
type formDataColsLayout struct{}

const formDataTypeColWidth float32 = 72

func (formDataColsLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 3 {
		return
	}
	pad := theme.Padding()
	typeW := formDataTypeColWidth
	rem := size.Width - typeW - 2*pad
	if rem < 0 {
		rem = 0
	}
	keyW := rem / 2
	valW := rem - keyW
	objects[0].Resize(fyne.NewSize(keyW, size.Height))
	objects[0].Move(fyne.NewPos(0, 0))
	objects[1].Resize(fyne.NewSize(typeW, size.Height))
	objects[1].Move(fyne.NewPos(keyW+pad, 0))
	objects[2].Resize(fyne.NewSize(valW, size.Height))
	objects[2].Move(fyne.NewPos(keyW+pad+typeW+pad, 0))
}

func (formDataColsLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	pad := theme.Padding()
	minH := float32(0)
	minKey, minVal := float32(40), float32(40)
	if len(objects) > 0 && objects[0] != nil {
		s := objects[0].MinSize()
		minKey = s.Width
		if s.Height > minH {
			minH = s.Height
		}
	}
	if len(objects) > 1 && objects[1] != nil {
		s := objects[1].MinSize()
		if s.Height > minH {
			minH = s.Height
		}
	}
	if len(objects) > 2 && objects[2] != nil {
		s := objects[2].MinSize()
		minVal = s.Width
		if s.Height > minH {
			minH = s.Height
		}
	}
	return fyne.NewSize(minKey+formDataTypeColWidth+minVal+2*pad, minH)
}

func kvHeader(addBtn fyne.CanvasObject, opts KVTableOptions) fyne.CanvasObject {
	keyLbl := widget.NewLabelWithStyle("Key", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	valueLbl := widget.NewLabelWithStyle("Value", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	var center fyne.CanvasObject
	if opts.ShowType {
		typeLbl := widget.NewLabelWithStyle("Type", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		center = container.New(&formDataColsLayout{}, keyLbl, typeLbl, valueLbl)
	} else {
		center = container.NewGridWithColumns(2, keyLbl, valueLbl)
	}

	var leftParts []fyne.CanvasObject
	if opts.Reorderable {
		leftParts = append(leftParts, NewMinSizeBox(fyne.NewSize(20, 1), widget.NewLabel("")))
	}
	if opts.ShowCheck {
		leftParts = append(leftParts, widget.NewLabelWithStyle("✓", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))
	}

	var left fyne.CanvasObject
	if len(leftParts) == 1 {
		left = leftParts[0]
	} else if len(leftParts) > 1 {
		left = container.NewHBox(leftParts...)
	}

	var right fyne.CanvasObject = widget.NewLabel("")
	if opts.ShowAdd {
		right = addBtn
	} else if opts.ShowDelete {
		right = widget.NewLabel("")
	}

	if left == nil {
		return container.NewBorder(nil, nil, nil, right, center)
	}
	return container.NewBorder(nil, nil, left, right, center)
}

func (t *KVTable) makeFileValue(idx int, row KVRow) fyne.CanvasObject {
	name := filepath.Base(row.Value)
	if row.Value == "" {
		name = ""
	}
	label := widget.NewLabel(name)
	label.Truncation = fyne.TextTruncateEllipsis
	if name == "" {
		label.SetText("No file selected")
		label.Importance = widget.LowImportance
	}

	btn := widget.NewButton("Select file", func() {
		win := t.opts.Window
		if win == nil {
			return
		}
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			uri := reader.URI()
			_ = reader.Close()
			path := ""
			if uri != nil {
				path = uri.Path()
				if path == "" {
					path = strings.TrimPrefix(uri.String(), "file://")
				}
			}
			t.mu.Lock()
			if idx < len(t.rows) {
				t.rows[idx].Value = path
				t.rows[idx].Type = constants.FormDataTypeFile
			}
			t.mu.Unlock()
			t.rebuild()
			t.Refresh()
			t.notify()
		}, win)
		fd.Show()
		fd.Resize(fyne.NewSize(800, 600))
	})
	btn.Importance = widget.LowImportance
	return container.NewBorder(nil, nil, nil, btn, label)
}

func withUsedAccent(content fyne.CanvasObject) fyne.CanvasObject {
	accent := canvas.NewRectangle(color.Transparent)
	accent.StrokeColor = EnvVarColor
	accent.StrokeWidth = theme.InputBorderSize() + 1
	accent.CornerRadius = theme.InputRadiusSize()
	return container.NewStack(accent, content)
}

// IsSecretHeaderKey reports headers whose values should be masked in the UI/logs.
func IsSecretHeaderKey(key string) bool {
	return strings.EqualFold(strings.TrimSpace(key), "Authorization")
}

// KeyConflicts reports whether key matches any of keys (case-insensitive, trimmed).
func KeyConflicts(key string, keys []string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	for _, k := range keys {
		if strings.EqualFold(key, strings.TrimSpace(k)) {
			return true
		}
	}
	return false
}

// KVRowsHaveKey reports whether any row has the given header key.
func KVRowsHaveKey(rows []KVRow, key string) bool {
	for _, r := range rows {
		if strings.EqualFold(strings.TrimSpace(r.Key), strings.TrimSpace(key)) {
			return true
		}
	}
	return false
}
