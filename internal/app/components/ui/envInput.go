package ui

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// EnvVarColor — цвет вставок {{var}} (как в URL input).
var EnvVarColor = color.NRGBA{R: 255, G: 165, B: 0, A: 255}

type envToken struct {
	text     string
	variable bool
}

// parseEnvTokens splits text into plain and {{var}} segments.
func parseEnvTokens(raw string) []envToken {
	if raw == "" {
		return nil
	}
	var tokens []envToken
	plain := &strings.Builder{}
	i := 0

	flush := func() {
		if plain.Len() > 0 {
			tokens = append(tokens, envToken{text: plain.String()})
			plain.Reset()
		}
	}

	for i < len(raw) {
		if i+1 < len(raw) && raw[i] == '{' && raw[i+1] == '{' {
			flush()
			end := strings.Index(raw[i:], "}}")
			if end == -1 {
				plain.WriteString(raw[i:])
				break
			}
			end += i + 2
			tokens = append(tokens, envToken{text: raw[i:end], variable: true})
			i = end
			continue
		}
		plain.WriteByte(raw[i])
		i++
	}
	flush()
	return tokens
}

type envHighlightDisplay struct {
	widget.BaseWidget
	box       *fyne.Container
	multiline bool
}

func newEnvHighlightDisplay(multiline bool) *envHighlightDisplay {
	h := &envHighlightDisplay{
		box:       container.NewVBox(),
		multiline: multiline,
	}
	h.ExtendBaseWidget(h)
	return h
}

func (h *envHighlightDisplay) SetText(raw string) {
	fg := theme.Color(theme.ColorNameForeground)
	size := theme.TextSize()
	style := fyne.TextStyle{Monospace: true}

	makeLine := func(line string) fyne.CanvasObject {
		tokens := parseEnvTokens(line)
		if len(tokens) == 0 {
			txt := canvas.NewText(" ", fg)
			txt.TextStyle = style
			txt.TextSize = size
			return container.NewHBox(txt)
		}
		objects := make([]fyne.CanvasObject, 0, len(tokens))
		for _, t := range tokens {
			c := fg
			if t.variable {
				c = EnvVarColor
			}
			txt := canvas.NewText(t.text, c)
			txt.TextStyle = style
			txt.TextSize = size
			objects = append(objects, txt)
		}
		return container.NewHBox(objects...)
	}

	if !h.multiline {
		h.box.Objects = []fyne.CanvasObject{makeLine(raw)}
		h.box.Refresh()
		return
	}

	lines := strings.Split(raw, "\n")
	objects := make([]fyne.CanvasObject, 0, len(lines))
	for _, line := range lines {
		objects = append(objects, makeLine(line))
	}
	h.box.Objects = objects
	h.box.Refresh()
}

func (h *envHighlightDisplay) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(h.box)
}

func (h *envHighlightDisplay) MinSize() fyne.Size {
	return h.box.MinSize()
}

type envFocusEntry struct {
	widget.Entry
	onFocusLost func()
}

func (e *envFocusEntry) FocusLost() {
	e.Entry.FocusLost()
	if e.onFocusLost != nil {
		e.onFocusLost()
	}
}

// EnvInput — Entry с подсветкой {{var}} вне фокуса.
type EnvInput struct {
	widget.BaseWidget

	focused   bool
	multiline bool

	entry   *envFocusEntry
	display *envHighlightDisplay
	bg      *canvas.Rectangle
	border  *canvas.Rectangle
	scroll  *container.Scroll
}

func NewEnvInput() *EnvInput {
	return newEnvInput(false)
}

func NewEnvMultiLineInput() *EnvInput {
	return newEnvInput(true)
}

func newEnvInput(multiline bool) *EnvInput {
	e := &EnvInput{multiline: multiline}
	e.ExtendBaseWidget(e)

	e.entry = &envFocusEntry{}
	e.entry.ExtendBaseWidget(e.entry)
	e.entry.TextStyle = fyne.TextStyle{Monospace: true}
	if multiline {
		e.entry.MultiLine = true
		e.entry.Wrapping = fyne.TextWrapOff
	}
	e.entry.onFocusLost = func() {
		e.focused = false
		e.Refresh()
	}

	e.display = newEnvHighlightDisplay(multiline)

	e.bg = canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	e.bg.CornerRadius = theme.InputRadiusSize()

	e.border = canvas.NewRectangle(color.Transparent)
	e.border.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	e.border.StrokeWidth = theme.InputBorderSize()
	e.border.CornerRadius = theme.InputRadiusSize()

	if multiline {
		e.scroll = container.NewScroll(e.display)
	}

	e.entry.Hide()
	return e
}

func (e *EnvInput) Text() string {
	return e.entry.Text
}

func (e *EnvInput) SetText(s string) {
	e.entry.SetText(s)
	if !e.focused {
		e.display.SetText(s)
	}
}

func (e *EnvInput) SetPlaceHolder(s string) {
	e.entry.SetPlaceHolder(s)
}

func (e *EnvInput) SetMinRowsVisible(rows int) {
	if e.multiline {
		e.entry.SetMinRowsVisible(rows)
	}
}

func (e *EnvInput) OnChanged(f func(string)) {
	e.entry.OnChanged = f
}

func (e *EnvInput) Disable() {
	e.entry.Disable()
}

func (e *EnvInput) Enable() {
	e.entry.Enable()
}

func (e *EnvInput) Tapped(_ *fyne.PointEvent) {
	if e.entry.Disabled() {
		return
	}
	if !e.focused {
		e.focused = true
		e.Refresh()
		if cnv := fyne.CurrentApp().Driver().CanvasForObject(e); cnv != nil {
			cnv.Focus(e.entry)
		}
	}
}

func (e *EnvInput) CreateRenderer() fyne.WidgetRenderer {
	var displayLayer fyne.CanvasObject
	if e.multiline && e.scroll != nil {
		displayLayer = container.NewStack(e.bg, e.border, container.NewPadded(e.scroll))
	} else {
		displayLayer = container.NewStack(e.bg, e.border, container.NewPadded(e.display))
	}
	stack := container.NewStack(displayLayer, e.entry)
	return &envInputRenderer{input: e, stack: stack, objects: []fyne.CanvasObject{stack}}
}

func (e *EnvInput) MinSize() fyne.Size {
	// Long lines must not expand layout — parent clips/scrolls.
	s := e.entry.MinSize()
	if s.Width < 80 {
		s.Width = 80
	} else if s.Width > 160 {
		s.Width = 160
	}
	return s
}

type envInputRenderer struct {
	input   *EnvInput
	stack   *fyne.Container
	objects []fyne.CanvasObject
}

func (r *envInputRenderer) Destroy() {}

func (r *envInputRenderer) Layout(size fyne.Size) {
	r.stack.Resize(size)
	r.stack.Move(fyne.NewPos(0, 0))
}

func (r *envInputRenderer) MinSize() fyne.Size {
	return r.input.MinSize()
}

func (r *envInputRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *envInputRenderer) Refresh() {
	r.stack.Refresh()
}

func (e *EnvInput) Refresh() {
	e.BaseWidget.Refresh()
	e.bg.FillColor = theme.Color(theme.ColorNameInputBackground)
	e.border.StrokeColor = theme.Color(theme.ColorNameInputBorder)

	if e.focused {
		e.display.Hide()
		if e.scroll != nil {
			e.scroll.Hide()
		}
		e.bg.Hide()
		e.border.Hide()
		e.entry.Show()
		e.entry.Refresh()
		return
	}

	e.entry.Hide()
	e.display.SetText(e.entry.Text)
	e.bg.Show()
	e.border.Show()
	if e.scroll != nil {
		e.scroll.Show()
		e.scroll.Refresh()
	}
	e.display.Show()
	e.display.Refresh()
}
