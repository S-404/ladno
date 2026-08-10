package container

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/rest"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/shared"
)

func CollectionContainer(app *shared.App) fyne.CanvasObject {
	wsStore := app.Store.Workspace
	wsItem := wsStore.GetItem()

	onSelectCollectionItem := func(item entity.CollectionItem) {
		fmt.Printf("selected collection item %s\n", item.Id)
	}

	collectionTree := rest.NewCollectionTree(onSelectCollectionItem)

	wsItem.AddListener(binding.NewDataListener(func() {
		workspace := wsStore.GetWorkspaceDataItem(wsItem)
		if workspace == nil {
			collectionTree.SetCollections(nil)
			return
		}
		collectionTree.SetCollections(workspace.Collections)
	}))

	return container.NewBorder(
		widget.NewToolbar(),
		nil, nil, nil,
		container.NewScroll(collectionTree),
	)
}
