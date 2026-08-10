package rest

import (
	"fmt"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/entity"
)

type treeNode struct {
	label string
	item  entity.CollectionItem
}

type CollectionTree struct {
	widget.BaseWidget

	mu       sync.RWMutex
	childIDs map[string][]string
	nodes    map[string]treeNode

	tree *widget.Tree
}

func NewCollectionTree(onSelectItem func(item entity.CollectionItem)) *CollectionTree {
	ct := &CollectionTree{
		childIDs: map[string][]string{"": {}},
		nodes:    map[string]treeNode{},
	}
	ct.ExtendBaseWidget(ct)

	ct.tree = widget.NewTree(
		func(uid widget.TreeNodeID) []widget.TreeNodeID {
			ct.mu.RLock()
			defer ct.mu.RUnlock()
			return ct.childIDs[uid]
		},
		func(uid widget.TreeNodeID) bool {
			if uid == "" {
				return true
			}
			ct.mu.RLock()
			defer ct.mu.RUnlock()
			n, ok := ct.nodes[uid]
			if !ok {
				return false
			}
			return len(n.item.Item) > 0
		},
		func(branch bool) fyne.CanvasObject {
			icon := widget.NewIcon(theme.DocumentIcon())
			icon.Resize(fyne.NewSize(theme.IconInlineSize(), theme.IconInlineSize()))
			label := widget.NewLabel("")
			label.Truncation = fyne.TextTruncateEllipsis
			return container.NewBorder(nil, nil, icon, nil, label)
		},
		func(uid widget.TreeNodeID, branch bool, obj fyne.CanvasObject) {
			ct.mu.RLock()
			n := ct.nodes[uid]
			ct.mu.RUnlock()

			c := obj.(*fyne.Container)
			c.Objects[1].(*widget.Icon).SetResource(
				map[bool]fyne.Resource{
					true:  theme.FolderIcon(),
					false: theme.DocumentIcon(),
				}[branch],
			)
			c.Objects[0].(*widget.Label).SetText(n.label)
		},
	)

	ct.tree.OnSelected = func(uid widget.TreeNodeID) {
		ct.mu.RLock()
		n, ok := ct.nodes[uid]
		ct.mu.RUnlock()

		if !ok || onSelectItem == nil {
			return
		}
		onSelectItem(n.item)
	}

	return ct
}

func (ct *CollectionTree) SetCollections(collections []entity.Collection) {
	childIDs := map[string][]string{"": {}}
	nodes := map[string]treeNode{}

	for _, col := range collections {
		colUID := "col:" + col.Id
		childIDs[""] = append(childIDs[""], colUID)
		nodes[colUID] = treeNode{
			label: col.Name,
			item: entity.CollectionItem{
				Id:   col.Id,
				Name: col.Name,
				Item: col.Items,
			},
		}
		fillItems(col.Items, colUID, childIDs, nodes)
	}

	ct.mu.Lock()
	ct.childIDs = childIDs
	ct.nodes = nodes
	ct.mu.Unlock()

	ct.tree.Refresh()

	ct.mu.RLock()
	roots := ct.childIDs[""]
	ct.mu.RUnlock()
	for _, uid := range roots {
		ct.tree.OpenBranch(uid)
	}
}

func (ct *CollectionTree) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(ct.tree)
}

func fillItems(
	collectionItems []entity.CollectionItem,
	parentUID string,
	childIDs map[string][]string,
	nodes map[string]treeNode,
) {
	for _, item := range collectionItems {
		uid := "item:" + item.Id
		childIDs[parentUID] = append(childIDs[parentUID], uid)

		label := item.Name
		if item.Request != nil {
			label = fmt.Sprintf("[%s] %s", item.Request.Method, item.Name)
		}

		nodes[uid] = treeNode{label: label, item: item}

		if len(item.Item) > 0 {
			fillItems(item.Item, uid, childIDs, nodes)
		}
	}
}
