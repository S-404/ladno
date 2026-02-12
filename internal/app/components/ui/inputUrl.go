package ui

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// --- Токенизатор URL ---

type tokenKind int

const (
	tokenPlain     tokenKind = iota
	tokenVariable            // {{...}}
	tokenPathParam           // :param
)

type urlToken struct {
	text string
	kind tokenKind
}

func parseURL(raw string) []urlToken {
	var tokens []urlToken
	plain := &strings.Builder{}
	i := 0

	flush := func() {
		if plain.Len() > 0 {
			tokens = append(tokens, urlToken{plain.String(), tokenPlain})
			plain.Reset()
		}
	}

	for i < len(raw) {
		// {{variable}}
		if i+1 < len(raw) && raw[i] == '{' && raw[i+1] == '{' {
			flush()
			end := strings.Index(raw[i:], "}}")
			if end == -1 {
				plain.WriteString(raw[i:])
				break
			}
			end += i + 2
			tokens = append(tokens, urlToken{raw[i:end], tokenVariable})
			i = end
			continue
		}

		// :param — пропускаем ://
		if raw[i] == ':' {
			if i+2 < len(raw) && raw[i+1] == '/' && raw[i+2] == '/' {
				plain.WriteByte(raw[i])
				i++
				continue
			}
			flush()
			j := i + 1
			for j < len(raw) && isParamChar(raw[j]) {
				j++
			}
			if j > i+1 {
				tokens = append(tokens, urlToken{raw[i:j], tokenPathParam})
				i = j
				continue
			}
			plain.WriteByte(raw[i])
			i++
			continue
		}

		plain.WriteByte(raw[i])
		i++
	}
	flush()
	return tokens
}

func isParamChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_' || c == '-'
}

// tokenColor возвращает цвет для типа токена
func tokenColor(kind tokenKind) color.Color {
	switch kind {
	case tokenVariable:
		return color.NRGBA{R: 255, G: 165, B: 0, A: 255} // оранжевый
	case tokenPathParam:
		return color.NRGBA{R: 86, G: 182, B: 255, A: 255} // голубой
	default:
		return theme.Color(theme.ColorNameForeground)
	}
}

// --- highlightDisplay — display-слой через canvas.Text в HBox ---

// highlightDisplay отображает подсвеченный URL как набор canvas.Text.
// Не использует widget.RichText — только canvas объекты в контейнере.
type highlightDisplay struct {
	widget.BaseWidget
	box *fyne.Container // container.NewHBox с canvas.Text элементами
}

func newHighlightDisplay() *highlightDisplay {
	h := &highlightDisplay{
		box: container.NewHBox(),
	}
	h.ExtendBaseWidget(h)
	return h
}

func (h *highlightDisplay) SetText(raw string) {
	tokens := parseURL(raw)
	objects := make([]fyne.CanvasObject, 0, len(tokens))

	for _, t := range tokens {
		txt := canvas.NewText(t.text, tokenColor(t.kind))
		txt.TextStyle = fyne.TextStyle{Monospace: true}
		txt.TextSize = theme.TextSize()
		objects = append(objects, txt)
	}

	h.box.Objects = objects
	h.box.Refresh()
}

func (h *highlightDisplay) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(h.box)
}

func (h *highlightDisplay) MinSize() fyne.Size {
	return h.box.MinSize()
}

// --- urlEntry — Entry с перехватом FocusLost ---

type urlEntry struct {
	widget.Entry
	onFocusLost func()
}

func newUrlEntry(value binding.String, onFocusLost func()) *urlEntry {
	e := &urlEntry{onFocusLost: onFocusLost}
	e.ExtendBaseWidget(e)
	e.Bind(value)
	e.TextStyle = fyne.TextStyle{Monospace: true}
	return e
}

func (e *urlEntry) FocusLost() {
	e.Entry.FocusLost()
	if e.onFocusLost != nil {
		e.onFocusLost()
	}
}

// --- UrlInput ---

// UrlInput — адресная строка в стиле Postman.
//
// display режим: подсвеченный URL ({{var}} оранжевый, :param голубой)
// edit режим:    обычный Entry при клике, возврат в display при потере фокуса
//
// Использование:
//
//	val := binding.NewString()
//	val.Set("{{host}}/api/user/:id")
//	input := ui.NewUrlInput(val)
type UrlInput struct {
	widget.BaseWidget

	value   binding.String
	focused bool

	entry   *urlEntry
	display *highlightDisplay
	bg      *canvas.Rectangle
	border  *canvas.Rectangle
}

func NewUrlInput(value binding.String) *UrlInput {
	u := &UrlInput{value: value}
	u.ExtendBaseWidget(u)

	u.entry = newUrlEntry(value, func() {
		u.focused = false
		u.Refresh()
	})

	u.display = newHighlightDisplay()

	u.bg = canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	u.bg.CornerRadius = theme.InputRadiusSize()

	u.border = canvas.NewRectangle(color.Transparent)
	u.border.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	u.border.StrokeWidth = theme.InputBorderSize()
	u.border.CornerRadius = theme.InputRadiusSize()

	// Слушаем изменения для обновления display
	value.AddListener(binding.NewDataListener(func() {
		raw, _ := value.Get()
		u.display.SetText(raw)
	}))

	// Начальное состояние — display режим
	u.entry.Hide()

	return u
}

// Tapped — клик переключает в режим редактирования
func (u *UrlInput) Tapped(_ *fyne.PointEvent) {
	if !u.focused {
		u.focused = true
		u.Refresh()
		if cnv := fyne.CurrentApp().Driver().CanvasForObject(u); cnv != nil {
			cnv.Focus(u.entry)
		}
	}
}

func (u *UrlInput) CreateRenderer() fyne.WidgetRenderer {
	displayLayer := container.NewStack(
		u.bg,
		u.border,
		container.NewPadded(u.display),
	)
	stack := container.NewStack(displayLayer, u.entry)
	return widget.NewSimpleRenderer(stack)
}

func (u *UrlInput) MinSize() fyne.Size {
	return u.entry.MinSize()
}

func (u *UrlInput) Refresh() {
	u.BaseWidget.Refresh()

	if u.focused {
		u.display.Hide()
		u.bg.Hide()
		u.border.Hide()
		u.entry.Show()
		u.entry.Refresh()
	} else {
		u.entry.Hide()
		raw, _ := u.value.Get()
		u.display.SetText(raw)
		u.bg.Show()
		u.border.Show()
		u.display.Show()
		u.display.Refresh()
	}
}
