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
	OnChange       func(auth entity.Auth)
}

type AuthPanel struct {
	fyne.CanvasObject
	Set func(auth entity.Auth)
	Get func() entity.Auth
}

type authTypeOption struct {
	value constants.AuthType
	label string
}

func NewAuthPanel(opts AuthPanelOptions) *AuthPanel {
	typeOpts := []authTypeOption{
		{constants.AuthTypeNoAuth, "No Auth"},
		{constants.AuthTypeBasic, "Basic Auth"},
		{constants.AuthTypeBearer, "Token"},
		{constants.AuthTypeAPIKey, "API Key"},
	}
	if opts.AllowInherited {
		typeOpts = append([]authTypeOption{{constants.AuthTypeInherited, "Inherited"}}, typeOpts...)
	}
	labels := make([]string, len(typeOpts))
	labelToType := map[string]constants.AuthType{}
	typeToLabel := map[constants.AuthType]string{}
	for i, o := range typeOpts {
		labels[i] = o.label
		labelToType[o.label] = o.value
		typeToLabel[o.value] = o.label
	}

	var applying bool
	var getAuth func() entity.Auth
	notify := func() {
		if applying || opts.OnChange == nil || getAuth == nil {
			return
		}
		opts.OnChange(getAuth())
	}

	authSelect := widget.NewSelect(labels, nil)

	username := NewEntry()
	username.SetPlaceHolder("Username")
	password := NewEntry()
	password.SetPlaceHolder("Password")
	password.Password = true

	token := NewEntry()
	token.SetPlaceHolder("Token")
	token.Password = true
	tokenPrefix := NewEntry()
	tokenPrefix.SetPlaceHolder("Bearer")
	tokenPrefix.SetText(constants.AuthDefaultTokenPrefix)

	apiKey := NewEntry()
	apiKey.SetPlaceHolder("Key (header/body name)")
	apiValue := NewEntry()
	apiValue.SetPlaceHolder("Value")
	apiValue.Password = true
	apiAddTo := widget.NewRadioGroup([]string{"Header", "Body"}, nil)
	apiAddTo.Horizontal = true
	apiAddTo.SetSelected("Header")

	basicForm := widget.NewForm(
		widget.NewFormItem("Username", ListWheelField(username)),
		widget.NewFormItem("Password", ListWheelField(password)),
	)
	bearerForm := widget.NewForm(
		widget.NewFormItem("Prefix", ListWheelField(tokenPrefix)),
		widget.NewFormItem("Token", ListWheelField(token)),
	)
	apiForm := widget.NewForm(
		widget.NewFormItem("Key", ListWheelField(apiKey)),
		widget.NewFormItem("Value", ListWheelField(apiValue)),
		widget.NewFormItem("Add to", apiAddTo),
	)
	inheritedHint := widget.NewLabel("Uses auth from the parent folder or collection.")
	inheritedHint.TextStyle = fyne.TextStyle{Italic: true}
	noAuthHint := widget.NewLabel("No authentication headers or body fields are added.")
	noAuthHint.TextStyle = fyne.TextStyle{Italic: true}

	details := container.NewStack(inheritedHint, noAuthHint, basicForm, bearerForm, apiForm)
	showDetails := func(t constants.AuthType) {
		inheritedHint.Hide()
		noAuthHint.Hide()
		basicForm.Hide()
		bearerForm.Hide()
		apiForm.Hide()
		switch constants.NormalizeAuthType(t) {
		case constants.AuthTypeInherited:
			inheritedHint.Show()
		case constants.AuthTypeBasic:
			basicForm.Show()
		case constants.AuthTypeBearer:
			bearerForm.Show()
		case constants.AuthTypeAPIKey:
			apiForm.Show()
		default:
			noAuthHint.Show()
		}
		details.Refresh()
	}

	selectedType := func() constants.AuthType {
		t, ok := labelToType[authSelect.Selected]
		if !ok {
			if opts.AllowInherited {
				return constants.AuthTypeInherited
			}
			return constants.AuthTypeNoAuth
		}
		return t
	}

	getAuth = func() entity.Auth {
		t := selectedType()
		if !opts.AllowInherited && t == constants.AuthTypeInherited {
			t = constants.AuthTypeNoAuth
		}
		t = constants.NormalizeAuthType(t)
		var data []entity.Variable
		switch t {
		case constants.AuthTypeBasic:
			data = []entity.Variable{
				{Key: constants.AuthDataUsername, Value: username.Text},
				{Key: constants.AuthDataPassword, Value: password.Text},
			}
		case constants.AuthTypeBearer:
			data = []entity.Variable{
				{Key: constants.AuthDataPrefix, Value: tokenPrefix.Text},
				{Key: constants.AuthDataToken, Value: token.Text},
			}
		case constants.AuthTypeAPIKey:
			addTo := constants.AuthAddToHeader
			if apiAddTo.Selected == "Body" {
				addTo = constants.AuthAddToBody
			}
			data = []entity.Variable{
				{Key: constants.AuthDataKey, Value: apiKey.Text},
				{Key: constants.AuthDataValue, Value: apiValue.Text},
				{Key: constants.AuthDataAddTo, Value: addTo},
			}
		}
		return entity.Auth{Type: t, Data: data}
	}

	authSelect.OnChanged = func(string) {
		showDetails(selectedType())
		notify()
	}
	username.OnChanged = func(string) { notify() }
	password.OnChanged = func(string) { notify() }
	token.OnChanged = func(string) { notify() }
	tokenPrefix.OnChanged = func(string) { notify() }
	apiKey.OnChanged = func(string) { notify() }
	apiValue.OnChanged = func(string) { notify() }
	apiAddTo.OnChanged = func(string) { notify() }

	if opts.AllowInherited {
		authSelect.SetSelected(typeToLabel[constants.AuthTypeInherited])
	} else {
		authSelect.SetSelected(typeToLabel[constants.AuthTypeNoAuth])
	}
	showDetails(selectedType())

	form := container.NewVBox(
		widget.NewForm(widget.NewFormItem("Type", authSelect)),
		details,
	)
	root := container.NewBorder(nil, nil, nil, nil, container.NewPadded(NewListVScroll(form)))

	p := &AuthPanel{CanvasObject: root}
	p.Set = func(auth entity.Auth) {
		applying = true
		t := constants.NormalizeAuthType(auth.Type)
		if auth.Type == "" {
			if opts.AllowInherited {
				t = constants.AuthTypeInherited
			} else {
				t = constants.AuthTypeNoAuth
			}
		}
		if !opts.AllowInherited && t == constants.AuthTypeInherited {
			t = constants.AuthTypeNoAuth
		}
		if label, ok := typeToLabel[t]; ok {
			authSelect.SetSelected(label)
		} else {
			authSelect.SetSelected(typeToLabel[constants.AuthTypeNoAuth])
			t = constants.AuthTypeNoAuth
		}
		username.SetText(entity.AuthVar(auth.Data, constants.AuthDataUsername))
		password.SetText(entity.AuthVar(auth.Data, constants.AuthDataPassword))
		token.SetText(entity.AuthVar(auth.Data, constants.AuthDataToken))
		if entity.AuthHasVar(auth.Data, constants.AuthDataPrefix) {
			tokenPrefix.SetText(entity.AuthVar(auth.Data, constants.AuthDataPrefix))
		} else {
			tokenPrefix.SetText(constants.AuthDefaultTokenPrefix)
		}
		apiKey.SetText(entity.AuthVar(auth.Data, constants.AuthDataKey))
		apiValue.SetText(entity.AuthVar(auth.Data, constants.AuthDataValue))
		if entity.AuthVar(auth.Data, constants.AuthDataAddTo) == constants.AuthAddToBody {
			apiAddTo.SetSelected("Body")
		} else {
			apiAddTo.SetSelected("Header")
		}
		showDetails(t)
		applying = false
	}
	p.Get = getAuth
	return p
}
