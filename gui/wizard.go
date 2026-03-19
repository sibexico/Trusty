package gui

import (
	"errors"
	"math/big"
	"strings"

	"github.com/sibexico/Trusty/crypto"
	"github.com/sibexico/Trusty/storage"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// Modal dialog to guide the user through the key exchange.
func ShowAddUserWizard(win fyne.Window, store *storage.Store, onComplete func()) {
	wizardWindow := fyne.CurrentApp().NewWindow("Add New Contact (3 Steps)")

	// State between steps
	var contactName string
	var privateKey *big.Int
	var sharedKey []byte

	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Enter contact's name")
	var step1, step2, step3 fyne.CanvasObject
	gotoStep3 := func(theirPubKeyB64 string) {
		theirPubKeyB64 = strings.TrimSpace(theirPubKeyB64)
		if theirPubKeyB64 == "" {
			dialog.ShowError(errors.New("Received key cannot be empty"), wizardWindow)
			return
		}
		var err error
		sharedKey, err = crypto.ComputeSharedSecret(privateKey, theirPubKeyB64)
		if err != nil {
			dialog.ShowError(err, wizardWindow)
			return
		}

		secretEntry := widget.NewPasswordEntry()
		secretEntry.SetPlaceHolder("Enter the pre-shared secret...")

		authCodeLabel := widget.NewLabel("Confirmation Code will appear here.")

		secretEntry.OnChanged = func(s string) {
			if s == "" {
				authCodeLabel.SetText("Confirmation Code will appear here.")
				return
			}
			authCode := crypto.GenerateAuthCode(sharedKey, s)
			authCodeLabel.SetText(authCode)
		}

		finishButton := widget.NewButton("Confirm & Add Contact", func() {
			if secretEntry.Text == "" {
				dialog.ShowError(errors.New("Pre-shared secret cannot be empty"), wizardWindow)
				return
			}
			newContact := &storage.Contact{Name: contactName, SharedKey: sharedKey}
			if err := store.AddContact(newContact); err != nil {
				dialog.ShowError(err, wizardWindow)
				return
			}

			if onComplete != nil {
				onComplete()
			}
			wizardWindow.Close()
		})

		step3 = container.NewVBox(
			widget.NewLabel("Step 3: Verification"),
			widget.NewForm(
				widget.NewFormItem("Instructions", widget.NewLabel("1. Enter the secret phrase you and your contact agreed on.\n2. Read your Confirmation Code to your contact over a trusted channel (e.g., phone call).\n3. Ensure it EXACTLY matches the code they see.\n4. If it matches, click Confirm.")),
				widget.NewFormItem("Pre-Shared Secret", secretEntry),
				widget.NewFormItem("Confirmation Code", authCodeLabel),
			),
			finishButton,
		)
		wizardWindow.SetContent(step3)
	}

	gotoStep2 := func() {
		contactName = strings.TrimSpace(nameEntry.Text)
		if contactName == "" {
			dialog.ShowError(errors.New("Name cannot be empty"), wizardWindow)
			return
		}
		if _, exists := store.Contacts[contactName]; exists {
			dialog.ShowError(errors.New("A contact with this name already exists"), wizardWindow)
			return
		}

		var myPubKey string
		var err error
		privateKey, myPubKey, err = crypto.GenerateDHKeyPair()
		if err != nil {
			dialog.ShowError(err, wizardWindow)
			return
		}

		myPubKeyEntry := widget.NewMultiLineEntry()
		myPubKeyEntry.SetText(myPubKey)
		myPubKeyEntry.Wrapping = fyne.TextWrapWord
		myPubKeyEntry.Disable()

		theirPubKeyEntry := widget.NewMultiLineEntry()
		theirPubKeyEntry.Wrapping = fyne.TextWrapWord

		nextButton2 := widget.NewButton("Next", func() {
			gotoStep3(theirPubKeyEntry.Text)
		})

		step2 = container.NewVBox(
			widget.NewLabel("Step 2: Key Exchange"),
			widget.NewForm(
				widget.NewFormItem("Your Public Key", widget.NewLabel("Copy this text block and send it to your contact.")),
			),
			myPubKeyEntry,
			widget.NewForm(
				widget.NewFormItem("Their Public Key", widget.NewLabel("Paste the text block you received from them here.")),
			),
			theirPubKeyEntry,
			nextButton2,
		)
		wizardWindow.SetContent(step2)
	}

	nextButton1 := widget.NewButton("Next", gotoStep2)
	step1 = container.NewVBox(
		widget.NewLabel("Step 1: Contact Name"),
		widget.NewForm(widget.NewFormItem("Name", nameEntry)),
		widget.NewLabel("First, agree on a 'pre-shared secret' with your contact over the phone or in person."),
		nextButton1,
	)

	wizardWindow.SetContent(step1)
	wizardWindow.Resize(fyne.NewSize(550, 500))
	wizardWindow.CenterOnScreen()
	wizardWindow.Show()
}
