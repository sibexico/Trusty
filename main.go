package main

import (
	"trusty/gui"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

func main() {
	myApp := app.NewWithID("com.example.securemessenger")
	myWindow := myApp.NewWindow("Secure Messenger")
	ui := gui.MakeUI(myWindow)
	myWindow.SetContent(ui)
	myWindow.Resize(fyne.NewSize(800, 600))
	myWindow.ShowAndRun()
}
