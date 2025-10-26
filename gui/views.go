package gui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"trusty/crypto"
	"trusty/storage"
)

func createMessagingView(win fyne.Window, contact *storage.Contact, store *storage.Store) fyne.CanvasObject {
	messageContainer := container.NewVBox()
	for _, msg := range store.Messages[contact.Name] {
		messageWidget := createMessageWidget(msg)
		messageContainer.Add(messageWidget)
	}

	historyScroll := container.NewScroll(messageContainer)
	newMessageEntry := widget.NewMultiLineEntry()
	newMessageEntry.SetPlaceHolder("Type your message to encrypt...")
	newMessageEntry.Wrapping = fyne.TextWrapWord
	newMessageEntry.SetMinRowsVisible(4)

	encryptButton := widget.NewButtonWithIcon("Encrypt & Copy", theme.UploadIcon(), func() {
		if newMessageEntry.Text == "" {
			return
		}

		encrypted, err := crypto.Encrypt([]byte(newMessageEntry.Text), contact.SharedKey)
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		win.Clipboard().SetContent(encrypted)

		msg := &storage.Message{Timestamp: time.Now().Unix(), IsSent: true, Content: newMessageEntry.Text}
		store.AddMessage(contact.Name, msg)

		newWidget := createMessageWidget(msg)
		messageContainer.Add(newWidget)
		historyScroll.ScrollToBottom() // Auto-scroll to the new message.

		newMessageEntry.SetText("")
		dialog.ShowInformation("Success", "Encrypted message copied to clipboard.", win)
	})

	ciphertextEntry := widget.NewMultiLineEntry()
	ciphertextEntry.SetPlaceHolder("Paste received message to decrypt...")
	ciphertextEntry.Wrapping = fyne.TextWrapWord
	ciphertextEntry.SetMinRowsVisible(4)

	decryptButton := widget.NewButtonWithIcon("Decrypt & View", theme.DownloadIcon(), func() {
		if ciphertextEntry.Text == "" {
			return
		}

		decrypted, err := crypto.Decrypt(ciphertextEntry.Text, contact.SharedKey)
		if err != nil {
			dialog.ShowError(err, win)
			return
		}

		msg := &storage.Message{Timestamp: time.Now().Unix(), IsSent: false, Content: decrypted}
		store.AddMessage(contact.Name, msg)
		newWidget := createMessageWidget(msg)
		messageContainer.Add(newWidget)
		historyScroll.ScrollToBottom()
		ciphertextEntry.SetText("")
	})

	encryptBox := container.NewBorder(nil, encryptButton, nil, nil, newMessageEntry)
	decryptBox := container.NewBorder(nil, decryptButton, nil, nil, ciphertextEntry)
	formSplit := container.NewHSplit(encryptBox, decryptBox)

	return container.NewBorder(
		container.NewCenter(widget.NewLabel(fmt.Sprintf("Messaging with: %s", contact.Name))), // Top
		formSplit, // Bottom
		nil, nil,  // Left, Right
		historyScroll, // Center
	)
}

func createMessageWidget(msg *storage.Message) fyne.CanvasObject {
	ts := time.Unix(msg.Timestamp, 0)
	timeStr := ts.In(time.Local).Format("Jan 2, 2006 at 3:04 PM")

	prefix := "Received"
	if msg.IsSent {
		prefix = "Sent"
	}

	metadataLabel := widget.NewLabel(fmt.Sprintf("%s on %s", prefix, timeStr))
	metadataLabel.TextStyle.Italic = true

	contentLabel := widget.NewLabel(msg.Content)
	contentLabel.Wrapping = fyne.TextWrapWord

	return container.NewVBox(
		metadataLabel,
		contentLabel,
		widget.NewSeparator(),
	)
}
