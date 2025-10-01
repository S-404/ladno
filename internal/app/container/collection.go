package container

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/goose/internal/app/components/rest"
	"github.com/s-404/goose/internal/app/components/ui"
	"github.com/s-404/goose/internal/app/entity"
	"github.com/s-404/goose/internal/app/entity/shared"
)

func CollectionContainer(app *shared.App) fyne.CanvasObject {
	wsStore := app.Store.Workspace
	wsItem := wsStore.GetItem()

	// Контейнер для дерева
	treeContainer := container.NewStack()

	updateTree := func(collections []entity.Collection) {
		treeContainer.Objects = nil
		tree := rest.NewCollectionTree(collections, nil)
		treeContainer.Add(tree)
		treeContainer.Refresh()
	}

	wsItem.AddListener(binding.NewDataListener(func() {
		workspace := wsStore.GetWorkspaceDataItem(wsItem)

		if workspace != nil {
			fmt.Printf("Workspace loaded: %s, ID: %s\n", workspace.Name, workspace.Id)
			fmt.Printf("Collections count: %d\n", len(workspace.Collection))

			if len(workspace.Collection) > 0 {
				updateTree(workspace.Collection)
			} else {
				fmt.Println("No collections found in workspace")
				updateTree([]entity.Collection{})
			}
		} else {
			fmt.Println("Workspace is nil")
			updateTree([]entity.Collection{})
		}
	}))

	loader := ui.NewLoader(app.Store.Workspace.GetIsFetching())
	collectionWithLoader := container.NewStack(container.NewVBox(loader), treeContainer)
	collection := container.NewHScroll(collectionWithLoader)

	return container.NewBorder(widget.NewToolbar(), nil, nil, nil, collection)
}
