package rest

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

func NewCollectionTree(data binding.DataTree, onSelect func()) fyne.CanvasObject {
	//return widget.NewTreeWithData()
	return widget.NewTreeWithData(
		data,
		func(b bool) fyne.CanvasObject {
			return widget.NewLabel("hello")
		},
		func(item binding.DataItem, b bool, object fyne.CanvasObject) {

		},
	)
	//return widget.NewLabel("collection tree")
}
