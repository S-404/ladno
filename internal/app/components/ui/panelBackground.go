package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// PanelBackground — непрозрачная подложка панели (чтобы Split не просвечивал).
type PanelBackground struct {
	widget.BaseWidget
	content fyne.CanvasObject
	bg      *canvas.Rectangle
}

func NewPanelBackground(content fyne.CanvasObject) *PanelBackground {
	p := &PanelBackground{
		content: content,
		bg:      canvas.NewRectangle(theme.Color(theme.ColorNameBackground)),
	}
	p.ExtendBaseWidget(p)
	return p
}

func (p *PanelBackground) CreateRenderer() fyne.WidgetRenderer {
	stack := container.NewStack(p.bg, p.content)
	return &panelBackgroundRenderer{panel: p, stack: stack, objects: []fyne.CanvasObject{stack}}
}

func (p *PanelBackground) MinSize() fyne.Size {
	if p.content != nil {
		return p.content.MinSize()
	}
	return fyne.NewSize(0, 0)
}

type panelBackgroundRenderer struct {
	panel   *PanelBackground
	stack   *fyne.Container
	objects []fyne.CanvasObject
}

func (r *panelBackgroundRenderer) Destroy() {}

func (r *panelBackgroundRenderer) Layout(size fyne.Size) {
	r.stack.Resize(size)
	r.stack.Move(fyne.NewPos(0, 0))
}

func (r *panelBackgroundRenderer) MinSize() fyne.Size {
	return r.panel.MinSize()
}

func (r *panelBackgroundRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *panelBackgroundRenderer) Refresh() {
	r.panel.bg.FillColor = theme.Color(theme.ColorNameBackground)
	r.panel.bg.Refresh()
	r.stack.Refresh()
}
