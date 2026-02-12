package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	ladnoApp "github.com/s-404/ladno/internal/app"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("ladno")
	myWindow.Resize(fyne.NewSize(1280, 800))
	ladnoApp.Init(myWindow)
	myWindow.ShowAndRun()
}
