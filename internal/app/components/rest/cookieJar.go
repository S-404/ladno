package rest

import (
	"fmt"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity"
)

// CookieJarView — cookies по доменам с редактированием raw-строки.
type CookieJarView struct {
	Object  fyne.CanvasObject
	Refresh func()
}

// NewCookieJarView builds a scrollable cookie jar grouped by domain.
func NewCookieJarView(
	win fyne.Window,
	listFn func() []entity.Cookie,
	domainsFn func() []string,
	onDelete func(c entity.Cookie),
	onClear func(),
	onUpdate func(c entity.Cookie),
	onReplace func(prev, next entity.Cookie),
	onAddDomain func(domain string),
	onDeleteDomain func(domain string),
	onAddCookie func(domain string),
) *CookieJarView {
	empty := widget.NewLabel("No cookies stored")
	empty.Importance = widget.LowImportance
	empty.Alignment = fyne.TextAlignCenter

	listBox := container.NewVBox(empty)
	scroll := container.NewVScroll(listBox)
	opened := map[string]bool{}

	var refresh func()
	refresh = func() {
		cookies := listFn()
		domains := domainsFn()
		if len(domains) == 0 && len(cookies) == 0 {
			listBox.Objects = []fyne.CanvasObject{empty}
			listBox.Refresh()
			return
		}

		byDomain := map[string][]entity.Cookie{}
		for _, d := range domains {
			byDomain[d] = nil
		}
		for _, c := range cookies {
			d := c.Domain
			if d == "" {
				d = "(unknown)"
			}
			byDomain[d] = append(byDomain[d], c)
		}
		domainList := make([]string, 0, len(byDomain))
		for d := range byDomain {
			domainList = append(domainList, d)
		}
		sort.Strings(domainList)

		sections := make([]fyne.CanvasObject, 0, len(domainList))
		for _, domain := range domainList {
			list := byDomain[domain]
			sort.Slice(list, func(i, j int) bool {
				if list[i].Name != list[j].Name {
					return list[i].Name < list[j].Name
				}
				return list[i].Path < list[j].Path
			})
			sections = append(sections, buildDomainSection(
				domain, list, opened[domain],
				func(open bool) { opened[domain] = open },
				onDelete, onUpdate, onReplace, onAddCookie, onDeleteDomain, refresh,
			))
		}
		listBox.Objects = sections
		listBox.Refresh()
	}

	addDomainBtn := widget.NewButtonWithIcon("Add domain", theme.ContentAddIcon(), func() {
		entry := ui.NewEntry()
		entry.SetPlaceHolder("example.com")
		dialog.ShowForm("Add domain", "Add", "Cancel", []*widget.FormItem{
			widget.NewFormItem("Domain", entry),
		}, func(ok bool) {
			if !ok {
				return
			}
			d := strings.TrimSpace(entry.Text)
			if d == "" || onAddDomain == nil {
				return
			}
			onAddDomain(d)
			opened[strings.ToLower(d)] = true
			refresh()
		}, win)
	})
	addDomainBtn.Importance = widget.LowImportance

	clearBtn := widget.NewButton("Clear all", func() {
		if onClear != nil {
			onClear()
		}
		refresh()
	})
	clearBtn.Importance = widget.DangerImportance

	toolbar := container.NewBorder(nil, nil,
		widget.NewLabel("Stored cookies"),
		container.NewHBox(addDomainBtn, clearBtn),
		nil,
	)
	root := container.NewBorder(toolbar, nil, nil, nil, scroll)

	refresh()
	return &CookieJarView{Object: root, Refresh: refresh}
}

func buildDomainSection(
	domain string,
	cookies []entity.Cookie,
	open bool,
	setOpen func(bool),
	onDelete func(c entity.Cookie),
	onUpdate func(c entity.Cookie),
	onReplace func(prev, next entity.Cookie),
	onAddCookie func(domain string),
	onDeleteDomain func(domain string),
	refresh func(),
) fyne.CanvasObject {
	title := widget.NewLabel(fmt.Sprintf("%s (%d)", domain, len(cookies)))
	title.TextStyle = fyne.TextStyle{Bold: true}

	body := container.NewVBox()
	isOpen := open

	toggleIcon := theme.MenuDropDownIcon()
	if !isOpen {
		toggleIcon = theme.NavigateNextIcon()
	}
	toggle := widget.NewButtonWithIcon("", toggleIcon, nil)
	toggle.Importance = widget.LowImportance

	delDomain := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		if onDeleteDomain != nil {
			onDeleteDomain(domain)
		}
		if refresh != nil {
			refresh()
		}
	})
	delDomain.Importance = widget.LowImportance

	header := container.NewBorder(nil, nil, toggle, delDomain, title)

	rebuildBody := func() {
		rows := make([]fyne.CanvasObject, 0, len(cookies)*2+1)
		for i, c := range cookies {
			rows = append(rows, buildCookieEditor(c, onDelete, onUpdate, onReplace, refresh))
			if i+1 < len(cookies) {
				rows = append(rows, widget.NewSeparator())
			}
		}
		addCookieBtn := widget.NewButtonWithIcon("Add cookie", theme.ContentAddIcon(), func() {
			setOpen(true)
			if onAddCookie != nil {
				onAddCookie(domain)
			}
			if refresh != nil {
				refresh()
			}
		})
		addCookieBtn.Importance = widget.LowImportance
		if len(rows) > 0 {
			rows = append(rows, widget.NewSeparator())
		}
		rows = append(rows, addCookieBtn)
		body.Objects = rows
		body.Refresh()
	}
	rebuildBody()

	applyOpen := func() {
		setOpen(isOpen)
		if isOpen {
			body.Show()
			toggle.SetIcon(theme.MenuDropDownIcon())
		} else {
			body.Hide()
			toggle.SetIcon(theme.NavigateNextIcon())
		}
		toggle.Refresh()
	}
	toggle.OnTapped = func() {
		isOpen = !isOpen
		applyOpen()
	}
	applyOpen()

	return container.NewPadded(container.NewVBox(header, body, widget.NewSeparator()))
}

func buildCookieEditor(
	cookie entity.Cookie,
	onDelete func(c entity.Cookie),
	onUpdate func(c entity.Cookie),
	onReplace func(prev, next entity.Cookie),
	refresh func(),
) fyne.CanvasObject {
	identity := cookie

	title := widget.NewLabel(cookie.Name)
	title.TextStyle = fyne.TextStyle{Bold: true}

	del := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		if onDelete != nil {
			onDelete(identity)
		}
		if refresh != nil {
			refresh()
		}
	})
	del.Importance = widget.LowImportance

	editor := ui.NewMultiLineEntry()
	editor.SetText(entity.FormatCookieRaw(cookie))
	editor.SetMinRowsVisible(3)
	editor.SetPlaceHolder("name=value; Path=/; …")
	editor.OnChanged = func(raw string) {
		next, ok := entity.ParseCookieRaw(raw, identity.Domain, identity.HostOnly)
		if !ok {
			return
		}
		title.SetText(next.Name)
		title.Refresh()
		if next.CookieKey() != identity.CookieKey() {
			if onReplace != nil {
				onReplace(identity, next)
			}
		} else if onUpdate != nil {
			onUpdate(next)
		}
		identity = next
	}

	header := container.NewBorder(nil, nil, nil, del, title)
	return container.NewBorder(header, nil, nil, nil, editor)
}
