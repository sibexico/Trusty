package main

import (
	"github.com/sibexico/Trusty/gui"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

func main() {
	myApp := app.NewWithID("com.github.sibexico.trusty")
	myWindow := myApp.NewWindow("Trusty")
	ui := gui.MakeUI(myWindow)
	myWindow.SetContent(ui)
	myWindow.Resize(fyne.NewSize(800, 600))
	myWindow.ShowAndRun()
}
