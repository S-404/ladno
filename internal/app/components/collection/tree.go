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

type ReorderHandler struct {
	OnCollection func(collectionID string, steps int)
	OnItem       func(collectionID, itemID string, steps int)
	OnRelocate   func(fromCollectionID, itemID, toCollectionID, toParentItemID string, toIndex int)
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

// DirtyResolver reports unsaved edits for tree rows.
type DirtyResolver struct {
	IsDirty      func(collectionID, itemID string, isCollection, isFolder bool) bool
	ResolveLabel func(collectionID, itemID string, isCollection, isFolder bool, fallback string) string
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
	reorder   ReorderHandler
	dirty     DirtyResolver
	tree      *widget.Tree

	rowsByUID map[string]*treeRow

	dragActive     bool
	dragSourceUID  string
	dropInsertIdx  int // index among destination siblings after removing source; -1 = none
	dropLineUID    string
	dropLineBefore bool
	dropIntoUID    string
	dropToColID    string
	dropToParentID string // "" = collection root
	dropCrossTree  bool   // relocate (possibly cross-parent) vs collection steps
	ghost          *widget.PopUp
	ghostLabel     *widget.Label
	ghostIcon      *widget.Icon
}

var colorConnected = color.NRGBA{R: 0x34, G: 0xA8, B: 0x53, A: 0xFF}
var colorDirty = color.NRGBA{R: 0xFF, G: 0x98, B: 0x00, A: 0xFF}
var colorInsert = color.NRGBA{R: 0x1E, G: 0x88, B: 0xE5, A: 0xFF}

const dragThreshold = float32(6)

func NewTree(handler SelectHandler, context ContextHandler, reorder ReorderHandler) *Tree {
	ct := &Tree{
		childIDs:  map[string][]string{"": {}},
		nodes:     map[string]treeNode{},
		cols:      map[string]entity.Collection{},
		connected: map[string]bool{},
		rowsByUID: map[string]*treeRow{},
		handler:   handler,
		context:   context,
		reorder:   reorder,
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
			return newTreeRow(ct)
		},
		func(uid widget.TreeNodeID, branch bool, obj fyne.CanvasObject) {
			ct.mu.RLock()
			n := ct.nodes[uid]
			connected := ct.connected[n.collectionID]
			filterOn := ct.filter != ""
			dragSource := ct.dragSourceUID
			dropUID := ct.dropLineUID
			dropBefore := ct.dropLineBefore
			dropInto := ct.dropIntoUID
			ct.mu.RUnlock()

			row := obj.(*treeRow)
			row.SetUID(uid)
			row.SetReorderEnabled(!filterOn)

			res := theme.DocumentIcon()
			if n.kind == nodeCollection {
				res = theme.ListIcon()
			}
			if n.kind == nodeFolder {
				res = theme.FolderIcon()
			}

			row.icon.SetResource(res)
			label := n.label
			isCol := n.kind == nodeCollection
			isFolder := n.kind == nodeFolder
			itemID := ""
			if !isCol {
				itemID = n.item.Id
			}
			if ct.dirty.ResolveLabel != nil {
				label = ct.dirty.ResolveLabel(n.collectionID, itemID, isCol, isFolder, n.label)
			}
			row.label.SetText(label)
			row.SetDraggingSource(uid != "" && uid == dragSource)
			if uid != "" && uid == dropInto {
				row.SetDropIndicator(dropIntoIndicator)
			} else if uid != "" && uid == dropUID {
				if dropBefore {
					row.SetDropIndicator(dropBeforeIndicator)
				} else {
					row.SetDropIndicator(dropAfterIndicator)
				}
			} else {
				row.SetDropIndicator(dropNone)
			}

			dirty := ct.dirty.IsDirty != nil && ct.dirty.IsDirty(n.collectionID, itemID, isCol, isFolder)
			if dirty {
				row.status.FillColor = colorDirty
				row.status.StrokeColor = colorDirty
				row.status.Show()
			} else if n.kind == nodeCollection && connected {
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

func (ct *Tree) SetDirtyResolver(r DirtyResolver) {
	ct.mu.Lock()
	ct.dirty = r
	ct.mu.Unlock()
	if ct.tree != nil {
		ct.tree.Refresh()
	}
}

func (ct *Tree) RefreshDirty() {
	if ct.tree != nil {
		ct.tree.Refresh()
	}
}

func (ct *Tree) registerRow(uid string, row *treeRow) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	if uid == "" || row == nil {
		return
	}
	ct.rowsByUID[uid] = row
}

func (ct *Tree) unregisterRow(uid string, row *treeRow) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	if uid == "" {
		return
	}
	if cur, ok := ct.rowsByUID[uid]; ok && cur == row {
		delete(ct.rowsByUID, uid)
	}
}

func (ct *Tree) parentUIDOf(uid string) string {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	n, ok := ct.nodes[uid]
	if !ok {
		return ""
	}
	if n.kind == nodeCollection {
		return ""
	}
	if len(n.path) <= 1 {
		return "col:" + n.collectionID
	}
	return "item:" + n.path[len(n.path)-2]
}

func (ct *Tree) siblingsOf(uid string) []string {
	parent := ct.parentUIDOf(uid)
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return append([]string{}, ct.childIDs[parent]...)
}

func (ct *Tree) beginDrag(uid string) {
	ct.mu.Lock()
	if ct.filter != "" || uid == "" {
		ct.mu.Unlock()
		return
	}
	n, ok := ct.nodes[uid]
	if !ok {
		ct.mu.Unlock()
		return
	}
	ct.dragActive = true
	ct.dragSourceUID = uid
	ct.clearDropLocked()
	label := n.label
	kind := n.kind
	ct.mu.Unlock()

	res := theme.DocumentIcon()
	if kind == nodeCollection {
		res = theme.ListIcon()
	} else if kind == nodeFolder {
		res = theme.FolderIcon()
	}
	ct.showGhost(label, res)
	ct.tree.Refresh()
}

func (ct *Tree) clearDropLocked() {
	ct.dropInsertIdx = -1
	ct.dropLineUID = ""
	ct.dropLineBefore = false
	ct.dropIntoUID = ""
	ct.dropToColID = ""
	ct.dropToParentID = ""
	ct.dropCrossTree = false
}

type dragDropTarget struct {
	valid      bool
	insertIdx  int
	lineUID    string
	lineBefore bool
	intoUID    string
	toColID    string
	toParentID string
	crossTree  bool
}

func (ct *Tree) updateDrag(abs fyne.Position) {
	ct.mu.RLock()
	active := ct.dragActive
	source := ct.dragSourceUID
	ct.mu.RUnlock()
	if !active || source == "" {
		return
	}
	ct.moveGhost(abs)

	target := ct.resolveDropTarget(source, abs)

	ct.mu.Lock()
	same := ct.dropInsertIdx == target.insertIdx &&
		ct.dropLineUID == target.lineUID &&
		ct.dropLineBefore == target.lineBefore &&
		ct.dropIntoUID == target.intoUID &&
		ct.dropToColID == target.toColID &&
		ct.dropToParentID == target.toParentID
	if target.valid {
		ct.dropInsertIdx = target.insertIdx
		ct.dropLineUID = target.lineUID
		ct.dropLineBefore = target.lineBefore
		ct.dropIntoUID = target.intoUID
		ct.dropToColID = target.toColID
		ct.dropToParentID = target.toParentID
		ct.dropCrossTree = target.crossTree
	} else {
		ct.clearDropLocked()
	}
	ct.mu.Unlock()
	if !same {
		ct.tree.Refresh()
	}
}

func (ct *Tree) resolveDropTarget(source string, abs fyne.Position) dragDropTarget {
	ct.mu.RLock()
	src, ok := ct.nodes[source]
	ct.mu.RUnlock()
	if !ok {
		return dragDropTarget{}
	}

	if src.kind == nodeCollection {
		return ct.resolveCollectionDrop(source, abs)
	}
	return ct.resolveItemDrop(source, src, abs)
}

func (ct *Tree) resolveCollectionDrop(source string, abs fyne.Position) dragDropTarget {
	siblings := ct.siblingsOf(source)
	if len(siblings) == 0 {
		return dragDropTarget{}
	}
	drv := fyne.CurrentApp().Driver()
	insertIdx := len(siblings)
	lineUID := ""
	lineBefore := false

	for i, uid := range siblings {
		ct.mu.RLock()
		row := ct.rowsByUID[uid]
		ct.mu.RUnlock()
		if row == nil {
			continue
		}
		pos := drv.AbsolutePositionForObject(row)
		h := row.Size().Height
		if h < 1 {
			h = 28
		}
		if abs.Y < pos.Y+h/2 {
			insertIdx = i
			lineUID = uid
			lineBefore = true
			break
		}
		lineUID = uid
		lineBefore = false
		insertIdx = i + 1
	}

	srcIdx := indexOfUID(siblings, source)
	adjusted := insertIdx
	if srcIdx >= 0 && srcIdx < insertIdx {
		adjusted--
	}
	if srcIdx >= 0 && adjusted == srcIdx {
		return dragDropTarget{}
	}
	return dragDropTarget{
		valid:      true,
		insertIdx:  adjusted,
		lineUID:    lineUID,
		lineBefore: lineBefore,
		crossTree:  false,
	}
}

func (ct *Tree) resolveItemDrop(source string, src treeNode, abs fyne.Position) dragDropTarget {
	hitUID, row, pos, h := ct.rowAtY(abs.Y)
	if hitUID == "" || row == nil {
		return dragDropTarget{}
	}

	ct.mu.RLock()
	hit, ok := ct.nodes[hitUID]
	ct.mu.RUnlock()
	if !ok {
		return dragDropTarget{}
	}

	srcType := constants.NormalizeCollectionType(src.colType)
	hitType := constants.NormalizeCollectionType(hit.colType)
	if srcType != hitType {
		return dragDropTarget{}
	}
	if hitUID == source {
		return dragDropTarget{}
	}
	if src.kind == nodeFolder && ct.isUnderFolder(source, hitUID) {
		return dragDropTarget{}
	}

	zone := dropZone(abs.Y, pos.Y, h)

	// Drop into collection.
	if hit.kind == nodeCollection {
		return ct.dropIntoContainer(source, hitUID, hit.collectionID, "", hitUID)
	}

	// Folder: middle = into; bottom of open folder = into at start (above children);
	// top/bottom of closed folder = before/after among siblings.
	if hit.kind == nodeFolder {
		open := ct.tree.IsBranchOpen(hitUID)
		switch zone {
		case zoneMiddle:
			return ct.dropIntoContainer(source, hitUID, hit.collectionID, hit.item.Id, hitUID)
		case zoneBottom:
			if open {
				return ct.dropIntoContainerAt(source, hitUID, hit.collectionID, hit.item.Id, hitUID, 0)
			}
		case zoneTop:
			// before folder among its siblings — fall through
		}
	}

	// Before / after among the hit item's siblings (same-type tree).
	parentUID := ct.parentUIDOf(hitUID)
	siblings := ct.childIDsCopy(parentUID)
	hitIdx := indexOfUID(siblings, hitUID)
	if hitIdx < 0 {
		return dragDropTarget{}
	}
	insertIdx := hitIdx
	lineBefore := true
	if hit.kind == nodeFolder {
		if zone == zoneBottom {
			insertIdx = hitIdx + 1
			lineBefore = false
		}
	} else if abs.Y >= pos.Y+h/2 {
		insertIdx = hitIdx + 1
		lineBefore = false
	}

	srcIdx := indexOfUID(siblings, source)
	adjusted := insertIdx
	if srcIdx >= 0 && srcIdx < insertIdx {
		adjusted--
	}
	if srcIdx >= 0 && adjusted == srcIdx {
		return dragDropTarget{}
	}

	toColID, toParentID := ct.parseParent(parentUID, hit.collectionID)
	return dragDropTarget{
		valid:      true,
		insertIdx:  adjusted,
		lineUID:    hitUID,
		lineBefore: lineBefore,
		toColID:    toColID,
		toParentID: toParentID,
		crossTree:  true,
	}
}

func (ct *Tree) dropIntoContainer(source, intoUID, colID, parentItemID, containerUID string) dragDropTarget {
	ct.mu.RLock()
	children := append([]string{}, ct.childIDs[containerUID]...)
	ct.mu.RUnlock()
	toIndex := len(children)
	srcIdx := indexOfUID(children, source)
	if srcIdx >= 0 {
		toIndex--
		if srcIdx == toIndex {
			return dragDropTarget{}
		}
	}
	return dragDropTarget{
		valid:      true,
		insertIdx:  toIndex,
		intoUID:    intoUID,
		toColID:    colID,
		toParentID: parentItemID,
		crossTree:  true,
	}
}

func (ct *Tree) dropIntoContainerAt(source, intoUID, colID, parentItemID, containerUID string, at int) dragDropTarget {
	ct.mu.RLock()
	children := append([]string{}, ct.childIDs[containerUID]...)
	ct.mu.RUnlock()
	srcIdx := indexOfUID(children, source)
	toIndex := at
	if toIndex < 0 {
		toIndex = 0
	}
	if toIndex > len(children) {
		toIndex = len(children)
	}
	if srcIdx >= 0 {
		if srcIdx < toIndex {
			toIndex--
		}
		if srcIdx == toIndex {
			return dragDropTarget{}
		}
		// after removal max index is len-1
		if toIndex > len(children)-1 {
			toIndex = len(children) - 1
		}
	}
	return dragDropTarget{
		valid:      true,
		insertIdx:  toIndex,
		intoUID:    intoUID,
		toColID:    colID,
		toParentID: parentItemID,
		crossTree:  true,
	}
}

type yZone int

const (
	zoneTop yZone = iota
	zoneMiddle
	zoneBottom
)

func dropZone(y, rowY, rowH float32) yZone {
	if rowH < 1 {
		rowH = 28
	}
	rel := (y - rowY) / rowH
	if rel < 0.28 {
		return zoneTop
	}
	if rel > 0.72 {
		return zoneBottom
	}
	return zoneMiddle
}

func (ct *Tree) rowAtY(y float32) (uid string, row *treeRow, pos fyne.Position, h float32) {
	drv := fyne.CurrentApp().Driver()
	ct.mu.RLock()
	rows := make([]*treeRow, 0, len(ct.rowsByUID))
	uids := make([]string, 0, len(ct.rowsByUID))
	for id, r := range ct.rowsByUID {
		uids = append(uids, id)
		rows = append(rows, r)
	}
	ct.mu.RUnlock()

	bestDist := float32(1e9)
	for i, r := range rows {
		if r == nil {
			continue
		}
		p := drv.AbsolutePositionForObject(r)
		rh := r.Size().Height
		if rh < 1 {
			rh = 28
		}
		if y >= p.Y && y < p.Y+rh {
			return uids[i], r, p, rh
		}
		// remember closest for gaps between rows
		mid := p.Y + rh/2
		d := abs32(y - mid)
		if d < bestDist {
			bestDist = d
			uid, row, pos, h = uids[i], r, p, rh
		}
	}
	return uid, row, pos, h
}

func (ct *Tree) childIDsCopy(parentUID string) []string {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return append([]string{}, ct.childIDs[parentUID]...)
}

func (ct *Tree) parseParent(parentUID, fallbackColID string) (colID, parentItemID string) {
	if strings.HasPrefix(parentUID, "col:") {
		return strings.TrimPrefix(parentUID, "col:"), ""
	}
	if strings.HasPrefix(parentUID, "item:") {
		itemID := strings.TrimPrefix(parentUID, "item:")
		ct.mu.RLock()
		n, ok := ct.nodes[parentUID]
		ct.mu.RUnlock()
		if ok {
			return n.collectionID, itemID
		}
		return fallbackColID, itemID
	}
	return fallbackColID, ""
}

func (ct *Tree) isUnderFolder(folderUID, targetUID string) bool {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	folder, ok := ct.nodes[folderUID]
	if !ok || folder.kind != nodeFolder {
		return false
	}
	target, ok := ct.nodes[targetUID]
	if !ok {
		return false
	}
	if targetUID == folderUID {
		return true
	}
	return itemInPath(target.path, folder.item.Id)
}

func itemInPath(path []string, id string) bool {
	for _, p := range path {
		if p == id {
			return true
		}
	}
	return false
}

func indexOfUID(uids []string, uid string) int {
	for i, id := range uids {
		if id == uid {
			return i
		}
	}
	return -1
}

func (ct *Tree) endDrag() {
	ct.mu.Lock()
	active := ct.dragActive
	source := ct.dragSourceUID
	toIdx := ct.dropInsertIdx
	toCol := ct.dropToColID
	toParent := ct.dropToParentID
	cross := ct.dropCrossTree
	n, ok := ct.nodes[source]
	ct.dragActive = false
	ct.dragSourceUID = ""
	ct.clearDropLocked()
	ct.mu.Unlock()

	ct.hideGhost()
	ct.tree.Refresh()

	if !active || !ok || source == "" || toIdx < 0 {
		return
	}

	if n.kind == nodeCollection {
		siblings := ct.siblingsOf(source)
		fromIdx := indexOfUID(siblings, source)
		if fromIdx < 0 || toIdx == fromIdx {
			return
		}
		if ct.reorder.OnCollection != nil {
			ct.reorder.OnCollection(n.collectionID, toIdx-fromIdx)
		}
		return
	}

	if !cross || toCol == "" {
		return
	}
	if ct.reorder.OnRelocate != nil {
		ct.reorder.OnRelocate(n.collectionID, n.item.Id, toCol, toParent, toIdx)
	}
}

func (ct *Tree) showGhost(label string, res fyne.Resource) {
	c := fyne.CurrentApp().Driver().CanvasForObject(ct)
	if c == nil {
		return
	}
	if ct.ghost == nil {
		ct.ghostLabel = widget.NewLabel(label)
		ct.ghostIcon = widget.NewIcon(res)
		bg := canvas.NewRectangle(theme.Color(theme.ColorNameOverlayBackground))
		content := container.NewPadded(container.NewHBox(ct.ghostIcon, ct.ghostLabel))
		ct.ghost = widget.NewPopUp(container.NewStack(bg, content), c)
	} else {
		ct.ghostLabel.SetText(label)
		ct.ghostIcon.SetResource(res)
	}
	ct.ghost.Resize(ct.ghost.MinSize())
	ct.ghost.Show()
}

func (ct *Tree) moveGhost(abs fyne.Position) {
	if ct.ghost == nil || !ct.ghost.Visible() {
		return
	}
	ct.ghost.Move(fyne.NewPos(abs.X+12, abs.Y-10))
	ct.ghost.Refresh()
}

func (ct *Tree) hideGhost() {
	if ct.ghost == nil {
		return
	}
	ct.ghost.Hide()
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

// RevealItem opens collection/folder ancestors and reloads the destination branch
// so a relocated node appears under its new parent immediately.
func (ct *Tree) RevealItem(collectionID, parentItemID, itemID string) {
	if collectionID == "" {
		return
	}
	colUID := ct.CollectionUID(collectionID)
	ct.tree.OpenBranch(colUID)

	parentUID := colUID
	if parentItemID != "" {
		ct.mu.RLock()
		n, ok := ct.nodes[ct.ItemUID(parentItemID)]
		path := []string{}
		if ok {
			path = append([]string{}, n.path...)
		}
		ct.mu.RUnlock()
		for _, id := range path {
			ct.tree.OpenBranch(ct.ItemUID(id))
		}
		parentUID = ct.ItemUID(parentItemID)
		// Force Fyne Tree to reload children after UID parent change.
		ct.tree.CloseBranch(parentUID)
		ct.tree.OpenBranch(parentUID)
	} else {
		ct.tree.CloseBranch(colUID)
		ct.tree.OpenBranch(colUID)
	}

	if itemID != "" {
		ct.tree.Select(ct.ItemUID(itemID))
		ct.tree.RefreshItem(ct.ItemUID(itemID))
	}
	ct.tree.RefreshItem(parentUID)
	ct.tree.Refresh()
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
	prevNodes := make([]string, 0, len(ct.nodes))
	for uid := range ct.nodes {
		prevNodes = append(prevNodes, uid)
	}
	ct.mu.RUnlock()

	opened := make([]string, 0, len(prevNodes))
	for _, uid := range prevNodes {
		if ct.tree.IsBranchOpen(uid) {
			opened = append(opened, uid)
		}
	}

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

	log.Printf("[collections] Tree.rebuild filter=%q collections=%d nodes=%d roots=%v opened=%d",
		filter, len(visible), len(nodes), childIDs[""], len(opened))

	ct.tree.Refresh()

	ct.mu.RLock()
	roots := append([]string{}, ct.childIDs[""]...)
	ct.mu.RUnlock()

	if filter != "" {
		for _, uid := range roots {
			ct.openAllUnder(uid)
		}
		return
	}

	// First populate: expand collections so the tree isn't fully collapsed.
	if len(prevNodes) == 0 {
		for _, uid := range roots {
			ct.tree.OpenBranch(uid)
		}
		return
	}

	// Preserve expand/collapse across rebuilds (e.g. Save → PublishWorkspace).
	for _, uid := range opened {
		ct.mu.RLock()
		_, exists := ct.nodes[uid]
		ct.mu.RUnlock()
		if exists {
			ct.tree.OpenBranch(uid)
		}
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
// Draggable на всей строке: превью + underline места вставки.
type treeRow struct {
	widget.BaseWidget
	ct             *Tree
	uid            string
	reorderEnabled bool
	dragging       bool
	dragArmed      bool
	dragAccum      float32

	icon       *widget.Icon
	status     *canvas.Circle
	dot        *ui.MinSizeBox
	label      *widget.Label
	topLine    *canvas.Rectangle
	bottomLine *canvas.Rectangle
	intoBG     *canvas.Rectangle
	body       *fyne.Container
	root       *fyne.Container
}

type dropIndicator int

const (
	dropNone dropIndicator = iota
	dropBeforeIndicator
	dropAfterIndicator
	dropIntoIndicator
)

func newTreeRow(ct *Tree) *treeRow {
	icon := widget.NewIcon(theme.DocumentIcon())
	status := canvas.NewCircle(color.Transparent)
	status.Hide()
	dot := ui.NewMinSizeBox(fyne.NewSize(8, 8), status)
	label := widget.NewLabel("")
	label.Truncation = fyne.TextTruncateEllipsis
	left := container.NewHBox(icon, container.NewCenter(dot))
	body := container.NewBorder(nil, nil, left, nil, label)

	topLine := canvas.NewRectangle(color.Transparent)
	topLine.SetMinSize(fyne.NewSize(1, 2))
	topLine.Hide()
	bottomLine := canvas.NewRectangle(color.Transparent)
	bottomLine.SetMinSize(fyne.NewSize(1, 2))
	bottomLine.Hide()
	intoBG := canvas.NewRectangle(color.Transparent)
	intoBG.Hide()

	bodyStack := container.NewStack(intoBG, body)
	root := container.NewBorder(topLine, bottomLine, nil, nil, bodyStack)

	r := &treeRow{
		ct:         ct,
		icon:       icon,
		status:     status,
		dot:        dot,
		label:      label,
		topLine:    topLine,
		bottomLine: bottomLine,
		intoBG:     intoBG,
		body:       body,
		root:       root,
	}
	r.ExtendBaseWidget(r)
	return r
}

func (r *treeRow) SetUID(uid string) {
	if r.uid != "" && r.ct != nil {
		r.ct.unregisterRow(r.uid, r)
	}
	r.uid = uid
	if uid != "" && r.ct != nil {
		r.ct.registerRow(uid, r)
	}
}

func (r *treeRow) SetReorderEnabled(on bool) {
	r.reorderEnabled = on
}

func (r *treeRow) SetDraggingSource(on bool) {
	if on {
		r.label.Importance = widget.LowImportance
	} else {
		r.label.Importance = widget.MediumImportance
	}
	r.label.Refresh()
}

func (r *treeRow) SetDropIndicator(where dropIndicator) {
	r.topLine.Hide()
	r.bottomLine.Hide()
	r.intoBG.Hide()
	switch where {
	case dropBeforeIndicator:
		r.topLine.FillColor = colorInsert
		r.topLine.Show()
	case dropAfterIndicator:
		r.bottomLine.FillColor = colorInsert
		r.bottomLine.Show()
	case dropIntoIndicator:
		c := colorInsert
		c.A = 0x40
		r.intoBG.FillColor = c
		r.intoBG.Show()
	}
	r.topLine.Refresh()
	r.bottomLine.Refresh()
	r.intoBG.Refresh()
}

func (r *treeRow) Tapped(_ *fyne.PointEvent) {
	if r.dragging {
		return
	}
	if r.ct != nil && r.uid != "" {
		r.ct.onRowPrimary(r.uid)
	}
}

func (r *treeRow) TappedSecondary(e *fyne.PointEvent) {
	if r.ct != nil && r.uid != "" {
		r.ct.onRowSecondary(r.uid, e)
	}
}

func (r *treeRow) Dragged(e *fyne.DragEvent) {
	if !r.reorderEnabled || r.uid == "" || r.ct == nil {
		return
	}

	if !r.dragArmed {
		r.dragAccum += abs32(e.Dragged.DX) + abs32(e.Dragged.DY)
		if r.dragAccum < dragThreshold {
			return
		}
		r.dragArmed = true
		r.dragging = true
		r.ct.beginDrag(r.uid)
	}
	r.ct.updateDrag(e.AbsolutePosition)
}

func (r *treeRow) DragEnd() {
	was := r.dragging
	r.dragging = false
	r.dragArmed = false
	r.dragAccum = 0
	if was && r.ct != nil {
		r.ct.endDrag()
	}
}

func (r *treeRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(r.root)
}

func (r *treeRow) MinSize() fyne.Size {
	return r.root.MinSize()
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
