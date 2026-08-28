package scripttab

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
)

const (
	actionSetLabel   = "set"
	actionClearLabel = "clear"
)

type preRow struct {
	Action string
	EnvKey string
	Value  string
}

type postRow struct {
	Action string
	EnvKey string
	Path   string
}

type ScriptView struct {
	Object     fyne.CanvasObject
	Get        func() entity.Event
	Set        func(ev entity.Event)
	SetError   func(msg string)
	SetEnvKeys func(keys []string)
}

// NewScriptView — вкладка Script с вертикальными табами Pre-request / Post-request.
func NewScriptView(onChange func(entity.Event)) *ScriptView {
	var applying bool
	var rebuilding bool
	var preRows []preRow
	var postRows []postRow
	var envKeys []string
	var preBox, postBox *fyne.Container
	var rebuildPre, rebuildPost func()

	errLabel := widget.NewLabel("")
	errLabel.Importance = widget.DangerImportance
	errLabel.Wrapping = fyne.TextWrapWord
	errLabel.Hide()

	notify := func() {
		if applying || rebuilding || onChange == nil {
			return
		}
		onChange(scriptToEvent(preRows, postRows))
	}

	newActionSelect := func(current string, onPick func(string)) *widget.Select {
		sel := widget.NewSelect([]string{actionSetLabel, actionClearLabel}, func(v string) {
			if applying || rebuilding {
				return
			}
			onPick(v)
		})
		sel.SetSelected(normalizeActionLabel(current))
		return sel
	}

	rebuildPre = func() {
		rebuilding = true
		addBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
			preRows = append(preRows, preRow{Action: actionSetLabel})
			rebuildPre()
			notify()
		})
		addBtn.Importance = widget.LowImportance

		header := container.NewBorder(nil, nil, nil, addBtn,
			container.NewGridWithColumns(3,
				widget.NewLabelWithStyle("Action", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabelWithStyle("Env key", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabelWithStyle("Value", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			),
		)

		objs := make([]fyne.CanvasObject, 0, len(preRows)+1)
		objs = append(objs, header)
		for i := range preRows {
			isClearAction := normalizeActionLabel(preRows[i].Action) == actionClearLabel

			actionSel := newActionSelect(preRows[i].Action, func(v string) {
				preRows[i].Action = v
				rebuildPre()
				notify()
			})

			keySel := widget.NewSelectEntry(append([]string(nil), envKeys...))
			keySel.SetPlaceHolder("env key")
			keySel.SetText(preRows[i].EnvKey)
			keySel.OnChanged = func(v string) {
				if applying || rebuilding {
					return
				}
				preRows[i].EnvKey = v
				notify()
			}

			valEntry := ui.NewEntry()
			valEntry.SetPlaceHolder("value")
			valEntry.SetText(preRows[i].Value)
			valEntry.OnChanged = func(v string) {
				if applying || rebuilding {
					return
				}
				preRows[i].Value = v
				notify()
			}

			var valueCol fyne.CanvasObject = valEntry
			if isClearAction {
				valueCol = widget.NewLabel("")
			}

			del := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
				if applying || rebuilding {
					return
				}
				preRows = append(preRows[:i], preRows[i+1:]...)
				rebuildPre()
				notify()
			})
			del.Importance = widget.LowImportance

			objs = append(objs, container.NewBorder(nil, nil, nil, del,
				container.NewGridWithColumns(3, actionSel, keySel, valueCol),
			))
		}
		preBox.Objects = objs
		preBox.Refresh()
		rebuilding = false
	}

	rebuildPost = func() {
		rebuilding = true
		addBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
			postRows = append(postRows, postRow{Action: actionSetLabel})
			rebuildPost()
			notify()
		})
		addBtn.Importance = widget.LowImportance

		header := container.NewBorder(nil, nil, nil, addBtn,
			container.NewGridWithColumns(3,
				widget.NewLabelWithStyle("Action", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabelWithStyle("Env key", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabelWithStyle("Body path", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			),
		)

		objs := make([]fyne.CanvasObject, 0, len(postRows)+1)
		objs = append(objs, header)
		for i := range postRows {
			isClearAction := normalizeActionLabel(postRows[i].Action) == actionClearLabel

			actionSel := newActionSelect(postRows[i].Action, func(v string) {
				postRows[i].Action = v
				rebuildPost()
				notify()
			})

			keySel := widget.NewSelectEntry(append([]string(nil), envKeys...))
			keySel.SetPlaceHolder("env key")
			keySel.SetText(postRows[i].EnvKey)
			keySel.OnChanged = func(v string) {
				if applying || rebuilding {
					return
				}
				postRows[i].EnvKey = v
				notify()
			}

			pathEntry := ui.NewEntry()
			pathEntry.SetPlaceHolder("data.token or user?.id")
			pathEntry.SetText(postRows[i].Path)
			pathEntry.OnChanged = func(v string) {
				if applying || rebuilding {
					return
				}
				postRows[i].Path = v
				notify()
			}

			var pathCol fyne.CanvasObject = pathEntry
			if isClearAction {
				pathCol = widget.NewLabel("")
			}

			del := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
				if applying || rebuilding {
					return
				}
				postRows = append(postRows[:i], postRows[i+1:]...)
				rebuildPost()
				notify()
			})
			del.Importance = widget.LowImportance

			objs = append(objs, container.NewBorder(nil, nil, nil, del,
				container.NewGridWithColumns(3, actionSel, keySel, pathCol),
			))
		}
		postBox.Objects = objs
		postBox.Refresh()
		rebuilding = false
	}

	preBox = container.NewVBox()
	postBox = container.NewVBox()

	prePanel := container.NewBorder(
		nil, nil, nil, nil,
		container.NewVScroll(preBox),
	)
	postPanel := container.NewBorder(
		nil, nil, nil, nil,
		container.NewVScroll(postBox),
	)

	tabs := container.NewAppTabs(
		container.NewTabItem("Pre-request", prePanel),
		container.NewTabItem("Post-request", postPanel),
	)
	tabs.SetTabLocation(container.TabLocationLeading)

	root := container.NewBorder(errLabel, nil, nil, nil, tabs)

	rebuildAll := func() {
		rebuildPre()
		rebuildPost()
	}

	v := &ScriptView{Object: root}
	v.Get = func() entity.Event {
		return scriptToEvent(preRows, postRows)
	}
	v.Set = func(ev entity.Event) {
		applying = true
		preRows = eventToPreRows(ev)
		postRows = eventToPostRows(ev)
		rebuildAll()
		applying = false
	}
	v.SetError = func(msg string) {
		msg = strings.TrimSpace(msg)
		if msg == "" {
			errLabel.SetText("")
			errLabel.Hide()
		} else {
			errLabel.SetText(msg)
			errLabel.Show()
		}
		errLabel.Refresh()
	}
	v.SetEnvKeys = func(keys []string) {
		envKeys = append([]string(nil), keys...)
		applying = true
		rebuildAll()
		applying = false
	}
	rebuildAll()
	return v
}

func normalizeActionLabel(a string) string {
	if constants.EnvEventAction(a) == constants.EnvEventActionClear || a == actionClearLabel {
		return actionClearLabel
	}
	return actionSetLabel
}

func actionFromLabel(a string) constants.EnvEventAction {
	if normalizeActionLabel(a) == actionClearLabel {
		return constants.EnvEventActionClear
	}
	return constants.EnvEventActionSet
}

func scriptToEvent(pre []preRow, post []postRow) entity.Event {
	preOut := make([]entity.PreRequestEnvEvent, 0, len(pre))
	for _, r := range pre {
		key := strings.TrimSpace(r.EnvKey)
		action := actionFromLabel(r.Action)
		if key == "" && action == constants.EnvEventActionSet && strings.TrimSpace(r.Value) == "" {
			continue
		}
		if key == "" && action == constants.EnvEventActionClear {
			continue
		}
		ev := entity.PreRequestEnvEvent{
			EnvKey: key,
			Action: action,
		}
		if action == constants.EnvEventActionSet {
			ev.Value = r.Value
		}
		preOut = append(preOut, ev)
	}
	postOut := make([]entity.PostRequestEnvEvent, 0, len(post))
	for _, r := range post {
		key := strings.TrimSpace(r.EnvKey)
		action := actionFromLabel(r.Action)
		path := strings.TrimSpace(r.Path)
		if key == "" && action == constants.EnvEventActionSet && path == "" {
			continue
		}
		if key == "" && action == constants.EnvEventActionClear {
			continue
		}
		ev := entity.PostRequestEnvEvent{
			EnvKey: key,
			Action: action,
		}
		if action == constants.EnvEventActionSet {
			ev.JSONPath = path
		}
		postOut = append(postOut, ev)
	}
	return entity.Event{PreRequest: preOut, PostRequest: postOut}
}

func eventToPreRows(ev entity.Event) []preRow {
	rows := make([]preRow, 0, len(ev.PreRequest))
	for _, e := range ev.PreRequest {
		rows = append(rows, preRow{
			Action: normalizeActionLabel(string(e.Action)),
			EnvKey: e.EnvKey,
			Value:  e.Value,
		})
	}
	return rows
}

func eventToPostRows(ev entity.Event) []postRow {
	rows := make([]postRow, 0, len(ev.PostRequest))
	for _, e := range ev.PostRequest {
		rows = append(rows, postRow{
			Action: normalizeActionLabel(string(e.Action)),
			EnvKey: e.EnvKey,
			Path:   e.JSONPath,
		})
	}
	return rows
}
