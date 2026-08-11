package kafkaui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/ui"
)

// MessagesView — сообщения по топику с текстовым фильтром (Latest / All).
type MessagesView struct {
	root      fyne.CanvasObject
	body      *widget.Entry
	filter    *widget.Entry
	modeLabel *widget.Label
	showAll   bool
	onToggle  func(all bool)
	onFilter  func(q string)
	onCopy    func()
	onClear   func()
}

func NewMessagesView(onToggle func(all bool), onFilter func(q string), onCopy, onClear func()) *MessagesView {
	v := &MessagesView{
		body:      widget.NewMultiLineEntry(),
		filter:    widget.NewEntry(),
		modeLabel: widget.NewLabel("Latest"),
		onToggle:  onToggle,
		onFilter:  onFilter,
		onCopy:    onCopy,
		onClear:   onClear,
	}
	v.body.TextStyle = fyne.TextStyle{Monospace: true}
	v.body.Wrapping = fyne.TextWrapOff
	v.body.SetMinRowsVisible(4)
	v.body.SetPlaceHolder("Messages for selected topic")
	v.modeLabel.TextStyle = fyne.TextStyle{Italic: true}
	v.filter.SetPlaceHolder("Filter messages…")
	v.filter.OnChanged = func(q string) {
		if v.onFilter != nil {
			v.onFilter(q)
		}
	}

	var toggleBtn *widget.Button
	toggleBtn = widget.NewButtonWithIcon("", theme.ListIcon(), func() {
		v.showAll = !v.showAll
		if v.showAll {
			v.modeLabel.SetText("All")
			toggleBtn.SetIcon(theme.DocumentIcon())
		} else {
			v.modeLabel.SetText("Latest")
			toggleBtn.SetIcon(theme.ListIcon())
		}
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
		container.NewHBox(widget.NewLabelWithStyle("Messages", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), v.modeLabel),
		container.NewHBox(toggleBtn, copyBtn, clearBtn),
		nil,
	)
	panel := container.NewBorder(
		container.NewVBox(header, v.filter),
		nil, nil, nil,
		container.NewScroll(v.body),
	)
	v.root = ui.NewPanelBackground(panel)
	return v
}

func (v *MessagesView) Object() fyne.CanvasObject {
	return v.root
}

func (v *MessagesView) ShowAll() bool {
	return v.showAll
}

func (v *MessagesView) Filter() string {
	return v.filter.Text
}

func (v *MessagesView) SetText(text string) {
	v.body.SetText(text)
}

func (v *MessagesView) Text() string {
	return v.body.Text
}
