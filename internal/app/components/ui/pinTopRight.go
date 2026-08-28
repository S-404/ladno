package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// PinTopRight fills the parent with content and pins accessory to the top-right
// without shrinking content. Clicks outside the accessory go to content.
func NewPinTopRight(content, accessory fyne.CanvasObject) fyne.CanvasObject {
	w := &pinTopRight{content: content, accessory: accessory}
	w.ExtendBaseWidget(w)
	return w
}

type pinTopRight struct {
	widget.BaseWidget
	content   fyne.CanvasObject
	accessory fyne.CanvasObject
}

func (w *pinTopRight) CreateRenderer() fyne.WidgetRenderer {
	return &pinTopRightRenderer{w: w, objects: []fyne.CanvasObject{w.content, w.accessory}}
}

func (w *pinTopRight) MinSize() fyne.Size {
	if w.content == nil {
		return fyne.NewSize(0, 0)
	}
	return w.content.MinSize()
}

type pinTopRightRenderer struct {
	w       *pinTopRight
	objects []fyne.CanvasObject
}

func (r *pinTopRightRenderer) Destroy() {}

func (r *pinTopRightRenderer) Layout(size fyne.Size) {
	if r.w.content != nil {
		r.w.content.Resize(size)
		r.w.content.Move(fyne.NewPos(0, 0))
	}
	if r.w.accessory == nil {
		return
	}
	min := r.w.accessory.MinSize()
	r.w.accessory.Resize(min)
	x := size.Width - min.Width
	if x < 0 {
		x = 0
	}
	r.w.accessory.Move(fyne.NewPos(x, 0))
}

func (r *pinTopRightRenderer) MinSize() fyne.Size {
	return r.w.MinSize()
}

func (r *pinTopRightRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *pinTopRightRenderer) Refresh() {
	if r.w.content != nil {
		r.w.content.Refresh()
	}
	if r.w.accessory != nil {
		r.w.accessory.Refresh()
	}
}
