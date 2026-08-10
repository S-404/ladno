package natsui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/ui"
)

// MessagesView — блок сообщений по выбранному subject (Latest / All).
type MessagesView struct {
	root      fyne.CanvasObject
	body      *widget.Entry
	modeLabel *widget.Label
	showAll   bool
	onToggle  func(all bool)
	onCopy    func()
	onClear   func()
}

func NewMessagesView(onToggle func(all bool), onCopy, onClear func()) *MessagesView {
	v := &MessagesView{
		body:      widget.NewMultiLineEntry(),
		modeLabel: widget.NewLabel("Latest"),
		onToggle:  onToggle,
		onCopy:    onCopy,
		onClear:   onClear,
	}
	v.body.TextStyle = fyne.TextStyle{Monospace: true}
	v.body.Wrapping = fyne.TextWrapOff
	v.body.SetMinRowsVisible(4)
	v.body.SetPlaceHolder("Messages for selected subject")
	v.modeLabel.TextStyle = fyne.TextStyle{Italic: true}

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
	panel := container.NewBorder(header, nil, nil, nil, container.NewScroll(v.body))
	v.root = ui.NewPanelBackground(panel)
	v.body.Disable()
	return v
}

func (v *MessagesView) Object() fyne.CanvasObject {
	return v.root
}

func (v *MessagesView) ShowAll() bool {
	return v.showAll
}

func (v *MessagesView) SetText(text string) {
	v.body.SetText(text)
}

func (v *MessagesView) Text() string {
	return v.body.Text
}
