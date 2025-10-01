package rest

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

func NewRequestParams(onUpdate func(key string, value string)) fyne.CanvasObject {
	headers := []string{"ID", "Name", "Status"}
	t := widget.NewTableWithHeaders(
		func() (int, int) {
			return 1, len(headers) // +1 строка для заголовков
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Cell 000, 000")
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
		})

	t.ShowHeaderColumn = false
	t.UpdateHeader = func(id widget.TableCellID, o fyne.CanvasObject) {
		b := o.(*widget.Label)
		switch id.Col {
		case 0:
			b.Text = headers[0]
		case 1:
			b.Text = headers[1]
		case 2:
			b.Text = headers[2]

		}
		b.Refresh()
	}

	return t
}
