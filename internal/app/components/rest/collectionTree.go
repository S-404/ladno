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

type CollectionTree struct {
	widget.BaseWidget

	mu       sync.RWMutex
	childIDs map[string][]string
	nodes    map[string]string // uid -> label
	branches map[string]bool   // uid -> isBranch

	tree *widget.Tree
}

func NewCollectionTree() *CollectionTree {
	ct := &CollectionTree{
		childIDs: map[string][]string{"": {}},
		nodes:    map[string]string{},
		branches: map[string]bool{},
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
			return ct.branches[uid]
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
			label := ct.nodes[uid]
			ct.mu.RUnlock()

			c := obj.(*fyne.Container)
			c.Objects[1].(*widget.Icon).SetResource(
				map[bool]fyne.Resource{
					true:  theme.FolderIcon(),
					false: theme.DocumentIcon(),
				}[branch],
			)
			c.Objects[0].(*widget.Label).SetText(label)
		},
	)

	return ct
}

func (ct *CollectionTree) SetCollections(collections []entity.Collection) {
	childIDs := map[string][]string{"": {}}
	nodes := map[string]string{}
	branches := map[string]bool{}

	for _, col := range collections {
		colUID := "col:" + col.Id
		childIDs[""] = append(childIDs[""], colUID)
		nodes[colUID] = col.Name
		branches[colUID] = len(col.Item) > 0
		fillItems(col.Item, colUID, childIDs, nodes, branches)
	}

	ct.mu.Lock()
	ct.childIDs = childIDs
	ct.nodes = nodes
	ct.branches = branches
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
	items []entity.CollectionItem,
	parentUID string,
	childIDs map[string][]string,
	nodes map[string]string,
	branches map[string]bool,
) {
	for _, item := range items {
		uid := "item:" + item.Id
		childIDs[parentUID] = append(childIDs[parentUID], uid)

		label := item.Name
		if item.Request != nil {
			label = fmt.Sprintf("[%s] %s", item.Request.Method, item.Name)
		}

		nodes[uid] = label
		branches[uid] = len(item.Item) > 0

		if len(item.Item) > 0 {
			fillItems(item.Item, uid, childIDs, nodes, branches)
		}
	}
}
