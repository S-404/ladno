package logs

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/entity"
)

// ExpandableEntry — раскрываемая строка лога.
type ExpandableEntry struct {
	widget.BaseWidget

	entry    *entity.LogEntry
	expanded bool
	root     *fyne.Container
}

func NewExpandableEntry(entry *entity.LogEntry, expanded bool) *ExpandableEntry {
	e := &ExpandableEntry{
		entry:    entry,
		expanded: expanded,
	}
	e.ExtendBaseWidget(e)
	e.root = container.NewBorder(nil, nil, nil, nil, nil)
	e.rebuild()
	return e
}

func (e *ExpandableEntry) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(e.root)
}

func (e *ExpandableEntry) SetExpanded(v bool) {
	if e.expanded == v {
		return
	}
	e.expanded = v
	e.rebuild()
	e.Refresh()
}

func (e *ExpandableEntry) IsExpanded() bool {
	return e.expanded
}

func (e *ExpandableEntry) EntryID() string {
	if e.entry == nil {
		return ""
	}
	return e.entry.Id
}

func (e *ExpandableEntry) rebuild() {
	toggleIcon := theme.NavigateNextIcon()
	if e.expanded {
		toggleIcon = theme.MoveDownIcon()
	}
	toggle := widget.NewButtonWithIcon("", toggleIcon, func() {
		e.expanded = !e.expanded
		e.rebuild()
		e.Refresh()
	})
	toggle.Importance = widget.LowImportance

	badgeText := FormatStatusLabel(e.entry.StatusCode, e.entry.IsError)
	badge := canvas.NewText(badgeText, StatusColor(e.entry.StatusCode, e.entry.IsError))
	badge.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	badge.TextSize = theme.TextSize()

	timeTxt := canvas.NewText(e.entry.Time.Format("15:04:05.000"), colorMuted)
	timeTxt.TextStyle = fyne.TextStyle{Monospace: true}
	timeTxt.TextSize = theme.TextSize()

	kindTxt := canvas.NewText("["+e.entry.Kind+"]", colorMuted)
	kindTxt.TextStyle = fyne.TextStyle{Monospace: true}
	kindTxt.TextSize = theme.TextSize()

	msg := widget.NewLabel(e.entry.Message)
	msg.TextStyle = fyne.TextStyle{Monospace: true}
	msg.Truncation = fyne.TextTruncateEllipsis

	header := container.NewBorder(
		nil, nil,
		container.NewHBox(toggle, timeTxt, badge, kindTxt),
		nil,
		msg,
	)

	if e.expanded {
		// VBox: деталь занимает полную высоту контента, скролл — у общего списка логов
		e.root.Objects = []fyne.CanvasObject{
			container.NewVBox(header, NewDetailView(e.entry)),
		}
	} else {
		e.root.Objects = []fyne.CanvasObject{header}
	}
	e.root.Refresh()
}
