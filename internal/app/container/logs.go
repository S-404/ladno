package container

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/logs"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/shared"
)

func LogsContainer(app *shared.App) fyne.CanvasObject {
	logStore := app.Store.Log
	items := logStore.GetItems()

	listBox := container.NewVBox()
	scroll := container.NewVScroll(listBox)

	var rows []*logs.ExpandableEntry

	rebuild := func() {
		expanded := map[string]bool{}
		for _, row := range rows {
			if row != nil && row.IsExpanded() {
				expanded[row.EntryID()] = true
			}
		}

		all, _ := (*items).Get()
		rows = rows[:0]
		objs := make([]fyne.CanvasObject, 0, len(all))
		for _, raw := range all {
			entry, ok := raw.(*entity.LogEntry)
			if !ok || entry == nil {
				continue
			}
			row := logs.NewExpandableEntry(entry, expanded[entry.Id])
			rows = append(rows, row)
			objs = append(objs, row)
		}
		listBox.Objects = objs
		listBox.Refresh()
		if len(objs) > 0 {
			scroll.ScrollToBottom()
		}
	}

	(*items).AddListener(binding.NewDataListener(rebuild))

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
