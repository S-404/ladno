package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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
}

// KVTable — таблица key/value без внутреннего скролла (растёт по контенту).
type KVTable struct {
	widget.BaseWidget

	rows     []KVRow
	onChange func(rows []KVRow)
	opts     KVTableOptions

	root *fyne.Container
}

func NewKVTable(initial []KVRow, onChange func(rows []KVRow)) *KVTable {
	return newKVTable(initial, onChange, KVTableOptions{
		ShowAdd:    true,
		ShowCheck:  true,
		ShowDelete: true,
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
		ShowCheck:  true,
		ShowDelete: true,
	})
}

func newKVTable(initial []KVRow, onChange func(rows []KVRow), opts KVTableOptions) *KVTable {
	if initial == nil {
		initial = []KVRow{}
	}
	t := &KVTable{
		rows:     initial,
		onChange: onChange,
		opts:     opts,
		root:     container.NewVBox(),
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
	t.rows = rows
	t.rebuild()
	t.Refresh()
}

func (t *KVTable) GetRows() []KVRow {
	return t.rows
}

func (t *KVTable) notify() {
	if t.onChange != nil {
		cp := make([]KVRow, len(t.rows))
		copy(cp, t.rows)
		t.onChange(cp)
	}
}

func (t *KVTable) addRow() {
	t.rows = append(t.rows, KVRow{Enabled: true})
	t.rebuild()
	t.Refresh()
	t.notify()
}

func (t *KVTable) deleteRow(idx int) {
	if idx < 0 || idx >= len(t.rows) {
		return
	}
	t.rows = append(t.rows[:idx], t.rows[idx+1:]...)
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

	rowsBox := container.NewVBox()
	for i := range t.rows {
		idx := i
		rowsBox.Add(t.makeRow(idx))
	}

	t.root.Objects = []fyne.CanvasObject{
		container.NewBorder(
			kvHeader(addAction, t.opts),
			nil, nil, nil,
			rowsBox,
		),
	}
	t.root.Refresh()
}

func (t *KVTable) makeRow(idx int) fyne.CanvasObject {
	row := t.rows[idx]

	keyEntry := NewEnvInput()
	keyEntry.SetPlaceHolder("Key")
	keyEntry.SetText(row.Key)
	if t.opts.KeyReadOnly {
		keyEntry.Disable()
	} else {
		keyEntry.OnChanged(func(v string) {
			if idx < len(t.rows) {
				t.rows[idx].Key = v
				t.notify()
			}
		})
	}

	valEntry := NewEnvInput()
	valEntry.SetPlaceHolder("Value")
	valEntry.SetText(row.Value)
	valEntry.OnChanged(func(v string) {
		if idx < len(t.rows) {
			t.rows[idx].Value = v
			t.notify()
		}
	})

	center := container.NewGridWithColumns(2, keyEntry, valEntry)

	var left fyne.CanvasObject
	if t.opts.ShowCheck {
		check := widget.NewCheck("", func(v bool) {
			if idx < len(t.rows) {
				t.rows[idx].Enabled = v
				t.notify()
			}
		})
		check.Checked = row.Enabled
		left = check
	}

	var right fyne.CanvasObject
	if t.opts.ShowDelete {
		right = widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
			t.deleteRow(idx)
		})
	}

	if left == nil && right == nil {
		return center
	}
	return container.NewBorder(nil, nil, left, right, center)
}

func (t *KVTable) MinSize() fyne.Size {
	if t.root != nil {
		return t.root.MinSize()
	}
	return fyne.NewSize(120, 32)
}

func kvHeader(addBtn fyne.CanvasObject, opts KVTableOptions) fyne.CanvasObject {
	keyLbl := widget.NewLabelWithStyle("Key", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	valueLbl := widget.NewLabelWithStyle("Value", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	center := container.NewGridWithColumns(2, keyLbl, valueLbl)

	var left fyne.CanvasObject
	if opts.ShowCheck {
		left = widget.NewLabelWithStyle("✓", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
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
