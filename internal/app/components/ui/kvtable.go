package ui

import (
	"image/color"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// KVRow — одна строка таблицы
type KVRow struct {
	Enabled bool
	Key     string
	Value   string
}

// KVTableOptions — поведение таблицы.
type KVTableOptions struct {
	ShowAdd     bool
	ShowCheck   bool
	ShowDelete  bool
	KeyReadOnly bool
	Reorderable bool
}

// KVTable — таблица key/value без внутреннего скролла (растёт по контенту).
type KVTable struct {
	widget.BaseWidget

	mu       sync.Mutex
	rows     []KVRow
	onChange func(rows []KVRow)
	opts     KVTableOptions

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
	t.rows = append(t.rows, KVRow{Enabled: true})
	t.mu.Unlock()
	t.rebuild()
	t.Refresh()
	t.notify()
}

func (t *KVTable) deleteRow(idx int) {
	t.mu.Lock()
	if idx < 0 || idx >= len(t.rows) {
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
	t.mu.Unlock()

	keyEntry := NewEnvInput()
	keyEntry.SetPlaceHolder("Key")
	keyEntry.SetText(row.Key)
	if t.opts.KeyReadOnly {
		keyEntry.Disable()
	} else {
		keyEntry.OnChanged(func(v string) {
			t.mu.Lock()
			if idx < len(t.rows) {
				t.rows[idx].Key = v
			}
			t.mu.Unlock()
			t.notify()
		})
	}

	valEntry := NewEnvInput()
	valEntry.SetPlaceHolder("Value")
	valEntry.SetText(row.Value)
	valEntry.OnChanged(func(v string) {
		t.mu.Lock()
		if idx < len(t.rows) {
			t.rows[idx].Value = v
		}
		t.mu.Unlock()
		t.notify()
	})

	center := container.NewGridWithColumns(2, keyEntry, valEntry)

	var leftParts []fyne.CanvasObject
	if t.opts.Reorderable {
		leftParts = append(leftParts, newKVDragHandle(t, idx))
	}
	if t.opts.ShowCheck {
		check := widget.NewCheck("", func(v bool) {
			t.mu.Lock()
			if idx < len(t.rows) {
				t.rows[idx].Enabled = v
			}
			t.mu.Unlock()
			t.notify()
		})
		check.Checked = row.Enabled
		leftParts = append(leftParts, check)
	}

	var left fyne.CanvasObject
	if len(leftParts) == 1 {
		left = leftParts[0]
	} else if len(leftParts) > 1 {
		left = container.NewHBox(leftParts...)
	}

	var right fyne.CanvasObject
	if t.opts.ShowDelete {
		right = widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
			t.deleteRow(idx)
		})
	}

	var body fyne.CanvasObject
	if left == nil && right == nil {
		body = center
	} else {
		body = container.NewBorder(nil, nil, left, right, center)
	}

	if !t.opts.Reorderable {
		return body
	}

	dragRow := newKVDragRow(t, idx, body)
	t.rowWidgets = append(t.rowWidgets, dragRow)
	return dragRow
}

func (t *KVTable) beginDrag(idx int) {
	t.mu.Lock()
	if idx < 0 || idx >= len(t.rows) {
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
	if t.root != nil {
		return t.root.MinSize()
	}
	return fyne.NewSize(120, 32)
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

func kvHeader(addBtn fyne.CanvasObject, opts KVTableOptions) fyne.CanvasObject {
	keyLbl := widget.NewLabelWithStyle("Key", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	valueLbl := widget.NewLabelWithStyle("Value", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	center := container.NewGridWithColumns(2, keyLbl, valueLbl)

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
