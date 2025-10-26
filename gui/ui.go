package gui

import (
	"log"
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"trusty/storage"
)

func MakeUI(win fyne.Window) fyne.CanvasObject {
	store, err := storage.NewStore()
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}

	contactNames := binding.NewStringList()
	updateContactList := func() {
		names := make([]string, 0, len(store.Contacts))
		for name := range store.Contacts {
			names = append(names, name)
		}
		sort.Strings(names)
		contactNames.Set(names)
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
			item.AddListener(binding.NewDataListener(func() {
				val, _ := item.(binding.String).Get()
				obj.(*widget.Label).SetText(val)
			}))
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
	)

	split := container.NewHSplit(contactList, rightPanel)
	split.Offset = 0.3
	return container.NewBorder(toolbar, nil, nil, nil, split)
}
