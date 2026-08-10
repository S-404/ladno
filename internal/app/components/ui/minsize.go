package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// MinSizeBox задаёт минимальный размер дочернего объекта, но позволяет ему
// занимать всё доступное место. Удобно для Split: лимит перетаскивания = Min.
type MinSizeBox struct {
	widget.BaseWidget
	Content fyne.CanvasObject
	Min     fyne.Size
}

func NewMinSizeBox(min fyne.Size, content fyne.CanvasObject) *MinSizeBox {
	b := &MinSizeBox{Content: content, Min: min}
	b.ExtendBaseWidget(b)
	return b
}

func (b *MinSizeBox) CreateRenderer() fyne.WidgetRenderer {
	return &minSizeBoxRenderer{box: b, objects: []fyne.CanvasObject{b.Content}}
}

func (b *MinSizeBox) MinSize() fyne.Size {
	// Fixed floor for Split panes: ignore content growth (e.g. long Entry lines).
	return b.Min
}

type minSizeBoxRenderer struct {
	box     *MinSizeBox
	objects []fyne.CanvasObject
}

func (r *minSizeBoxRenderer) Destroy() {}

func (r *minSizeBoxRenderer) Layout(size fyne.Size) {
	if r.box.Content != nil {
		r.box.Content.Resize(size)
		r.box.Content.Move(fyne.NewPos(0, 0))
	}
}

func (r *minSizeBoxRenderer) MinSize() fyne.Size {
	return r.box.Min
}

func (r *minSizeBoxRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *minSizeBoxRenderer) Refresh() {
	if r.box.Content != nil {
		r.box.Content.Refresh()
	}
}
