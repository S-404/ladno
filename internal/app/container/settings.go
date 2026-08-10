package container

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/entity/shared"
	"github.com/s-404/ladno/internal/app/store"
)

func SettingsContainer(app *shared.App) fyne.CanvasObject {
	tabs := container.NewAppTabs(
		container.NewTabItem("Workspace", workspaceSettingsTab(app)),
		container.NewTabItem("General", generalSettingsTab(app)),
	)
	return tabs
}

func workspaceSettingsTab(app *shared.App) fyne.CanvasObject {
	wsStore := app.Store.Workspace

	empty := container.NewCenter(widget.NewLabel("Select a workspace in the header"))

	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Workspace name")
	connEntry := widget.NewMultiLineEntry()
	connEntry.SetPlaceHolder("Connection config (optional)")
	connEntry.SetMinRowsVisible(4)

	idLabel := widget.NewLabel("")
	idLabel.TextStyle = fyne.TextStyle{Italic: true}

	status := widget.NewLabel("")
	status.TextStyle = fyne.TextStyle{Italic: true}

	saveBtn := widget.NewButton("Save", func() {
		if !wsStore.UpdateSelectedWorkspace(nameEntry.Text, connEntry.Text) {
			status.SetText("No workspace selected")
			return
		}
		status.SetText("Workspace saved")
	})
	saveBtn.Importance = widget.HighImportance

	form := container.NewPadded(container.NewVBox(
		widget.NewLabelWithStyle("Selected workspace", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("ID", idLabel),
			widget.NewFormItem("Name", nameEntry),
			widget.NewFormItem("Connection config", connEntry),
		),
		saveBtn,
		status,
	))

	stack := container.NewStack(empty, form)
	showEmpty := func(on bool) {
		if on {
			empty.Show()
			form.Hide()
		} else {
			empty.Hide()
			form.Show()
		}
		stack.Refresh()
	}
	showEmpty(true)

	load := func() {
		ws := wsStore.GetSelectedWorkspace()
		if ws == nil {
			showEmpty(true)
			status.SetText("")
			return
		}
		idLabel.SetText(ws.Id)
		nameEntry.SetText(ws.Name)
		connEntry.SetText(ws.ConnectionConfig)
		status.SetText("")
		showEmpty(false)
	}

	wsStore.GetItem().AddListener(binding.NewDataListener(load))
	load()

	return stack
}

func generalSettingsTab(app *shared.App) fyne.CanvasObject {
	settings := app.Store.Settings
	logStore := app.Store.Log
	natsStore := app.Store.Nats

	limitEntry := widget.NewEntry()
	limitEntry.SetText(strconv.Itoa(settings.GetMessageLimit()))
	limitEntry.SetPlaceHolder("1000")

	limitHint := widget.NewLabel("Applies to Logs and NATS Messages (10–100000). Only the newest entries are kept.")
	limitHint.Wrapping = fyne.TextWrapWord
	limitHint.TextStyle = fyne.TextStyle{Italic: true}

	themeSelect := widget.NewSelect([]string{"Light", "Dark"}, nil)
	switch settings.GetTheme() {
	case store.ThemeLight:
		themeSelect.SetSelected("Light")
	default:
		themeSelect.SetSelected("Dark")
	}
	themeSelect.OnChanged = func(v string) {
		name := store.ThemeDark
		if v == "Light" {
			name = store.ThemeLight
		}
		settings.SetTheme(name)
	}

	status := widget.NewLabel("")
	status.TextStyle = fyne.TextStyle{Italic: true}

	saveBtn := widget.NewButton("Save", func() {
		n, err := strconv.Atoi(limitEntry.Text)
		if err != nil {
			status.SetText("Enter a valid number")
			return
		}
		applied := settings.SetMessageLimit(n)
		limitEntry.SetText(strconv.Itoa(applied))
		logStore.TrimToLimit()
		natsStore.TrimMessagesToLimit()

		themeName := store.ThemeDark
		if themeSelect.Selected == "Light" {
			themeName = store.ThemeLight
		}
		settings.SetTheme(themeName)

		status.SetText(fmt.Sprintf("Saved: theme=%s, keep last %d messages", themeName, applied))
	})
	saveBtn.Importance = widget.HighImportance

	form := container.NewVBox(
		widget.NewLabelWithStyle("Appearance", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Theme", themeSelect),
		),
		widget.NewLabelWithStyle("History", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Message limit", limitEntry),
		),
		limitHint,
		saveBtn,
		status,
	)

	return container.NewPadded(form)
}
