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

// kvRowWidget — именованная структура ячейки, без индексов
type kvRowWidget struct {
	check    *widget.Check
	keyEntry *widget.Entry
	valEntry *widget.Entry
	delBtn   *widget.Button
	root     fyne.CanvasObject
}

func newKVRowWidget() *kvRowWidget {
	r := &kvRowWidget{}
	r.check = widget.NewCheck("", nil)
	r.keyEntry = widget.NewEntry()
	r.keyEntry.PlaceHolder = "Key"
	r.valEntry = widget.NewEntry()
	r.valEntry.PlaceHolder = "Value"
	r.delBtn = widget.NewButtonWithIcon("", theme.DeleteIcon(), nil)

	r.root = container.NewBorder(
		nil, nil,
		r.check,  // left
		r.delBtn, // right
		container.NewGridWithColumns(2, r.keyEntry, r.valEntry),
	)
	return r
}

// KVTable — таблица key/value с inline-редактированием и чекбоксом
type KVTable struct {
	widget.BaseWidget

	rows       []KVRow
	onChange   func(rows []KVRow)
	showAddRow bool

	list *widget.List
}

func NewKVTable(initial []KVRow, onChange func(rows []KVRow)) *KVTable {
	return newKVTable(initial, onChange, true)
}

// NewKVTableReadOnly — таблица без кнопки добавления строк (только просмотр/редактирование).
func NewKVTableReadOnly(initial []KVRow, onChange func(rows []KVRow)) *KVTable {
	return newKVTable(initial, onChange, false)
}

func newKVTable(initial []KVRow, onChange func(rows []KVRow), showAddRow bool) *KVTable {
	if initial == nil {
		initial = []KVRow{}
	}
	t := &KVTable{
		rows:       initial,
		onChange:   onChange,
		showAddRow: showAddRow,
	}
	t.ExtendBaseWidget(t)
	t.buildList()
	return t
}

func (t *KVTable) SetRows(rows []KVRow) {
	if rows == nil {
		rows = []KVRow{}
	}
	t.rows = rows
	t.list.Refresh()
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
	t.list.Refresh()
	t.notify()
}

func (t *KVTable) deleteRow(idx int) {
	if idx < 0 || idx >= len(t.rows) {
		return
	}
	t.rows = append(t.rows[:idx], t.rows[idx+1:]...)
	t.list.Refresh()
	t.notify()
}

func (t *KVTable) buildList() {
	t.list = widget.NewList(
		func() int { return len(t.rows) },

		// createItem — создаём виджет и прячем указатель в Tag (нет Tag в fyne, используем замыкание)
		// Возвращаем root, но сам kvRowWidget живёт снаружи через map
		func() fyne.CanvasObject {
			return newKVRowWidget().root
		},

		// updateItem — находим виджеты через обход дерева явно
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(t.rows) {
				return
			}
			idx := id

			// obj — это container.NewBorder(...), его структура:
			// Border.Objects: [center, left, right] = [GridWithColumns, check, delBtn]
			// НО fyne.Border хранит Objects в порядке: все добавленные + позиционные.
			// Безопаснее пересоздать виджет из root-контейнера по типам через хелпер.
			rw := extractKVRowWidget(obj)
			if rw == nil {
				return
			}

			// --- чекбокс ---
			rw.check.OnChanged = nil
			rw.check.Checked = t.rows[idx].Enabled
			rw.check.Refresh()
			rw.check.OnChanged = func(v bool) {
				if idx < len(t.rows) {
					t.rows[idx].Enabled = v
					t.notify()
				}
			}

			// --- key ---
			rw.keyEntry.OnChanged = nil
			rw.keyEntry.SetText(t.rows[idx].Key)
			rw.keyEntry.OnChanged = func(v string) {
				if idx < len(t.rows) {
					t.rows[idx].Key = v
					t.notify()
				}
			}

			// --- value ---
			rw.valEntry.OnChanged = nil
			rw.valEntry.SetText(t.rows[idx].Value)
			rw.valEntry.OnChanged = func(v string) {
				if idx < len(t.rows) {
					t.rows[idx].Value = v
					t.notify()
				}
			}

			// --- delete ---
			rw.delBtn.OnTapped = func() {
				t.deleteRow(idx)
			}
		},
	)
}

// extractKVRowWidget восстанавливает указатели на виджеты из дерева контейнера
// по типам, без хрупких индексов.
//
// Структура дерева (создана в newKVRowWidget):
//
//	Border
//	  left:   *widget.Check
//	  right:  *widget.Button
//	  center: *fyne.Container (GridWithColumns)
//	            [0] *widget.Entry  (key)
//	            [1] *widget.Entry  (value)
func extractKVRowWidget(obj fyne.CanvasObject) *kvRowWidget {
	root, ok := obj.(*fyne.Container)
	if !ok {
		return nil
	}

	rw := &kvRowWidget{}

	for _, o := range root.Objects {
		switch v := o.(type) {
		case *widget.Check:
			rw.check = v
		case *widget.Button:
			rw.delBtn = v
		case *fyne.Container:
			// GridWithColumns с двумя Entry
			entries := []*widget.Entry{}
			for _, child := range v.Objects {
				if e, ok := child.(*widget.Entry); ok {
					entries = append(entries, e)
				}
			}
			if len(entries) >= 2 {
				rw.keyEntry = entries[0]
				rw.valEntry = entries[1]
			}
		}
	}

	if rw.check == nil || rw.keyEntry == nil || rw.valEntry == nil || rw.delBtn == nil {
		return nil
	}
	return rw
}

func (t *KVTable) CreateRenderer() fyne.WidgetRenderer {
	var addAction fyne.CanvasObject = widget.NewLabel("")
	if t.showAddRow {
		addBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
			t.addRow()
		})
		addBtn.Importance = widget.LowImportance
		addAction = addBtn
	}

	content := container.NewBorder(
		kvHeader(addAction),
		nil, nil, nil,
		t.list,
	)
	return widget.NewSimpleRenderer(content)
}

func (t *KVTable) MinSize() fyne.Size {
	return fyne.NewSize(400, 200)
}

// kvHeader — шапка таблицы с кнопкой + справа
func kvHeader(addBtn fyne.CanvasObject) fyne.CanvasObject {
	enabledLbl := widget.NewLabelWithStyle("✓", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	keyLbl := widget.NewLabelWithStyle("Key", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	valueLbl := widget.NewLabelWithStyle("Value", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	return container.NewBorder(nil, nil,
		enabledLbl,
		addBtn,
		container.NewGridWithColumns(2, keyLbl, valueLbl),
	)
}
