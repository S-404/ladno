package components

import (
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

func NewActivity(isActive *binding.Bool) *widget.Activity {
	activity := widget.NewActivity()
	activity.Hide()

	(*isActive).AddListener(binding.NewDataListener(func() {
		val, _ := (*isActive).Get()
		if val {
			activity.Start()
			activity.Show()
		} else {
			activity.Hide()
			activity.Stop()
		}
	}))

	return activity
}
