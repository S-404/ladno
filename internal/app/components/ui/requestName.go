package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// EditableName — имя как лейбл; по клику становится обычным инпутом.
// Клик в любом месте кроме инпута завершает редактирование.
type EditableName struct {
	Object    fyne.CanvasObject // поле без подписи "Name"
	Set       func(name string)
	Get       func() string
	FocusEdit func() // открыть инпут и выделить текст (для новой записи)
}

func NewEditableName(placeholder string, onChange func(name string)) *EditableName {
	var name string
	var editOriginal string
	var editing bool
	var applying bool
	var overlay *nameEditOverlay

	label := &tappableNameLabel{}
	label.ExtendBaseWidget(label)
	label.Truncation = fyne.TextTruncateEllipsis

	entry := &nameFocusEntry{}
	entry.ExtendBaseWidget(entry)
	entry.SetPlaceHolder(placeholder)

	field := container.NewStack(label)

	syncLabel := func() {
		if name == "" {
			label.SetText(placeholder)
			label.Importance = widget.LowImportance
		} else {
			label.SetText(name)
			label.Importance = widget.MediumImportance
		}
		label.Refresh()
	}
	syncLabel()

	notify := func() {
		if applying || onChange == nil {
			return
		}
		onChange(name)
	}

	removeOverlay := func() {
		if overlay == nil {
			return
		}
		cnv := fyne.CurrentApp().Driver().CanvasForObject(overlay)
		if cnv == nil {
			cnv = fyne.CurrentApp().Driver().CanvasForObject(field)
		}
		if cnv != nil {
			cnv.Overlays().Remove(overlay)
		}
		overlay = nil
	}

	finishEdit := func() {
		if !editing {
			return
		}
		editing = false
		name = entry.Text
		removeOverlay()
		if cnv := fyne.CurrentApp().Driver().CanvasForObject(field); cnv != nil {
			cnv.Unfocus()
		}
		syncLabel()
		if name != editOriginal {
			notify()
		}
	}

	startEdit := func() {
		if editing {
			return
		}
		cnv := fyne.CurrentApp().Driver().CanvasForObject(field)
		if cnv == nil {
			return
		}

		editing = true
		editOriginal = name
		applying = true
		entry.SetText(name)
		applying = false

		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(field)
		size := field.Size()
		if size.Width < 120 {
			size.Width = 120
		}
		if size.Height < entry.MinSize().Height {
			size.Height = entry.MinSize().Height
		}

		overlay = newNameEditOverlay(entry, pos, size, finishEdit)
		overlay.Resize(cnv.Size())
		cnv.Overlays().Add(overlay)
		cnv.Focus(entry)
	}

	label.onTap = startEdit
	entry.onFocusLost = finishEdit
	entry.OnSubmitted = func(string) {
		finishEdit()
	}
	entry.OnChanged = func(v string) {
		if applying || !editing {
			return
		}
		name = v
		if v != editOriginal {
			notify()
		}
	}

	return &EditableName{
		Object: field,
		Set: func(n string) {
			applying = true
			name = n
			editOriginal = n
			entry.SetText(n)
			if editing {
				finishEdit()
			} else {
				syncLabel()
			}
			applying = false
		},
		Get: func() string {
			if editing {
				return entry.Text
			}
			return name
		},
		FocusEdit: func() {
			startEdit()
			if !editing {
				fyne.Do(func() {
					startEdit()
					if editing {
						entry.TypedShortcut(&fyne.ShortcutSelectAll{})
					}
				})
				return
			}
			entry.TypedShortcut(&fyne.ShortcutSelectAll{})
		},
	}
}

// nameEditOverlay covers the canvas; taps outside the entry end editing.
type nameEditOverlay struct {
	widget.BaseWidget

	entry     *nameFocusEntry
	entryPos  fyne.Position
	entrySize fyne.Size
	onDismiss func()
	bg        *canvas.Rectangle
}

func newNameEditOverlay(entry *nameFocusEntry, pos fyne.Position, size fyne.Size, onDismiss func()) *nameEditOverlay {
	o := &nameEditOverlay{
		entry:     entry,
		entryPos:  pos,
		entrySize: size,
		onDismiss: onDismiss,
		bg:        canvas.NewRectangle(color.Transparent),
	}
	o.ExtendBaseWidget(o)
	return o
}

func (o *nameEditOverlay) CreateRenderer() fyne.WidgetRenderer {
	return &nameEditOverlayRenderer{o: o, objects: []fyne.CanvasObject{o.bg, o.entry}}
}

func (o *nameEditOverlay) Tapped(e *fyne.PointEvent) {
	if pointInRect(e.Position, o.entryPos, o.entrySize) {
		return
	}
	if o.onDismiss != nil {
		o.onDismiss()
	}
}

func (o *nameEditOverlay) TappedSecondary(e *fyne.PointEvent) {
	o.Tapped(e)
}

func pointInRect(p, origin fyne.Position, size fyne.Size) bool {
	return p.X >= origin.X && p.Y >= origin.Y &&
		p.X < origin.X+size.Width && p.Y < origin.Y+size.Height
}

type nameEditOverlayRenderer struct {
	o       *nameEditOverlay
	objects []fyne.CanvasObject
}

func (r *nameEditOverlayRenderer) Layout(size fyne.Size) {
	r.o.bg.Resize(size)
	r.o.entry.Move(r.o.entryPos)
	r.o.entry.Resize(r.o.entrySize)
}

func (r *nameEditOverlayRenderer) MinSize() fyne.Size {
	return r.o.entry.MinSize()
}

func (r *nameEditOverlayRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *nameEditOverlayRenderer) Refresh() {
	r.o.bg.Refresh()
	r.o.entry.Refresh()
}
func (r *nameEditOverlayRenderer) Destroy() {}

type tappableNameLabel struct {
	widget.Label
	onTap func()
}

func (t *tappableNameLabel) Tapped(_ *fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}

func (t *tappableNameLabel) TappedSecondary(_ *fyne.PointEvent) {}

func (t *tappableNameLabel) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

type nameFocusEntry struct {
	widget.Entry
	onFocusLost func()
}

func (e *nameFocusEntry) FocusLost() {
	e.Entry.FocusLost()
	if e.onFocusLost != nil {
		e.onFocusLost()
	}
}

func (e *nameFocusEntry) TypedShortcut(s fyne.Shortcut) {
	if isSaveShortcut(s) {
		triggerGlobalSave()
		return
	}
	e.Entry.TypedShortcut(s)
}
