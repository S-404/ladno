package collection

import (
	"fmt"
	"image/color"
	"log"
	"strings"
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

type ContextHandler struct {
	OnEmpty      func(pos fyne.Position)
	OnCollection func(col entity.Collection, pos fyne.Position)
	OnFolder     func(col entity.Collection, item entity.CollectionItem, path []string, pos fyne.Position)
	OnRequest    func(col entity.Collection, item entity.CollectionItem, path []string, pos fyne.Position)
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
	source    []entity.Collection
	filter    string
	handler   SelectHandler
	context   ContextHandler
	tree      *widget.Tree
}

var colorConnected = color.NRGBA{R: 0x34, G: 0xA8, B: 0x53, A: 0xFF}

func NewTree(handler SelectHandler, context ContextHandler) *Tree {
	ct := &Tree{
		childIDs:  map[string][]string{"": {}},
		nodes:     map[string]treeNode{},
		cols:      map[string]entity.Collection{},
		connected: map[string]bool{},
		handler:   handler,
		context:   context,
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
			return len(n.item.Item) > 0 || n.kind == nodeCollection || n.kind == nodeFolder
		},
		func(branch bool) fyne.CanvasObject {
			return newTreeRow(ct.onRowPrimary, ct.onRowSecondary)
		},
		func(uid widget.TreeNodeID, branch bool, obj fyne.CanvasObject) {
			ct.mu.RLock()
			n := ct.nodes[uid]
			connected := ct.connected[n.collectionID]
			ct.mu.RUnlock()

			row := obj.(*treeRow)
			row.SetUID(uid)

			res := theme.DocumentIcon()
			if n.kind == nodeCollection || n.kind == nodeFolder {
				res = theme.FolderIcon()
			}
			row.icon.SetResource(res)
			row.label.SetText(n.label)

			if n.kind == nodeCollection && connected {
				row.status.FillColor = colorConnected
				row.status.StrokeColor = colorConnected
				row.status.Show()
			} else {
				row.status.Hide()
			}
			row.status.Refresh()
		},
	)

	ct.tree.OnSelected = func(uid widget.TreeNodeID) {
		ct.fireSelect(uid)
	}

	return ct
}

func (ct *Tree) onRowPrimary(uid string) {
	ct.tree.Select(uid)
}

func (ct *Tree) fireSelect(uid widget.TreeNodeID) {
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

func (ct *Tree) onRowSecondary(uid string, e *fyne.PointEvent) {
	ct.mu.RLock()
	n, ok := ct.nodes[uid]
	col, colOK := ct.cols[n.collectionID]
	ct.mu.RUnlock()
	if !ok || !colOK {
		return
	}

	ct.tree.Select(uid)
	pos := e.AbsolutePosition
	switch n.kind {
	case nodeCollection:
		if ct.context.OnCollection != nil {
			ct.context.OnCollection(col, pos)
		}
	case nodeFolder:
		if ct.context.OnFolder != nil {
			ct.context.OnFolder(col, n.item, append([]string{}, n.path...), pos)
		}
	case nodeRequest:
		if ct.context.OnRequest != nil {
			ct.context.OnRequest(col, n.item, append([]string{}, n.path...), pos)
		}
	}
}

// TappedSecondary — ПКМ по пустому месту дерева.
func (ct *Tree) TappedSecondary(e *fyne.PointEvent) {
	if ct.context.OnEmpty != nil {
		ct.context.OnEmpty(e.AbsolutePosition)
	}
}

func (ct *Tree) SelectUID(uid string) {
	if uid == "" {
		return
	}
	ct.tree.Select(uid)
}

func (ct *Tree) CollectionUID(collectionID string) string {
	return "col:" + collectionID
}

func (ct *Tree) ItemUID(itemID string) string {
	return "item:" + itemID
}

func (ct *Tree) OpenUID(uid string) {
	if uid == "" {
		return
	}
	ct.tree.OpenBranch(uid)
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

func (ct *Tree) SetFilter(query string) {
	ct.mu.Lock()
	ct.filter = strings.TrimSpace(query)
	ct.mu.Unlock()
	ct.rebuild()
}

func (ct *Tree) SetCollections(collections []entity.Collection) {
	ct.mu.Lock()
	if collections == nil {
		ct.source = nil
	} else {
		ct.source = append([]entity.Collection(nil), collections...)
	}
	ct.mu.Unlock()
	ct.rebuild()
}

func (ct *Tree) rebuild() {
	ct.mu.RLock()
	source := append([]entity.Collection(nil), ct.source...)
	filter := ct.filter
	ct.mu.RUnlock()

	visible := filterCollections(source, filter)

	childIDs := map[string][]string{"": {}}
	nodes := map[string]treeNode{}
	cols := map[string]entity.Collection{}

	for _, col := range visible {
		col = normalizeCol(col)
		cols[col.Id] = col
		colUID := "col:" + col.Id
		childIDs[""] = append(childIDs[""], colUID)
		childIDs[colUID] = []string{}
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

	log.Printf("[collections] Tree.rebuild filter=%q collections=%d nodes=%d roots=%v",
		filter, len(visible), len(nodes), childIDs[""])

	ct.tree.Refresh()

	ct.mu.RLock()
	roots := append([]string{}, ct.childIDs[""]...)
	ct.mu.RUnlock()
	for _, uid := range roots {
		if filter != "" {
			ct.openAllUnder(uid)
			continue
		}
		ct.tree.OpenBranch(uid)
	}
}

func (ct *Tree) openAllUnder(uid string) {
	ct.tree.OpenBranch(uid)
	ct.mu.RLock()
	children := append([]string{}, ct.childIDs[uid]...)
	ct.mu.RUnlock()
	for _, child := range children {
		ct.openAllUnder(child)
	}
}

func (ct *Tree) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(ct.tree)
}

func normalizeCol(col entity.Collection) entity.Collection {
	col.Type = constants.NormalizeCollectionType(col.Type)
	return col
}

func filterCollections(collections []entity.Collection, query string) []entity.Collection {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return collections
	}
	out := make([]entity.Collection, 0, len(collections))
	for _, col := range collections {
		col = normalizeCol(col)
		colMatch := strings.Contains(strings.ToLower(col.Name), q) ||
			strings.Contains(strings.ToLower(string(col.Type)), q)
		items, ok := filterItems(col.Items, q, colMatch)
		if !colMatch && !ok {
			continue
		}
		col.Items = items
		out = append(out, col)
	}
	return out
}

func filterItems(items []entity.CollectionItem, q string, keepAll bool) ([]entity.CollectionItem, bool) {
	if keepAll {
		return items, true
	}
	out := make([]entity.CollectionItem, 0, len(items))
	any := false
	for _, item := range items {
		nameMatch := strings.Contains(strings.ToLower(item.Name), q)
		if item.Request != nil {
			if nameMatch || strings.Contains(strings.ToLower(string(item.Request.Method)), q) {
				out = append(out, item)
				any = true
				continue
			}
			if item.Request.Nats != nil && strings.Contains(strings.ToLower(item.Request.Nats.Subject), q) {
				out = append(out, item)
				any = true
				continue
			}
			continue
		}
		// folder
		children, childOK := filterItems(item.Item, q, false)
		if nameMatch {
			out = append(out, item)
			any = true
			continue
		}
		if childOK {
			item.Item = children
			out = append(out, item)
			any = true
		}
	}
	return out, any
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

		if kind == nodeFolder {
			if _, ok := childIDs[uid]; !ok {
				childIDs[uid] = []string{}
			}
		}

		if len(item.Item) > 0 {
			fillItems(col, item.Item, uid, path, childIDs, nodes)
		}
	}
}

// treeRow — строка дерева; Tapped обязателен, иначе SecondaryTappable перехватывает ЛКМ.
type treeRow struct {
	widget.BaseWidget
	uid         string
	onPrimary   func(uid string)
	onSecondary func(uid string, e *fyne.PointEvent)

	icon   *widget.Icon
	status *canvas.Circle
	dot    *ui.MinSizeBox
	label  *widget.Label
	root   *fyne.Container
}

func newTreeRow(onPrimary func(uid string), onSecondary func(uid string, e *fyne.PointEvent)) *treeRow {
	icon := widget.NewIcon(theme.DocumentIcon())
	status := canvas.NewCircle(color.Transparent)
	status.Hide()
	dot := ui.NewMinSizeBox(fyne.NewSize(8, 8), status)
	label := widget.NewLabel("")
	label.Truncation = fyne.TextTruncateEllipsis
	left := container.NewHBox(icon, container.NewCenter(dot))
	root := container.NewBorder(nil, nil, left, nil, label)

	r := &treeRow{
		onPrimary:   onPrimary,
		onSecondary: onSecondary,
		icon:        icon,
		status:      status,
		dot:         dot,
		label:       label,
		root:        root,
	}
	r.ExtendBaseWidget(r)
	return r
}

func (r *treeRow) SetUID(uid string) {
	r.uid = uid
}

func (r *treeRow) Tapped(_ *fyne.PointEvent) {
	if r.onPrimary != nil && r.uid != "" {
		r.onPrimary(r.uid)
	}
}

func (r *treeRow) TappedSecondary(e *fyne.PointEvent) {
	if r.onSecondary != nil && r.uid != "" {
		r.onSecondary(r.uid, e)
	}
}

func (r *treeRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(r.root)
}

func (r *treeRow) MinSize() fyne.Size {
	return r.root.MinSize()
}
