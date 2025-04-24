package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	gooseApp "github.com/s-404/goose/internal/app"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("goose")
	myWindow.Resize(fyne.NewSize(1280, 800))
	gooseApp.Init(myWindow)
	myWindow.ShowAndRun()
}
