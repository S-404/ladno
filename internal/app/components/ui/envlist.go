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

// EnvListItem — элемент списка окружений.
type EnvListItem struct {
	ID   string
	Name string
}

// EnvList — список env с drag-and-drop reorder (hold → ghost → insert line).
type EnvList struct {
	widget.BaseWidget

	mu         sync.Mutex
	items      []EnvListItem
	activeID   string
	selectedID string
	onSelect   func(id string)
	onReorder  func(id string, toIndex int)
	isDirty    func(id string) bool

	rowsByID map[string]*envListRow
	root     *fyne.Container
	rowsBox  *fyne.Container

	dragActive    bool
	dragSourceID  string
	dropInsertIdx int
	dropLineID    string
	dropBefore    bool
	ghost         DragGhost
}

func NewEnvList(onSelect func(id string), onReorder func(id string, toIndex int)) *EnvList {
	rowsBox := container.NewVBox()
	scroll := container.NewVScroll(rowsBox)
	l := &EnvList{
		onSelect:      onSelect,
		onReorder:     onReorder,
		rowsByID:      map[string]*envListRow{},
		root:          container.NewStack(scroll),
		rowsBox:       rowsBox,
		dropInsertIdx: -1,
	}
	l.ExtendBaseWidget(l)
	return l
}

func (l *EnvList) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(l.root)
}

func (l *EnvList) SetItems(items []EnvListItem, selectedID, activeID string) {
	l.mu.Lock()
	same := envItemsEqual(l.items, items)
	l.items = append([]EnvListItem(nil), items...)
	l.selectedID = selectedID
	l.activeID = activeID
	l.mu.Unlock()
	if same {
		l.refreshIndicators()
		return
	}
	l.rebuild()
}

func envItemsEqual(a, b []EnvListItem) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].ID != b[i].ID || a[i].Name != b[i].Name {
			return false
		}
	}
	return true
}

func (l *EnvList) SetSelected(id string) {
	l.mu.Lock()
	l.selectedID = id
	l.mu.Unlock()
	l.refreshIndicators()
}

func (l *EnvList) SetDirtyCheck(fn func(id string) bool) {
	l.mu.Lock()
	l.isDirty = fn
	l.mu.Unlock()
	l.refreshIndicators()
}

func (l *EnvList) rebuild() {
	l.mu.Lock()
	items := append([]EnvListItem(nil), l.items...)
	selected := l.selectedID
	active := l.activeID
	l.rowsByID = map[string]*envListRow{}
	l.mu.Unlock()

	l.mu.Lock()
	dirtyFn := l.isDirty
	l.mu.Unlock()
	objs := make([]fyne.CanvasObject, 0, len(items))
	for _, it := range items {
		row := newEnvListRow(l)
		dirty := dirtyFn != nil && dirtyFn(it.ID)
		row.SetItem(it.ID, it.Name, it.ID == selected, it.ID == active, dirty)
		l.mu.Lock()
		l.rowsByID[it.ID] = row
		l.mu.Unlock()
		objs = append(objs, row)
	}
	l.rowsBox.Objects = objs
	l.rowsBox.Refresh()
	l.root.Refresh()
	l.Refresh()
}

func (l *EnvList) refreshIndicators() {
	l.mu.Lock()
	selected := l.selectedID
	active := l.activeID
	source := l.dragSourceID
	lineID := l.dropLineID
	before := l.dropBefore
	rows := make(map[string]*envListRow, len(l.rowsByID))
	for id, r := range l.rowsByID {
		rows[id] = r
	}
	items := append([]EnvListItem(nil), l.items...)
	dirtyFn := l.isDirty
	l.mu.Unlock()

	for _, it := range items {
		row := rows[it.ID]
		if row == nil {
			continue
		}
		dirty := dirtyFn != nil && dirtyFn(it.ID)
		row.SetItem(it.ID, it.Name, it.ID == selected, it.ID == active, dirty)
		row.SetDraggingSource(it.ID == source && source != "")
		ind := dropNone
		if lineID != "" && it.ID == lineID {
			if before {
				ind = dropBeforeIndicator
			} else {
				ind = dropAfterIndicator
			}
		}
		row.SetDropIndicator(ind)
	}
}

type dropIndicator int

const (
	dropNone dropIndicator = iota
	dropBeforeIndicator
	dropAfterIndicator
)

func (l *EnvList) beginDrag(id string) {
	l.mu.Lock()
	var label string
	for _, it := range l.items {
		if it.ID == id {
			label = it.Name
			break
		}
	}
	if label == "" {
		l.mu.Unlock()
		return
	}
	l.dragActive = true
	l.dragSourceID = id
	l.dropInsertIdx = -1
	l.dropLineID = ""
	l.mu.Unlock()
	l.ghost.Show(l, label, theme.ListIcon())
	l.refreshIndicators()
}

func (l *EnvList) updateDrag(abs fyne.Position) {
	l.mu.Lock()
	active := l.dragActive
	source := l.dragSourceID
	l.mu.Unlock()
	if !active || source == "" {
		return
	}
	l.ghost.Move(abs)

	insertIdx, lineID, before, ok := l.resolveDrop(source, abs)
	l.mu.Lock()
	same := l.dropInsertIdx == insertIdx && l.dropLineID == lineID && l.dropBefore == before
	if ok {
		l.dropInsertIdx = insertIdx
		l.dropLineID = lineID
		l.dropBefore = before
	} else {
		l.dropInsertIdx = -1
		l.dropLineID = ""
	}
	l.mu.Unlock()
	if !same {
		l.refreshIndicators()
	}
}

func (l *EnvList) resolveDrop(source string, abs fyne.Position) (insertIdx int, lineID string, before, ok bool) {
	l.mu.Lock()
	items := append([]EnvListItem(nil), l.items...)
	rows := make(map[string]*envListRow, len(l.rowsByID))
	for id, r := range l.rowsByID {
		rows[id] = r
	}
	l.mu.Unlock()
	if len(items) == 0 {
		return -1, "", false, false
	}

	drv := fyne.CurrentApp().Driver()
	insertIdx = len(items)
	for i, it := range items {
		row := rows[it.ID]
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
			lineID = it.ID
			before = true
			break
		}
		lineID = it.ID
		before = false
		insertIdx = i + 1
	}

	srcIdx := -1
	for i, it := range items {
		if it.ID == source {
			srcIdx = i
			break
		}
	}
	adjusted := insertIdx
	if srcIdx >= 0 && srcIdx < insertIdx {
		adjusted--
	}
	if srcIdx >= 0 && adjusted == srcIdx {
		return -1, "", false, false
	}
	return adjusted, lineID, before, true
}

func (l *EnvList) endDrag() {
	l.mu.Lock()
	active := l.dragActive
	source := l.dragSourceID
	toIdx := l.dropInsertIdx
	l.dragActive = false
	l.dragSourceID = ""
	l.dropInsertIdx = -1
	l.dropLineID = ""
	cb := l.onReorder
	l.mu.Unlock()

	l.ghost.Hide()

	if !active || source == "" || toIdx < 0 {
		l.refreshIndicators()
		return
	}
	// Сразу обновляем UI: binding.UntypedList не шлёт list-listener при reorder без смены длины.
	l.reorderLocal(source, toIdx)
	if cb != nil {
		cb(source, toIdx)
	}
}

func (l *EnvList) reorderLocal(id string, toIndex int) {
	l.mu.Lock()
	from := -1
	for i, it := range l.items {
		if it.ID == id {
			from = i
			break
		}
	}
	if from < 0 || toIndex < 0 || toIndex >= len(l.items) || from == toIndex {
		l.mu.Unlock()
		l.refreshIndicators()
		return
	}
	items := append([]EnvListItem(nil), l.items...)
	item := items[from]
	items = append(items[:from], items[from+1:]...)
	items = append(items[:toIndex], append([]EnvListItem{item}, items[toIndex:]...)...)
	l.items = items
	l.mu.Unlock()
	l.rebuild()
}

func (l *EnvList) selectID(id string) {
	l.mu.Lock()
	l.selectedID = id
	cb := l.onSelect
	l.mu.Unlock()
	l.refreshIndicators()
	if cb != nil {
		cb(id)
	}
}

var colorEnvDirty = color.NRGBA{R: 0xFF, G: 0x98, B: 0x00, A: 0xFF}

type envListRow struct {
	widget.BaseWidget
	list *EnvList
	id   string
	hold HoldDrag

	label      *widget.Label
	dot        *canvas.Circle
	topLine    *canvas.Rectangle
	bottomLine *canvas.Rectangle
	root       *fyne.Container
}

func newEnvListRow(list *EnvList) *envListRow {
	label := widget.NewLabel("")
	label.Truncation = fyne.TextTruncateEllipsis
	dot := canvas.NewCircle(color.Transparent)
	dot.Hide()
	dotBox := NewMinSizeBox(fyne.NewSize(8, 8), dot)
	topLine := canvas.NewRectangle(color.Transparent)
	topLine.SetMinSize(fyne.NewSize(1, 2))
	topLine.Hide()
	bottomLine := canvas.NewRectangle(color.Transparent)
	bottomLine.SetMinSize(fyne.NewSize(1, 2))
	bottomLine.Hide()
	body := container.NewBorder(nil, nil, container.NewCenter(dotBox), nil, label)
	root := container.NewBorder(topLine, bottomLine, nil, nil, body)
	r := &envListRow{
		list:       list,
		label:      label,
		dot:        dot,
		topLine:    topLine,
		bottomLine: bottomLine,
		root:       root,
	}
	r.ExtendBaseWidget(r)
	return r
}

func (r *envListRow) SetItem(id, name string, selected, active, dirty bool) {
	r.id = id
	text := name
	if active {
		text = "● " + name
	}
	r.label.SetText(text)
	if selected {
		r.label.Importance = widget.HighImportance
	} else {
		r.label.Importance = widget.MediumImportance
	}
	r.label.Refresh()
	if dirty {
		r.dot.FillColor = colorEnvDirty
		r.dot.StrokeColor = colorEnvDirty
		r.dot.Show()
	} else {
		r.dot.Hide()
	}
	r.dot.Refresh()
}

func (r *envListRow) SetDraggingSource(on bool) {
	if on {
		r.label.Importance = widget.LowImportance
	}
	r.label.Refresh()
}

func (r *envListRow) SetDropIndicator(where dropIndicator) {
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

func (r *envListRow) Tapped(_ *fyne.PointEvent) {
	if r.hold.Dragging {
		return
	}
	if r.list != nil && r.id != "" {
		r.list.selectID(r.id)
	}
}

func (r *envListRow) MouseDown(e *desktop.MouseEvent) {
	if e.Button != desktop.MouseButtonPrimary || r.id == "" {
		return
	}
	r.hold.MouseDown()
}

func (r *envListRow) MouseUp(*desktop.MouseEvent) {
	r.hold.MouseUp()
}

func (r *envListRow) Dragged(e *fyne.DragEvent) {
	if r.list == nil || r.id == "" {
		return
	}
	started, active := r.hold.Dragged(e.Dragged.DX, e.Dragged.DY)
	if started {
		r.list.beginDrag(r.id)
	}
	if active {
		r.list.updateDrag(e.AbsolutePosition)
	}
}

func (r *envListRow) DragEnd() {
	if r.hold.DragEnd() && r.list != nil {
		r.list.endDrag()
	}
}

func (r *envListRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(r.root)
}

func (r *envListRow) MinSize() fyne.Size {
	return r.root.MinSize()
}

var _ fyne.Tappable = (*envListRow)(nil)
var _ fyne.Draggable = (*envListRow)(nil)
var _ desktop.Mouseable = (*envListRow)(nil)
