package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const DragThreshold = float32(6)

var ColorInsert = color.NRGBA{R: 0x1E, G: 0x88, B: 0xE5, A: 0xFF}

// HoldDrag — порог движения перед захватом (без hold-таймера).
type HoldDrag struct {
	DragAccum float32
	Armed     bool
	Dragging  bool
}

func (h *HoldDrag) MouseDown() {
	h.DragAccum = 0
	h.Armed = false
	h.Dragging = false
}

func (h *HoldDrag) MouseUp() {
	h.DragAccum = 0
	h.Armed = false
}

// Dragged обновляет состояние. started=true когда захват только начался;
// active=true пока drag идёт (после порога).
func (h *HoldDrag) Dragged(dx, dy float32) (started, active bool) {
	if !h.Armed {
		h.DragAccum += abs32(dx) + abs32(dy)
		if h.DragAccum < DragThreshold {
			return false, false
		}
		h.Armed = true
		h.Dragging = true
		return true, true
	}
	return false, true
}

func (h *HoldDrag) DragEnd() (wasDragging bool) {
	wasDragging = h.Dragging
	h.Dragging = false
	h.Armed = false
	h.DragAccum = 0
	return wasDragging
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// DragGhost — плавающее превью при перетаскивании.
type DragGhost struct {
	popup *widget.PopUp
	label *widget.Label
	icon  *widget.Icon
}

func (g *DragGhost) Show(host fyne.CanvasObject, text string, res fyne.Resource) {
	c := fyne.CurrentApp().Driver().CanvasForObject(host)
	if c == nil {
		return
	}
	if g.popup == nil {
		g.label = widget.NewLabel(text)
		g.icon = widget.NewIcon(res)
		bg := canvas.NewRectangle(theme.Color(theme.ColorNameOverlayBackground))
		content := container.NewPadded(container.NewHBox(g.icon, g.label))
		g.popup = widget.NewPopUp(container.NewStack(bg, content), c)
	} else {
		g.label.SetText(text)
		if res != nil {
			g.icon.SetResource(res)
		}
	}
	g.popup.Resize(g.popup.MinSize())
	g.popup.Show()
}

func (g *DragGhost) Move(abs fyne.Position) {
	if g.popup == nil || !g.popup.Visible() {
		return
	}
	g.popup.Move(fyne.NewPos(abs.X+12, abs.Y-10))
	g.popup.Refresh()
}

func (g *DragGhost) Hide() {
	if g.popup != nil {
		g.popup.Hide()
	}
}
