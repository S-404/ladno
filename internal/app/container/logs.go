package container

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/logs"
	"github.com/s-404/ladno/internal/app/entity/shared"
	"github.com/s-404/ladno/internal/app/store"
)

func LogsContainer(app *shared.App) fyne.CanvasObject {
	logStore := app.Store.Log

	listBox := container.NewVBox()
	scroll := container.NewVScroll(listBox)

	var rows []*logs.ExpandableEntry

	applyBatch := func(batch store.LogBatch) {
		if batch.Cleared {
			rows = nil
			listBox.Objects = nil
			listBox.Refresh()
			return
		}
		if batch.Reset != nil {
			rows = rows[:0]
			objs := make([]fyne.CanvasObject, 0, len(batch.Reset))
			for _, e := range batch.Reset {
				if e == nil {
					continue
				}
				row := logs.NewExpandableEntry(e, false)
				rows = append(rows, row)
				objs = append(objs, row)
			}
			listBox.Objects = objs
			listBox.Refresh()
			if len(objs) > 0 {
				scroll.ScrollToBottom()
			}
			return
		}

		for _, e := range batch.Appended {
			if e == nil {
				continue
			}
			row := logs.NewExpandableEntry(e, false)
			rows = append(rows, row)
			listBox.Objects = append(listBox.Objects, row)
		}
		if batch.DropOldest > 0 {
			if batch.DropOldest >= len(rows) {
				rows = nil
				listBox.Objects = nil
			} else {
				rows = rows[batch.DropOldest:]
				listBox.Objects = listBox.Objects[batch.DropOldest:]
			}
		}
		listBox.Refresh()
		if len(rows) > 0 {
			scroll.ScrollToBottom()
		}
	}

	go func() {
		for batch := range logStore.Updates() {
			b := batch
			fyne.Do(func() { applyBatch(b) })
		}
	}()

	clearBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		logStore.Clear()
	})
	clearBtn.Importance = widget.LowImportance

	header := container.NewBorder(
		nil, nil,
		widget.NewLabelWithStyle("Logs", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		clearBtn,
		nil,
	)

	return container.NewBorder(header, nil, nil, nil, scroll)
}
