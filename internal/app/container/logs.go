package container

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/logs"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity/shared"
	"github.com/s-404/ladno/internal/app/store"
)

func LogsContainer(app *shared.App) fyne.CanvasObject {
	logStore := app.Store.Log

	listBox := container.NewVBox()
	scroll := container.NewVScroll(listBox)

	var rows []*logs.ExpandableEntry
	onBottom := true

	var stickBtn *widget.Button
	applyStickBtn := func() {
		if stickBtn == nil {
			return
		}
		if onBottom {
			stickBtn.Importance = widget.MediumImportance
		} else {
			stickBtn.Importance = widget.LowImportance
		}
		stickBtn.Refresh()
	}

	scroll.OnScrolled = func(fyne.Position) {
		now := scrollAtBottom(scroll)
		if now == onBottom {
			return
		}
		onBottom = now
		applyStickBtn()
	}

	applyBatch := func(batch store.LogBatch) {
		if batch.Cleared {
			rows = nil
			listBox.Objects = nil
			listBox.Refresh()
			onBottom = true
			applyStickBtn()
			return
		}

		droppedH := float32(0)
		if batch.DropOldest > 0 && batch.Reset == nil {
			droppedH = vboxPrefixHeight(listBox.Objects, batch.DropOldest)
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
			if onBottom && len(objs) > 0 {
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
		if len(rows) == 0 {
			return
		}
		if onBottom {
			scroll.ScrollToBottom()
			return
		}
		if droppedH > 0 {
			y := scroll.Offset.Y - droppedH
			if y < 0 {
				y = 0
			}
			scroll.ScrollToOffset(fyne.NewPos(scroll.Offset.X, y))
		}
	}

	go func() {
		for batch := range logStore.Updates() {
			b := batch
			fyne.Do(func() { applyBatch(b) })
		}
	}()

	stickBtn = widget.NewButtonWithIcon("", theme.MoveDownIcon(), func() {
		onBottom = !onBottom
		applyStickBtn()
		if onBottom {
			scroll.ScrollToBottom()
		}
	})
	stickBtn.Importance = widget.MediumImportance

	clearBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		logStore.Clear()
	})
	clearBtn.Importance = widget.LowImportance

	header := container.NewBorder(
		nil, nil,
		widget.NewLabelWithStyle("Logs", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(stickBtn, clearBtn),
		nil,
	)

	return ui.NewPanelBackground(container.NewBorder(header, nil, nil, nil, scroll))
}

func scrollAtBottom(s *container.Scroll) bool {
	if s == nil || s.Content == nil {
		return true
	}
	return offsetAtBottom(s.Offset.Y, s.Content.MinSize().Height, s.Size().Height)
}

func offsetAtBottom(offsetY, contentH, viewH float32) bool {
	const slop = float32(1)
	overflow := contentH - viewH
	if overflow <= slop {
		return true
	}
	return offsetY >= overflow-slop
}

func vboxPrefixHeight(objs []fyne.CanvasObject, n int) float32 {
	if n <= 0 || n > len(objs) {
		return 0
	}
	var h float32
	for _, o := range objs[:n] {
		if o == nil {
			continue
		}
		h += o.Size().Height
	}
	pad := theme.Padding()
	remaining := len(objs) - n
	if remaining > 0 {
		h += pad * float32(n)
	} else if n > 1 {
		h += pad * float32(n-1)
	}
	return h
}
