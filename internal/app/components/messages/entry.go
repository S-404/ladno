package messages

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var (
	colorMuted = color.NRGBA{R: 0xB0, G: 0xB8, B: 0xC4, A: 0xFF}
	colorIn    = color.NRGBA{R: 0x4F, G: 0xC3, B: 0xF7, A: 0xFF}
	colorOut   = color.NRGBA{R: 0xFF, G: 0xB7, B: 0x4D, A: 0xFF}
)

func dirColor(dir string) color.Color {
	if dir == DirIn {
		return colorIn
	}
	return colorOut
}

// ExpandableItem is a log-style row: header [time | direction] and expandable body.
type ExpandableItem struct {
	widget.BaseWidget

	item     Item
	expanded bool
	root     *fyne.Container
}

func NewExpandableItem(item Item, expanded bool) *ExpandableItem {
	e := &ExpandableItem{item: item, expanded: expanded}
	e.ExtendBaseWidget(e)
	e.root = container.NewBorder(nil, nil, nil, nil, nil)
	e.rebuild()
	return e
}

func (e *ExpandableItem) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(e.root)
}

func (e *ExpandableItem) MinSize() fyne.Size {
	s := e.BaseWidget.MinSize()
	s.Width = 1
	return s
}

func (e *ExpandableItem) rebuild() {
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

	timeTxt := canvas.NewText(FormatTime(e.item.Time), colorMuted)
	timeTxt.TextStyle = fyne.TextStyle{Monospace: true}
	timeTxt.TextSize = theme.TextSize()

	arrowTxt := canvas.NewText(DirArrow(e.item.Dir), dirColor(e.item.Dir))
	arrowTxt.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	arrowTxt.TextSize = theme.TextSize()

	sep := canvas.NewText("|", colorMuted)
	sep.TextStyle = fyne.TextStyle{Monospace: true}
	sep.TextSize = theme.TextSize()

	header := container.NewHBox(toggle, timeTxt, sep, arrowTxt)

	if e.expanded {
		body := widget.NewLabel(e.item.Body)
		body.Wrapping = fyne.TextWrapBreak
		body.TextStyle = fyne.TextStyle{Monospace: true}
		body.Selectable = true
		e.root.Objects = []fyne.CanvasObject{
			container.NewVBox(header, body),
		}
	} else {
		e.root.Objects = []fyne.CanvasObject{header}
	}
	e.root.Refresh()
}
