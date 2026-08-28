package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

func NewLoader(isActive *binding.Bool) fyne.CanvasObject {
	activity := widget.NewActivity()
	activity.Hide()

	(*isActive).AddListener(binding.NewDataListener(func() {
		val, _ := (*isActive).Get()
		if val {
			activity.Show()
			activity.Start()
		} else {
			activity.Hide()
			activity.Stop()
		}
	}))

	return activity
}
