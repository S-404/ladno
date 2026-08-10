package rest

import (
	"fyne.io/fyne/v2"
	"github.com/s-404/ladno/internal/app/components/ui"
)

// NewRequestHeaders возвращает таб Headers.
// onChange вызывается при любом изменении строк.
func NewRequestHeaders(initial []ui.KVRow, onChange func(rows []ui.KVRow)) fyne.CanvasObject {
	return ui.NewKVTable(initial, onChange)
}
