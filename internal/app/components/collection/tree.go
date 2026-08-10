package collection

import (
	"fmt"
	"image/color"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/ui"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
)

type SelectHandler struct {
	OnCollection func(col entity.Collection)
	OnFolder     func(col entity.Collection, item entity.CollectionItem, path []string)
	OnRequest    func(col entity.Collection, item entity.CollectionItem, path []string)
}

type treeNodeKind int

const (
	nodeCollection treeNodeKind = iota
	nodeFolder
	nodeRequest
)

type treeNode struct {
	kind         treeNodeKind
	label        string
	collectionID string
	colType      constants.CollectionType
	item         entity.CollectionItem
	path         []string
}

type Tree struct {
	widget.BaseWidget

	mu        sync.RWMutex
	childIDs  map[string][]string
	nodes     map[string]treeNode
	cols      map[string]entity.Collection
	connected map[string]bool
	handler   SelectHandler
	tree      *widget.Tree
}

var colorConnected = color.NRGBA{R: 0x34, G: 0xA8, B: 0x53, A: 0xFF}

func NewTree(handler SelectHandler) *Tree {
	ct := &Tree{
		childIDs:  map[string][]string{"": {}},
		nodes:     map[string]treeNode{},
		cols:      map[string]entity.Collection{},
		connected: map[string]bool{},
		handler:   handler,
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
			status := canvas.NewCircle(color.Transparent)
			status.Hide()
			dot := ui.NewMinSizeBox(fyne.NewSize(8, 8), status)
			label := widget.NewLabel("")
			label.Truncation = fyne.TextTruncateEllipsis
			left := container.NewHBox(icon, container.NewCenter(dot))
			return container.NewBorder(nil, nil, left, nil, label)
		},
		func(uid widget.TreeNodeID, branch bool, obj fyne.CanvasObject) {
			ct.mu.RLock()
			n := ct.nodes[uid]
			connected := ct.connected[n.collectionID]
			ct.mu.RUnlock()

			c := obj.(*fyne.Container)
			left := c.Objects[1].(*fyne.Container)
			icon := left.Objects[0].(*widget.Icon)
			dotWrap := left.Objects[1].(*fyne.Container)
			dotBox := dotWrap.Objects[0].(*ui.MinSizeBox)
			status := dotBox.Content.(*canvas.Circle)
			label := c.Objects[0].(*widget.Label)

			res := theme.DocumentIcon()
			if n.kind == nodeCollection || n.kind == nodeFolder {
				res = theme.FolderIcon()
			}
			icon.SetResource(res)
			label.SetText(n.label)

			if n.kind == nodeCollection && connected {
				status.FillColor = colorConnected
				status.StrokeColor = colorConnected
				status.Show()
			} else {
				status.Hide()
			}
			status.Refresh()
		},
	)

	ct.tree.OnSelected = func(uid widget.TreeNodeID) {
		ct.mu.RLock()
		n, ok := ct.nodes[uid]
		col, colOK := ct.cols[n.collectionID]
		ct.mu.RUnlock()
		if !ok || !colOK {
			return
		}

		switch n.kind {
		case nodeCollection:
			if ct.handler.OnCollection != nil {
				ct.handler.OnCollection(col)
			}
		case nodeFolder:
			if ct.handler.OnFolder != nil {
				ct.handler.OnFolder(col, n.item, append([]string{}, n.path...))
			}
		case nodeRequest:
			if ct.handler.OnRequest != nil {
				ct.handler.OnRequest(col, n.item, append([]string{}, n.path...))
			}
		}
	}

	return ct
}

func (ct *Tree) SetConnected(ids map[string]bool) {
	ct.mu.Lock()
	if ids == nil {
		ct.connected = map[string]bool{}
	} else {
		ct.connected = ids
	}
	ct.mu.Unlock()
	ct.tree.Refresh()
}

func (ct *Tree) SetCollections(collections []entity.Collection) {
	childIDs := map[string][]string{"": {}}
	nodes := map[string]treeNode{}
	cols := map[string]entity.Collection{}

	for _, col := range collections {
		col = normalizeCol(col)
		cols[col.Id] = col
		colUID := "col:" + col.Id
		childIDs[""] = append(childIDs[""], colUID)
		nodes[colUID] = treeNode{
			kind:         nodeCollection,
			label:        fmt.Sprintf("%s · %s", col.Name, col.Type),
			collectionID: col.Id,
			colType:      col.Type,
			item: entity.CollectionItem{
				Id:   col.Id,
				Name: col.Name,
				Item: col.Items,
			},
			path: nil,
		}
		fillItems(col, col.Items, colUID, nil, childIDs, nodes)
	}

	ct.mu.Lock()
	ct.childIDs = childIDs
	ct.nodes = nodes
	ct.cols = cols
	ct.mu.Unlock()

	ct.tree.Refresh()

	ct.mu.RLock()
	roots := ct.childIDs[""]
	ct.mu.RUnlock()
	for _, uid := range roots {
		ct.tree.OpenBranch(uid)
	}
}

func (ct *Tree) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(ct.tree)
}

func normalizeCol(col entity.Collection) entity.Collection {
	col.Type = constants.NormalizeCollectionType(col.Type)
	return col
}

func fillItems(
	col entity.Collection,
	collectionItems []entity.CollectionItem,
	parentUID string,
	parentPath []string,
	childIDs map[string][]string,
	nodes map[string]treeNode,
) {
	for _, item := range collectionItems {
		uid := "item:" + item.Id
		childIDs[parentUID] = append(childIDs[parentUID], uid)
		path := append(append([]string{}, parentPath...), item.Id)

		kind := nodeFolder
		label := item.Name
		if item.Request != nil {
			kind = nodeRequest
			if col.Type == constants.CollectionTypeREST {
				label = fmt.Sprintf("[%s] %s", item.Request.Method, item.Name)
			}
		}

		nodes[uid] = treeNode{
			kind:         kind,
			label:        label,
			collectionID: col.Id,
			colType:      col.Type,
			item:         item,
			path:         path,
		}

		if len(item.Item) > 0 {
			fillItems(col, item.Item, uid, path, childIDs, nodes)
		}
	}
}
