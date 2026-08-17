package ui

import (
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

var (
	globalSaveMu sync.RWMutex
	globalSave   func()
)

// SetGlobalSaveHandler sets the Ctrl+S handler used by focused text inputs.
func SetGlobalSaveHandler(fn func()) {
	globalSaveMu.Lock()
	globalSave = fn
	globalSaveMu.Unlock()
}

func triggerGlobalSave() {
	globalSaveMu.RLock()
	fn := globalSave
	globalSaveMu.RUnlock()
	if fn != nil {
		fn()
	}
}

func isSaveShortcut(s fyne.Shortcut) bool {
	cs, ok := s.(*desktop.CustomShortcut)
	return ok && cs.KeyName == fyne.KeyS && cs.Modifier&fyne.KeyModifierControl != 0
}

// Entry is widget.Entry that forwards Ctrl+S to the global save handler.
type Entry struct {
	widget.Entry
}

func NewEntry() *Entry {
	e := &Entry{}
	e.ExtendBaseWidget(e)
	return e
}

func (e *Entry) TypedShortcut(s fyne.Shortcut) {
	if isSaveShortcut(s) {
		triggerGlobalSave()
		return
	}
	e.Entry.TypedShortcut(s)
}

// MultiLineEntry is widget.Entry (multiline) that forwards Ctrl+S.
type MultiLineEntry struct {
	widget.Entry
}

func NewMultiLineEntry() *MultiLineEntry {
	e := &MultiLineEntry{}
	e.ExtendBaseWidget(e)
	e.MultiLine = true
	e.Wrapping = fyne.TextWrapWord
	return e
}

func (e *MultiLineEntry) TypedShortcut(s fyne.Shortcut) {
	if isSaveShortcut(s) {
		triggerGlobalSave()
		return
	}
	e.Entry.TypedShortcut(s)
}
