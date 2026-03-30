package gui

import (
	"fmt"
	"log"
	"path/filepath"
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	fynestorage "fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/sibexico/Trusty/storage"
)

func MakeUI(win fyne.Window) fyne.CanvasObject {
	root := container.NewMax()

	var showProfileGate func(initialPath string)
	var showMainView func(store *storage.Store, profilePath string)

	showMainView = func(store *storage.Store, profilePath string) {
		main := createMainView(win, store, filepath.Base(profilePath), func() {
			showProfileGate(profilePath)
		})
		root.Objects = []fyne.CanvasObject{main}
		root.Refresh()
	}

	showProfileGate = func(initialPath string) {
		profilePathEntry := widget.NewEntry()
		profilePathEntry.SetPlaceHolder("Select or create a profile file...")
		profilePathEntry.SetText(initialPath)

		passphraseEntry := widget.NewPasswordEntry()
		passphraseEntry.SetPlaceHolder("Enter profile passphrase")

		statusLabel := widget.NewLabel("Select or create a profile to continue.")

		selectButton := widget.NewButtonWithIcon("Select Profile File", theme.FolderOpenIcon(), func() {
			picker := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
				if err != nil {
					dialog.ShowError(err, win)
					return
				}
				if reader == nil {
					return
				}
				selectedPath := reader.URI().Path()
				_ = reader.Close()
				profilePathEntry.SetText(selectedPath)
			}, win)

			if profilesDir, err := storage.ProfilesDir(); err == nil {
				if dirURI := fynestorage.NewFileURI(profilesDir); dirURI != nil {
					if lister, listerErr := fynestorage.ListerForURI(dirURI); listerErr == nil {
						picker.SetLocation(lister)
					}
				}
			}

			picker.Show()
		})

		createButton := widget.NewButtonWithIcon("Create Profile File", theme.DocumentCreateIcon(), func() {
			path := profilePathEntry.Text
			passphrase := passphraseEntry.Text
			if passphrase == "" {
				dialog.ShowError(fmt.Errorf("passphrase cannot be empty"), win)
				return
			}

			saver := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
				if err != nil {
					dialog.ShowError(err, win)
					return
				}
				if writer == nil {
					return
				}
				selectedPath := writer.URI().Path()
				_ = writer.Close()

				profilePathEntry.SetText(selectedPath)
				newStore, createErr := storage.NewStore(selectedPath, passphrase)
				if createErr != nil {
					dialog.ShowError(createErr, win)
					return
				}
				if saveErr := newStore.Save(); saveErr != nil {
					dialog.ShowError(saveErr, win)
					return
				}
				showMainView(newStore, selectedPath)
			}, win)

			if path != "" {
				saver.SetFileName(filepath.Base(path))
			}
			if profilesDir, err := storage.ProfilesDir(); err == nil {
				if dirURI := fynestorage.NewFileURI(profilesDir); dirURI != nil {
					if lister, listerErr := fynestorage.ListerForURI(dirURI); listerErr == nil {
						saver.SetLocation(lister)
					}
				}
			}
			saver.Show()
		})

		unlockButton := widget.NewButtonWithIcon("Open Profile", theme.ConfirmIcon(), func() {
			profilePath := profilePathEntry.Text
			passphrase := passphraseEntry.Text
			if profilePath == "" {
				dialog.ShowError(fmt.Errorf("profile path cannot be empty"), win)
				return
			}
			if passphrase == "" {
				dialog.ShowError(fmt.Errorf("passphrase cannot be empty"), win)
				return
			}

			store, err := storage.NewStore(profilePath, passphrase)
			if err != nil {
				if err == storage.ErrInvalidPassphrase {
					statusLabel.SetText("Invalid passphrase for the selected profile.")
				} else {
					statusLabel.SetText("Failed to open profile.")
				}
				dialog.ShowError(err, win)
				return
			}

			showMainView(store, profilePath)
		})

		gate := container.NewVBox(
			widget.NewLabel("Profile Required"),
			widget.NewLabel("You must select or create a profile file and enter its passphrase before using Trusty."),
			widget.NewForm(
				widget.NewFormItem("Profile File", profilePathEntry),
				widget.NewFormItem("Passphrase", passphraseEntry),
			),
			container.NewGridWithColumns(2, selectButton, createButton),
			unlockButton,
			statusLabel,
		)

		root.Objects = []fyne.CanvasObject{container.NewPadded(container.NewCenter(gate))}
		root.Refresh()
	}

	showProfileGate("")
	return root
}

func createMainView(win fyne.Window, store *storage.Store, profileName string, onSwitchProfile func()) fyne.CanvasObject {
	contactNames := binding.NewStringList()
	updateContactList := func() {
		names := make([]string, 0, len(store.Contacts))
		for name := range store.Contacts {
			names = append(names, name)
		}
		sort.Strings(names)
		if err := contactNames.Set(names); err != nil {
			log.Printf("failed to update contact list: %v", err)
		}
	}
	updateContactList()

	rightPanel := container.NewMax(
		container.NewCenter(widget.NewLabel("Select a contact to start messaging")),
	)

	contactList := widget.NewListWithData(
		contactNames,
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			val, err := item.(binding.String).Get()
			if err != nil {
				val = ""
			}
			obj.(*widget.Label).SetText(val)
		},
	)

	contactList.OnSelected = func(id widget.ListItemID) {
		name, _ := contactNames.GetValue(id)
		contact := store.Contacts[name]
		if contact != nil {
			messagingView := createMessagingView(win, contact, store)
			rightPanel.Objects = []fyne.CanvasObject{messagingView}
			rightPanel.Refresh()
		}
	}

	// Top Toolbar
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.ContentAddIcon(), func() {
			ShowAddUserWizard(win, store, func() {
				updateContactList()
				contactList.UnselectAll()
			})
		}),
		widget.NewToolbarAction(theme.SettingsIcon(), func() {
			if onSwitchProfile == nil {
				return
			}
			onSwitchProfile()
		}),
	)

	split := container.NewHSplit(contactList, rightPanel)
	split.Offset = 0.3
	header := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("Profile: %s", profileName)),
		widget.NewSeparator(),
	)
	return container.NewBorder(container.NewVBox(header, toolbar), nil, nil, nil, split)
}
