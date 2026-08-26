package ui

import (
	"image/color"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var (
	listScrollMu sync.Mutex
	listScrolls  []*container.Scroll
)

// NewListVScroll is a vertical scroll for lists that contain text fields.
// Wheel events over those fields are forwarded here via ListWheelField.
func NewListVScroll(content fyne.CanvasObject) *container.Scroll {
	s := container.NewVScroll(content)
	listScrollMu.Lock()
	listScrolls = append(listScrolls, s)
	listScrollMu.Unlock()
	return s
}

func containingListScroll(obj fyne.CanvasObject) fyne.Scrollable {
	d := fyne.CurrentApp().Driver()
	if d == nil || obj == nil {
		return nil
	}
	pos := d.AbsolutePositionForObject(obj)
	sz := obj.Size()
	if sz.Width <= 0 || sz.Height <= 0 {
		return nil
	}
	cx := pos.X + sz.Width/2
	cy := pos.Y + sz.Height/2

	listScrollMu.Lock()
	defer listScrollMu.Unlock()

	var best *container.Scroll
	var bestArea float32
	for _, s := range listScrolls {
		if s == nil {
			continue
		}
		sp := d.AbsolutePositionForObject(s)
		ss := s.Size()
		if ss.Width <= 0 || ss.Height <= 0 {
			continue
		}
		if cx < sp.X || cy < sp.Y || cx >= sp.X+ss.Width || cy >= sp.Y+ss.Height {
			continue
		}
		area := ss.Width * ss.Height
		if best == nil || area < bestArea {
			best = s
			bestArea = area
		}
	}
	return best
}

// wheelPass sits above a field, catches wheel, and scrolls the parent list.
type wheelPass struct {
	widget.BaseWidget
}

func newWheelPass() *wheelPass {
	w := &wheelPass{}
	w.ExtendBaseWidget(w)
	return w
}

func (w *wheelPass) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(color.Transparent))
}

func (w *wheelPass) MinSize() fyne.Size {
	return fyne.NewSize(1, 1)
}

func (w *wheelPass) Scrolled(ev *fyne.ScrollEvent) {
	if s := containingListScroll(w); s != nil {
		s.Scrolled(ev)
	}
	// If no list scroll owns this field, the event is ignored (no field scroll on wheel).
}

// ListWheelField overlays content so the mouse wheel scrolls the parent list
// instead of the field's internal scroll. Clicks still reach the field below.
func ListWheelField(content fyne.CanvasObject) fyne.CanvasObject {
	return container.NewStack(content, newWheelPass())
}
