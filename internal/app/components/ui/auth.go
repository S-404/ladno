package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
)

type AuthPanelOptions struct {
	// AllowInherited включает тип Inherited (для folder/request).
	AllowInherited bool
	// OnChange вызывается при изменении auth (без отдельной кнопки Save).
	OnChange func(auth entity.Auth)
	// OnSave если задан — показывает кнопку Save (legacy).
	OnSave func(auth entity.Auth)
}

type AuthPanel struct {
	fyne.CanvasObject
	Set func(auth entity.Auth)
	Get func() entity.Auth
}

func NewAuthPanel(opts AuthPanelOptions) *AuthPanel {
	options := []string{
		string(constants.AuthTypeNoAuth),
		string(constants.AuthTypeBasic),
		string(constants.AuthTypeJWT),
	}
	if opts.AllowInherited {
		options = append([]string{string(constants.AuthTypeInherited)}, options...)
	}

	var applying bool
	var getAuth func() entity.Auth
	notify := func() {
		if applying || opts.OnChange == nil || getAuth == nil {
			return
		}
		opts.OnChange(getAuth())
	}

	authSelect := widget.NewSelect(options, func(string) { notify() })
	authSelect.SetSelected(string(constants.AuthTypeNoAuth))

	authData := NewKVTable(nil, func([]KVRow) { notify() })

	getAuth = func() entity.Auth {
		rows := authData.GetRows()
		vars := make([]entity.Variable, 0, len(rows))
		for _, r := range rows {
			if r.Key == "" {
				continue
			}
			vars = append(vars, entity.Variable{Key: r.Key, Value: r.Value, Type: "string"})
		}
		t := constants.AuthType(authSelect.Selected)
		if !opts.AllowInherited && t == constants.AuthTypeInherited {
			t = constants.AuthTypeNoAuth
		}
		return entity.Auth{Type: t, Data: vars}
	}

	form := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Type", authSelect),
		),
		widget.NewLabel("Auth data"),
		authData,
	)

	var root fyne.CanvasObject = container.NewPadded(container.NewVScroll(form))
	if opts.OnSave != nil && opts.OnChange == nil {
		saveBtn := widget.NewButton("Save", func() {
			opts.OnSave(getAuth())
		})
		saveBtn.Importance = widget.HighImportance
		root = container.NewBorder(nil, container.NewPadded(saveBtn), nil, nil,
			container.NewPadded(container.NewVScroll(form)),
		)
	}

	p := &AuthPanel{CanvasObject: root}
	p.Set = func(auth entity.Auth) {
		applying = true
		t := auth.Type
		if t == "" {
			if opts.AllowInherited {
				t = constants.AuthTypeInherited
			} else {
				t = constants.AuthTypeNoAuth
			}
		}
		if !opts.AllowInherited && t == constants.AuthTypeInherited {
			t = constants.AuthTypeNoAuth
		}
		authSelect.SetSelected(string(t))
		rows := make([]KVRow, 0, len(auth.Data))
		for _, d := range auth.Data {
			rows = append(rows, KVRow{Enabled: true, Key: d.Key, Value: d.Value})
		}
		authData.SetRows(rows)
		applying = false
	}
	p.Get = getAuth
	return p
}
