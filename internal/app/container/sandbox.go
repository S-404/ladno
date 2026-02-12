package container

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity/shared"
)

func SandboxContainer(app *shared.App) fyne.CanvasObject {
	dataList := app.Store.Bar.GetItems()
	list := widget.NewListWithData(*dataList,
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			person := app.Store.Bar.GetDataItem(item)
			if person != nil {
				obj.(*widget.Label).SetText(fmt.Sprintf("%s (%s)", person.Name, person.Email))
			}
		})

	list.OnSelected = func(id widget.ListItemID) {
		selected := app.Store.Bar.GetItemByIndex(id)
		fmt.Println("SandboxContainer list.OnSelected", selected.Name, selected.Id)
	}

	loader := ui.NewLoader(app.Store.Bar.GetIsFetching())

	fetchData := func() {
		app.Store.Bar.Fetch()
	}

	fetchDataAsync := func() {
		app.Store.Bar.FetchAsync()
	}

	clearData := func() {
		app.Store.Bar.Clear()
	}

	fetchBtn := widget.NewButton("Fetch data", fetchData)
	fetchAsyncBtn := widget.NewButton("Fetch data async", fetchDataAsync)

	clearBtn := widget.NewButton("clear", clearData)

	controlPanel := container.NewVBox(fetchBtn, fetchAsyncBtn, clearBtn)

	listBoxWithLoader := container.NewStack(container.NewVBox(loader), list)
	listBox := container.NewHScroll(listBoxWithLoader)

	return container.NewHSplit(
		controlPanel,
		listBox,
	)
}
