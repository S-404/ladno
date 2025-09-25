package rest

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

func NewRequestParams(onUpdate func(key string, value string)) fyne.CanvasObject {
	headers := []string{"ID", "Name", "Status"} // Ваши заголовки
	t := widget.NewTableWithHeaders(
		func() (int, int) {
			return 1 + 1, len(headers) // +1 строка для заголовков
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Cell 000, 000")
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)

			// Обработка строки заголовков (первая строка)
			if id.Row == 0 {
				label.SetText(headers[id.Col])
				label.TextStyle.Bold = true
				return
			}

			// Остальные строки (данные)
			label.SetText(fmt.Sprintf("Data %d-%d", id.Row, id.Col))
		})

	//t.SetColumnWidth(0, 300)
	//t.SetColumnWidth(1, 300)
	//t.SetColumnWidth(2, 300)

	t.ShowHeaderColumn = false
	t.UpdateHeader = func(id widget.TableCellID, o fyne.CanvasObject) {
		b := o.(*widget.Label)

		b.Refresh()
	}

	//return container.NewVBox(stretchedTable)
	return t
}
