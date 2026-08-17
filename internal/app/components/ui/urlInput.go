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

// url tokenizer

type tokenKind int

const (
	tokenPlain       tokenKind = iota
	tokenVariable              // {{...}}
	tokenPathParam             // :param
	tokenQueryKey              // ?key= or &key=
	tokenQueryEquals           // =
	tokenQueryValue            // value after =
	tokenQuerySep              // ? and &
)

type urlToken struct {
	text string
	kind tokenKind
}

// parseURL split url per tokens
func parseURL(raw string) []urlToken {
	qIdx := queryStart(raw)

	if qIdx == -1 {
		// if no query part then only path parsing
		return parsePath(raw)
	}

	tokens := parsePath(raw[:qIdx])
	tokens = append(tokens, parseQuery(raw[qIdx:])...)
	return tokens
}

// queryStart returns index of first '?' outside {{...}}
func queryStart(raw string) int {
	i := 0
	for i < len(raw) {
		if i+1 < len(raw) && raw[i] == '{' && raw[i+1] == '{' {
			end := strings.Index(raw[i:], "}}")
			if end == -1 {
				return -1
			}
			i += end + 2
			continue
		}
		if raw[i] == '?' {
			return i
		}
		i++
	}
	return -1
}

// parsePath  URL part before '?' — variables and path-parameters
func parsePath(raw string) []urlToken {
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

		// skip :param
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

// parseQuery handle query-string starts with '?'
func parseQuery(raw string) []urlToken {
	var tokens []urlToken
	i := 0

	for i < len(raw) {
		// ? or & separator
		if raw[i] == '?' || raw[i] == '&' {
			tokens = append(tokens, urlToken{string(raw[i]), tokenQuerySep})
			i++

			// read key to the '=' / '&' / end
			j := i
			for j < len(raw) && raw[j] != '=' && raw[j] != '&' {
				j++
			}
			if j > i {
				tokens = append(tokens, urlToken{raw[i:j], tokenQueryKey})
				i = j
			}

			// read '='
			if i < len(raw) && raw[i] == '=' {
				tokens = append(tokens, urlToken{"=", tokenQueryEquals})
				i++

				// read value to the '&' or end, with highlight {{var}} and :param
				tokens = append(tokens, parseQueryValue(raw, &i)...)
			}
			continue
		}
		i++
	}

	return tokens
}

// parseQueryValue read value of query-parameter to the '&' or end.
// highlights {{var}} and :param.
func parseQueryValue(raw string, i *int) []urlToken {
	var tokens []urlToken
	plain := &strings.Builder{}

	flush := func() {
		if plain.Len() > 0 {
			tokens = append(tokens, urlToken{plain.String(), tokenQueryValue})
			plain.Reset()
		}
	}

	for *i < len(raw) {
		// end of value
		if raw[*i] == '&' {
			break
		}

		// {{variable}} at value
		if *i+1 < len(raw) && raw[*i] == '{' && raw[*i+1] == '{' {
			flush()
			end := strings.Index(raw[*i:], "}}")
			if end == -1 {
				plain.WriteString(raw[*i:])
				*i = len(raw)
				break
			}
			end += *i + 2
			tokens = append(tokens, urlToken{raw[*i:end], tokenVariable})
			*i = end
			continue
		}

		// :param at value
		if raw[*i] == ':' {
			flush()
			j := *i + 1
			for j < len(raw) && raw[j] != '&' && isParamChar(raw[j]) {
				j++
			}
			if j > *i+1 {
				tokens = append(tokens, urlToken{raw[*i:j], tokenPathParam})
				*i = j
				continue
			}
			plain.WriteByte(raw[*i])
			*i++
			continue
		}

		plain.WriteByte(raw[*i])
		*i++
	}
	flush()
	return tokens
}

func isParamChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_' || c == '-'
}

// tokenColor returns color for token type
func tokenColor(kind tokenKind) color.Color {
	switch kind {
	case tokenVariable:
		return EnvVarColor // оранжевый  {{var}}
	case tokenPathParam:
		return color.NRGBA{R: 86, G: 182, B: 255, A: 255} // голубой    :param
	case tokenQueryKey:
		return color.NRGBA{R: 130, G: 210, B: 130, A: 255} // зелёный    key
	case tokenQueryEquals, tokenQuerySep:
		return color.NRGBA{R: 140, G: 140, B: 140, A: 255} // серый      ? & =
	case tokenQueryValue:
		return theme.Color(theme.ColorNameForeground) // обычный    value
	default:
		return theme.Color(theme.ColorNameForeground)
	}
}

// --- highlightDisplay — display-layer through canvas.Text in HBox ---

type highlightDisplay struct {
	widget.BaseWidget
	box *fyne.Container
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

// --- urlEntry — Entry with interceptor FocusLost ---

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

func (e *urlEntry) TypedShortcut(s fyne.Shortcut) {
	if isSaveShortcut(s) {
		triggerGlobalSave()
		return
	}
	e.Entry.TypedShortcut(s)
}

func (e *urlEntry) KeyDown(key *fyne.KeyEvent) {
	switch key.Name {
	case fyne.KeyReturn, fyne.KeyEscape:
		if cnv := fyne.CurrentApp().Driver().CanvasForObject(e); cnv != nil {
			cnv.Unfocus()
		}
	default:
		e.Entry.KeyDown(key)
	}
}

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

	value.AddListener(binding.NewDataListener(func() {
		raw, _ := value.Get()
		u.display.SetText(raw)
	}))

	u.entry.Hide()

	return u
}

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
