package container

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/shared"
	"github.com/s-404/ladno/internal/app/store"
	"github.com/s-404/ladno/internal/buildinfo"
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
	settings := app.Store.Settings

	empty := container.NewCenter(widget.NewLabel("Select a workspace in the header"))

	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Workspace name")
	connEntry := widget.NewMultiLineEntry()
	connEntry.SetPlaceHolder("Connection config (optional)")
	connEntry.SetMinRowsVisible(4)
	nestingEntry := widget.NewEntry()
	nestingEntry.SetPlaceHolder(strconv.Itoa(entity.DefaultFolderNestingLimit))
	nestingHint := widget.NewLabel("Max folder depth in collection trees. -1 means unlimited. Default is 5.")
	nestingHint.Wrapping = fyne.TextWrapWord
	nestingHint.TextStyle = fyne.TextStyle{Italic: true}

	idLabel := widget.NewLabel("")
	idLabel.TextStyle = fyne.TextStyle{Italic: true}

	status := widget.NewLabel("")
	status.TextStyle = fyne.TextStyle{Italic: true}

	saveBtn := widget.NewButton("Save", func() {
		depth, err := strconv.Atoi(nestingEntry.Text)
		if err != nil {
			status.SetText("Enter a valid folder nesting limit")
			return
		}
		depth = entity.ClampFolderNestingLimit(depth)
		if !wsStore.UpdateSelectedWorkspace(nameEntry.Text, connEntry.Text, depth) {
			status.SetText("No workspace selected")
			return
		}
		nestingEntry.SetText(strconv.Itoa(depth))
		status.SetText(fmt.Sprintf("Workspace saved, folder depth %d", depth))
	})
	saveBtn.Importance = widget.HighImportance

	deleteBtn := widget.NewButton("Delete", func() {
		ws := wsStore.GetSelectedWorkspace()
		if ws == nil {
			status.SetText("No workspace selected")
			return
		}
		dialog.ShowConfirm("Delete workspace", fmt.Sprintf("Delete %q?", ws.Name), func(ok bool) {
			if !ok {
				return
			}
			id := ws.Id
			wsStore.Delete(id, func(err error) {
				if err != nil {
					status.SetText(fmt.Sprintf("Delete failed: %v", err))
					return
				}
				if settings.GetLastWorkspaceID() == id {
					settings.SetLastWorkspaceID("")
				}
				status.SetText("Workspace deleted")
			})
		}, app.Window)
	})
	deleteBtn.Importance = widget.DangerImportance

	form := container.NewPadded(container.NewVBox(
		widget.NewLabelWithStyle("Selected workspace", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("ID", idLabel),
			widget.NewFormItem("Name", nameEntry),
			widget.NewFormItem("Connection config", connEntry),
			widget.NewFormItem("Folder nesting limit", nestingEntry),
		),
		nestingHint,
		container.NewHBox(saveBtn, deleteBtn),
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
		nestingEntry.SetText(strconv.Itoa(ws.GetFolderNestingLimit()))
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
	kafkaStore := app.Store.Kafka
	wsConnStore := app.Store.Ws
	sioConnStore := app.Store.SocketIO

	limitEntry := widget.NewEntry()
	limitEntry.SetText(strconv.Itoa(settings.GetMessageLimit()))
	limitEntry.SetPlaceHolder("1000")
	limitHint := widget.NewLabel("Applies to Logs, NATS, Kafka, WebSocket and Socket.IO Messages (10–100000). Only the newest entries are kept.")
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

	fontLabels := []string{"Small", "Medium", "Large", "Extra large"}
	fontSelect := widget.NewSelect(fontLabels, nil)
	switch settings.GetFontSize() {
	case store.FontSizeSmall:
		fontSelect.SetSelected("Small")
	case store.FontSizeLarge:
		fontSelect.SetSelected("Large")
	case store.FontSizeXLarge:
		fontSelect.SetSelected("Extra large")
	default:
		fontSelect.SetSelected("Medium")
	}
	fontSizeFromLabel := func(v string) string {
		switch v {
		case "Small":
			return store.FontSizeSmall
		case "Large":
			return store.FontSizeLarge
		case "Extra large":
			return store.FontSizeXLarge
		default:
			return store.FontSizeMedium
		}
	}
	fontSelect.OnChanged = func(v string) {
		settings.SetFontSize(fontSizeFromLabel(v))
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
		kafkaStore.TrimMessagesToLimit()
		wsConnStore.TrimMessagesToLimit()
		sioConnStore.TrimMessagesToLimit()

		themeName := store.ThemeDark
		if themeSelect.Selected == "Light" {
			themeName = store.ThemeLight
		}
		settings.SetTheme(themeName)
		fontName := fontSizeFromLabel(fontSelect.Selected)
		settings.SetFontSize(fontName)

		status.SetText(fmt.Sprintf("Saved: theme=%s, font=%s, keep last %d messages", themeName, fontName, applied))
	})
	saveBtn.Importance = widget.HighImportance

	form := container.NewVBox(
		widget.NewLabelWithStyle("About", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Version", widget.NewLabel(buildinfo.Version())),
		),
		widget.NewLabelWithStyle("Appearance", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Theme", themeSelect),
			widget.NewFormItem("Font size", fontSelect),
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
