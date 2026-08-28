package messages

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/ui"
)

type View struct {
	root        fyne.CanvasObject
	body        *widget.Entry
	filter      *widget.Entry
	modeLabel   *widget.Label
	latestTime  *canvas.Text
	latestArrow *canvas.Text
	latestMeta  *fyne.Container
	listBox     *fyne.Container
	listScroll  *container.Scroll
	latestPane  fyne.CanvasObject
	listPane    fyne.CanvasObject
	showAll     bool
	items       []Item
	visible     []Item
	onToggle    func(all bool)
	onCopy      func()
	onClear     func()
}

func NewView(placeholder string, onToggle func(all bool), onCopy, onClear func()) *View {
	v := &View{
		body:        widget.NewMultiLineEntry(),
		filter:      widget.NewEntry(),
		modeLabel:   widget.NewLabel("Latest"),
		latestTime:  canvas.NewText("", colorMuted),
		latestArrow: canvas.NewText("", colorMuted),
		listBox:     container.NewVBox(),
		onToggle:    onToggle,
		onCopy:      onCopy,
		onClear:     onClear,
	}
	v.body.TextStyle = fyne.TextStyle{Monospace: true}
	v.body.Wrapping = fyne.TextWrapOff
	v.body.SetMinRowsVisible(4)
	v.body.SetPlaceHolder(placeholder)
	v.modeLabel.TextStyle = fyne.TextStyle{Italic: true}
	v.latestTime.TextStyle = fyne.TextStyle{Monospace: true}
	v.latestTime.TextSize = theme.TextSize()
	v.latestArrow.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	v.latestArrow.TextSize = theme.TextSize()
	latestSep := canvas.NewText("|", colorMuted)
	latestSep.TextStyle = fyne.TextStyle{Monospace: true}
	latestSep.TextSize = theme.TextSize()
	v.filter.SetPlaceHolder("Filter messages…")
	v.filter.OnChanged = func(string) { v.apply() }
	v.filter.Hide()

	v.latestMeta = container.NewHBox(v.latestTime, latestSep, v.latestArrow)
	v.latestMeta.Hide()

	var toggleBtn *widget.Button
	toggleBtn = widget.NewButtonWithIcon("", theme.ListIcon(), func() {
		v.showAll = !v.showAll
		if v.showAll {
			v.modeLabel.SetText("All")
			toggleBtn.SetIcon(theme.DocumentIcon())
			v.filter.Show()
		} else {
			v.modeLabel.SetText("Latest")
			toggleBtn.SetIcon(theme.ListIcon())
			v.filter.Hide()
		}
		v.apply()
		if v.onToggle != nil {
			v.onToggle(v.showAll)
		}
	})
	toggleBtn.Importance = widget.LowImportance

	copyBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		if v.onCopy != nil {
			v.onCopy()
		}
	})
	copyBtn.Importance = widget.LowImportance

	clearBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		if v.onClear != nil {
			v.onClear()
		}
	})
	clearBtn.Importance = widget.LowImportance

	header := container.NewBorder(
		nil, nil,
		container.NewHBox(
			widget.NewLabelWithStyle("Messages", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			v.modeLabel,
			v.latestMeta,
		),
		container.NewHBox(toggleBtn, copyBtn, clearBtn),
		nil,
	)

	v.latestPane = container.NewScroll(v.body)
	v.listScroll = container.NewVScroll(v.listBox)
	v.listPane = v.listScroll
	v.listPane.Hide()

	bodyStack := container.NewStack(v.latestPane, v.listPane)
	panel := container.NewBorder(
		container.NewVBox(header, v.filter),
		nil, nil, nil,
		bodyStack,
	)
	v.root = ui.NewPanelBackground(panel)
	return v
}

func (v *View) Object() fyne.CanvasObject {
	return v.root
}

func (v *View) ShowAll() bool {
	return v.showAll
}

func (v *View) Filter() string {
	return v.filter.Text
}

func (v *View) SetItems(items []Item) {
	v.items = items
	v.apply()
}

func (v *View) Text() string {
	if len(v.visible) == 0 {
		return ""
	}
	parts := make([]string, 0, len(v.visible))
	for _, it := range v.visible {
		parts = append(parts, FormatItem(it))
	}
	return strings.Join(parts, "\n\n")
}

func (v *View) apply() {
	v.visible = VisibleItems(v.items, v.filter.Text, v.showAll)
	if v.showAll {
		v.latestPane.Hide()
		v.listPane.Show()
		v.latestMeta.Hide()
		objs := make([]fyne.CanvasObject, 0, len(v.visible))
		for _, it := range v.visible {
			objs = append(objs, NewExpandableItem(it, true))
		}
		v.listBox.Objects = objs
		v.listBox.Refresh()
		if len(objs) > 0 && v.listScroll != nil {
			v.listScroll.ScrollToBottom()
		}
		v.refreshRoot()
		return
	}

	v.listPane.Hide()
	v.latestPane.Show()
	if len(v.visible) == 0 {
		v.body.SetText("")
		v.latestMeta.Hide()
		v.refreshRoot()
		return
	}
	it := v.visible[0]
	v.body.SetText(it.Body)
	v.latestTime.Text = FormatTime(it.Time)
	v.latestTime.Refresh()
	v.latestArrow.Text = DirArrow(it.Dir)
	v.latestArrow.Color = dirColor(it.Dir)
	v.latestArrow.Refresh()
	v.latestMeta.Show()
	v.latestMeta.Refresh()
	v.refreshRoot()
}

func (v *View) refreshRoot() {
	if v.root != nil {
		v.root.Refresh()
	}
}
