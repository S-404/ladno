package store

import (
	"log"
	"time"

	"fyne.io/fyne/v2/data/binding"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
	"github.com/s-404/ladno/internal/app/utils"
)

// selectionWorkspace is the workspace surface SelectionStore mutates.
type selectionWorkspace interface {
	GetSelectedWorkspace() *entity.Workspace
	PublishWorkspace(ws *entity.Workspace)
}

type SelectionStore struct {
	Selection binding.Untyped
	workspace selectionWorkspace
}

func NewSelectionStore(workspace selectionWorkspace) *SelectionStore {
	s := &SelectionStore{
		Selection: binding.NewUntyped(),
		workspace: workspace,
	}
	_ = s.Selection.Set(&entity.Selection{Kind: entity.SelectionNone})
	return s
}

func (s *SelectionStore) GetSelection() binding.Untyped {
	return s.Selection
}

func (s *SelectionStore) SetSelection(sel entity.Selection) {
	cp := sel
	if len(sel.Path) > 0 {
		cp.Path = append([]string{}, sel.Path...)
	}
	_ = s.Selection.Set(&cp)
}

func (s *SelectionStore) ClearSelection() {
	s.SetSelection(entity.Selection{Kind: entity.SelectionNone})
}

func (s *SelectionStore) current() *entity.Selection {
	val, err := s.Selection.Get()
	if err != nil || val == nil {
		return &entity.Selection{Kind: entity.SelectionNone}
	}
	sel, ok := val.(*entity.Selection)
	if !ok || sel == nil {
		return &entity.Selection{Kind: entity.SelectionNone}
	}
	return sel
}

func (s *SelectionStore) UpdateCollection(id string, name string, auth entity.Auth, nats *entity.NatsConnection, kafka *entity.KafkaConnection) {
	ws := s.workspace.GetSelectedWorkspace()
	if ws == nil {
		return
	}
	for i := range ws.Collections {
		if ws.Collections[i].Id != id {
			continue
		}
		ws.Collections[i].Name = name
		ws.Collections[i].Auth = auth
		if nats != nil {
			cp := *nats
			ws.Collections[i].Nats = &cp
		} else {
			ws.Collections[i].Nats = nil
		}
		if kafka != nil {
			cp := *kafka
			ws.Collections[i].Kafka = &cp
		} else {
			ws.Collections[i].Kafka = nil
		}
		s.workspace.PublishWorkspace(ws)

		cur := s.current()
		if cur.Kind == entity.SelectionCollection && cur.CollectionID == id {
			var natsSel *entity.NatsConnection
			if ws.Collections[i].Nats != nil {
				cp := *ws.Collections[i].Nats
				natsSel = &cp
			}
			var kafkaSel *entity.KafkaConnection
			if ws.Collections[i].Kafka != nil {
				cp := *ws.Collections[i].Kafka
				kafkaSel = &cp
			}
			s.SetSelection(entity.Selection{
				Kind:           entity.SelectionCollection,
				CollectionID:   id,
				CollectionType: ws.Collections[i].Type,
				Name:           name,
				Auth:           auth,
				Nats:           natsSel,
				Kafka:          kafkaSel,
			})
		}
		return
	}
}

func (s *SelectionStore) UpdateFolder(collectionID, itemID, name string, auth entity.Auth) {
	ws := s.workspace.GetSelectedWorkspace()
	if ws == nil {
		return
	}
	col, ok := findCollection(ws, collectionID)
	if !ok {
		return
	}
	item := findItemByID(&col.Items, itemID)
	if item == nil || item.Request != nil {
		return
	}
	item.Name = name
	item.Auth = auth
	s.workspace.PublishWorkspace(ws)

	cur := s.current()
	if cur.Kind == entity.SelectionFolder && cur.ItemID == itemID {
		s.SetSelection(entity.Selection{
			Kind:           entity.SelectionFolder,
			CollectionID:   collectionID,
			CollectionType: col.Type,
			ItemID:         itemID,
			Path:           cur.Path,
			Name:           name,
			Auth:           auth,
		})
	}
}

func (s *SelectionStore) UpdateRequestAuth(collectionID, itemID string, auth entity.Auth) {
	ws := s.workspace.GetSelectedWorkspace()
	if ws == nil {
		return
	}
	col, ok := findCollection(ws, collectionID)
	if !ok {
		return
	}
	item := findItemByID(&col.Items, itemID)
	if item == nil || item.Request == nil {
		return
	}
	item.Request.Auth = auth
	s.workspace.PublishWorkspace(ws)

	cur := s.current()
	if cur.Kind == entity.SelectionRequest && cur.ItemID == itemID {
		req := item.Request
		s.SetSelection(entity.Selection{
			Kind:           entity.SelectionRequest,
			CollectionID:   collectionID,
			CollectionType: col.Type,
			ItemID:         itemID,
			Path:           cur.Path,
			Name:           item.Name,
			Auth:           auth,
			Request:        req,
		})
	}
}

func (s *SelectionStore) UpdateRequestName(collectionID, itemID, name string) {
	ws := s.workspace.GetSelectedWorkspace()
	if ws == nil {
		return
	}
	col, ok := findCollection(ws, collectionID)
	if !ok {
		return
	}
	item := findItemByID(&col.Items, itemID)
	if item == nil || item.Request == nil {
		return
	}
	if item.Name == name {
		return
	}
	item.Name = name
	s.workspace.PublishWorkspace(ws)

	// Update selection in-place — avoid SetSelection so request editors keep unsaved fields.
	if cur := s.current(); cur.Kind == entity.SelectionRequest && cur.ItemID == itemID {
		cur.Name = name
	}
}

func (s *SelectionStore) CreateCollection(colType constants.CollectionType) (string, bool) {
	ws := s.workspace.GetSelectedWorkspace()
	if ws == nil {
		log.Printf("[collections] CreateCollection(%s): no workspace", colType)
		return "", false
	}
	colType = constants.NormalizeCollectionType(colType)
	now := time.Now()
	col := entity.Collection{
		Id:        utils.NewID("c"),
		Version:   1,
		Type:      colType,
		Name:      entity.DefaultNewCollectionName,
		Auth:      entity.Auth{Type: constants.AuthTypeNoAuth},
		Items:     []entity.CollectionItem{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if colType == constants.CollectionTypeNATS {
		col.Nats = &entity.NatsConnection{}
	}
	if colType == constants.CollectionTypeKafka {
		col.Kafka = &entity.KafkaConnection{}
	}
	ws.Collections = append(ws.Collections, col)
	log.Printf("[collections] CreateCollection type=%s id=%s name=%q total=%d",
		colType, col.Id, col.Name, len(ws.Collections))
	s.workspace.PublishWorkspace(ws)
	return col.Id, true
}

func (s *SelectionStore) DeleteCollection(id string) bool {
	ws := s.workspace.GetSelectedWorkspace()
	if ws == nil {
		log.Printf("[collections] DeleteCollection(%s): no workspace", id)
		return false
	}
	idx := -1
	for i := range ws.Collections {
		if ws.Collections[i].Id == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		log.Printf("[collections] DeleteCollection(%s): not found", id)
		return false
	}
	name := ws.Collections[idx].Name
	ws.Collections = append(ws.Collections[:idx], ws.Collections[idx+1:]...)
	log.Printf("[collections] DeleteCollection id=%s name=%q remaining=%d", id, name, len(ws.Collections))
	s.workspace.PublishWorkspace(ws)

	cur := s.current()
	if cur.CollectionID == id {
		s.ClearSelection()
	}
	return true
}

func (s *SelectionStore) MoveCollection(id string, steps int) bool {
	if id == "" || steps == 0 {
		return false
	}
	ws := s.workspace.GetSelectedWorkspace()
	if ws == nil {
		return false
	}
	idx := -1
	for i := range ws.Collections {
		if ws.Collections[i].Id == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false
	}
	newIdx := clampIndex(idx+steps, len(ws.Collections))
	if newIdx == idx {
		return false
	}
	ws.Collections = moveSliceItem(ws.Collections, idx, newIdx)
	log.Printf("[collections] MoveCollection id=%s %d→%d", id, idx, newIdx)
	s.workspace.PublishWorkspace(ws)
	return true
}

func (s *SelectionStore) AddFolder(collectionID, parentItemID string) (string, []string, bool) {
	ws := s.workspace.GetSelectedWorkspace()
	if ws == nil {
		return "", nil, false
	}
	col, ok := findCollection(ws, collectionID)
	if !ok {
		return "", nil, false
	}
	if !s.folderNestingAllowed(col.Items, parentItemID, 1) {
		log.Printf("[collections] AddFolder: nesting limit reached col=%s parent=%s", collectionID, parentItemID)
		return "", nil, false
	}
	return s.addItem(collectionID, parentItemID, newFolderItem())
}

// CanAddFolder reports whether a new folder may be created under parentItemID.
func (s *SelectionStore) CanAddFolder(collectionID, parentItemID string) bool {
	ws := s.workspace.GetSelectedWorkspace()
	if ws == nil {
		return false
	}
	col, ok := findCollection(ws, collectionID)
	if !ok {
		return false
	}
	return s.folderNestingAllowed(col.Items, parentItemID, 1)
}

func (s *SelectionStore) AddRequest(collectionID, parentItemID string, kind constants.RequestKind) (string, []string, bool) {
	ws := s.workspace.GetSelectedWorkspace()
	if ws == nil {
		return "", nil, false
	}
	col, ok := findCollection(ws, collectionID)
	if !ok {
		return "", nil, false
	}
	if constants.IsHTTPCollection(col.Type) {
		switch kind {
		case constants.RequestKindREST, constants.RequestKindGRPC, constants.RequestKindWS, constants.RequestKindSocketIO:
		default:
			kind = constants.RequestKindREST
		}
	} else {
		kind = constants.RequestKindForCollection(col.Type)
	}
	return s.addItem(collectionID, parentItemID, newRequestItem(kind))
}

func (s *SelectionStore) DuplicateRequest(collectionID, itemID string) (string, []string, bool) {
	ws := s.workspace.GetSelectedWorkspace()
	if ws == nil {
		log.Printf("[collections] DuplicateRequest: no workspace")
		return "", nil, false
	}
	col, ok := findCollection(ws, collectionID)
	if !ok {
		log.Printf("[collections] DuplicateRequest: collection %s not found", collectionID)
		return "", nil, false
	}
	src := findItemByID(&col.Items, itemID)
	if src == nil || src.Request == nil {
		log.Printf("[collections] DuplicateRequest: request %s not found", itemID)
		return "", nil, false
	}

	dup := cloneRequestItem(*src)
	parentPath := findItemPath(col.Items, itemID)
	if parentPath == nil {
		return "", nil, false
	}
	// Insert after source in the same sibling list.
	siblings, idx := findSiblingList(&col.Items, itemID)
	if siblings == nil || idx < 0 {
		return "", nil, false
	}
	insertAt := idx + 1
	*siblings = append((*siblings)[:insertAt], append([]entity.CollectionItem{dup}, (*siblings)[insertAt:]...)...)

	path := append(append([]string{}, parentPath[:len(parentPath)-1]...), dup.Id)
	log.Printf("[collections] DuplicateRequest src=%s new=%s name=%q col=%s", itemID, dup.Id, dup.Name, collectionID)
	s.workspace.PublishWorkspace(ws)
	return dup.Id, path, true
}

func (s *SelectionStore) addItem(collectionID, parentItemID string, item entity.CollectionItem) (string, []string, bool) {
	ws := s.workspace.GetSelectedWorkspace()
	if ws == nil {
		log.Printf("[collections] addItem: no workspace (parentCol=%s parentItem=%s)", collectionID, parentItemID)
		return "", nil, false
	}
	col, ok := findCollection(ws, collectionID)
	if !ok {
		log.Printf("[collections] addItem: collection %s not found", collectionID)
		return "", nil, false
	}

	var path []string
	if parentItemID == "" {
		col.Items = append(col.Items, item)
		path = []string{item.Id}
	} else {
		parent := findItemByID(&col.Items, parentItemID)
		if parent == nil || parent.Request != nil {
			log.Printf("[collections] addItem: bad parent item=%s (nil=%v isRequest=%v)",
				parentItemID, parent == nil, parent != nil && parent.Request != nil)
			return "", nil, false
		}
		parentPath := findItemPath(col.Items, parentItemID)
		if parentPath == nil {
			log.Printf("[collections] addItem: parent path not found item=%s", parentItemID)
			return "", nil, false
		}
		parent.Item = append(parent.Item, item)
		path = append(append([]string{}, parentPath...), item.Id)
	}

	kind := "folder"
	if item.Request != nil {
		kind = "request"
	}
	log.Printf("[collections] addItem kind=%s id=%s name=%q col=%s parent=%s path=%v itemsInCol=%d",
		kind, item.Id, item.Name, collectionID, parentItemID, path, len(col.Items))
	s.workspace.PublishWorkspace(ws)
	return item.Id, path, true
}

func (s *SelectionStore) DeleteItem(collectionID, itemID string) bool {
	ws := s.workspace.GetSelectedWorkspace()
	if ws == nil {
		log.Printf("[collections] DeleteItem: no workspace")
		return false
	}
	col, ok := findCollection(ws, collectionID)
	if !ok {
		log.Printf("[collections] DeleteItem: collection %s not found", collectionID)
		return false
	}
	if !deleteItemByID(&col.Items, itemID) {
		log.Printf("[collections] DeleteItem: item %s not found in %s", itemID, collectionID)
		return false
	}
	log.Printf("[collections] DeleteItem col=%s item=%s", collectionID, itemID)
	s.workspace.PublishWorkspace(ws)

	cur := s.current()
	if cur.CollectionID == collectionID && itemInPath(cur.Path, itemID) {
		s.ClearSelection()
	}
	return true
}

func (s *SelectionStore) MoveItem(collectionID, itemID string, steps int) bool {
	if collectionID == "" || itemID == "" || steps == 0 {
		return false
	}
	ws := s.workspace.GetSelectedWorkspace()
	if ws == nil {
		return false
	}
	col, ok := findCollection(ws, collectionID)
	if !ok {
		return false
	}
	list, idx := findSiblingList(&col.Items, itemID)
	if list == nil || idx < 0 {
		return false
	}
	newIdx := clampIndex(idx+steps, len(*list))
	if newIdx == idx {
		return false
	}
	*list = moveSliceItem(*list, idx, newIdx)
	log.Printf("[collections] MoveItem col=%s item=%s %d→%d", collectionID, itemID, idx, newIdx)
	s.workspace.PublishWorkspace(ws)
	return true
}

func (s *SelectionStore) RelocateItem(fromCollectionID, itemID, toCollectionID, toParentItemID string, toIndex int) bool {
	if fromCollectionID == "" || itemID == "" || toCollectionID == "" {
		return false
	}
	ws := s.workspace.GetSelectedWorkspace()
	if ws == nil {
		return false
	}
	fromCol, ok := findCollection(ws, fromCollectionID)
	if !ok {
		return false
	}
	toCol, ok := findCollection(ws, toCollectionID)
	if !ok {
		return false
	}
	if constants.NormalizeCollectionType(fromCol.Type) != constants.NormalizeCollectionType(toCol.Type) {
		log.Printf("[collections] RelocateItem: type mismatch %s → %s", fromCol.Type, toCol.Type)
		return false
	}

	src := findItemByID(&fromCol.Items, itemID)
	if src == nil {
		return false
	}

	srcPath := findItemPath(fromCol.Items, itemID)
	currentParentID := ""
	if len(srcPath) > 1 {
		currentParentID = srcPath[len(srcPath)-2]
	}
	sameParent := fromCollectionID == toCollectionID && currentParentID == toParentItemID
	if src.Request == nil && !sameParent {
		if !s.folderNestingAllowed(toCol.Items, toParentItemID, subtreeFolderDepth(*src)) {
			log.Printf("[collections] RelocateItem: nesting limit item=%s under=%s", itemID, toParentItemID)
			return false
		}
	}

	// Prevent moving a folder into itself or its descendants.
	if src.Request == nil && toParentItemID != "" {
		if toParentItemID == itemID {
			return false
		}
		if fromCollectionID == toCollectionID {
			parentPath := findItemPath(fromCol.Items, toParentItemID)
			if itemInPath(parentPath, itemID) {
				log.Printf("[collections] RelocateItem: refuse cycle item=%s under=%s", itemID, toParentItemID)
				return false
			}
		}
	}

	// Resolve destination before detach only to compute index / no-op.
	var destBefore *[]entity.CollectionItem
	if toParentItemID == "" {
		destBefore = &toCol.Items
	} else {
		parent := findItemByID(&toCol.Items, toParentItemID)
		if parent == nil || parent.Request != nil {
			log.Printf("[collections] RelocateItem: bad parent %s", toParentItemID)
			return false
		}
		destBefore = &parent.Item
	}

	fromList, fromIdx := findSiblingList(&fromCol.Items, itemID)
	sameList := fromList == destBefore
	if toIndex < 0 {
		toIndex = 0
	}
	if sameList && fromIdx == toIndex {
		return false
	}
	if !sameList && toIndex > len(*destBefore) {
		toIndex = len(*destBefore)
	}
	if sameList && toIndex > len(*destBefore)-1 {
		toIndex = len(*destBefore) - 1
	}

	item, ok := detachItemByID(&fromCol.Items, itemID)
	if !ok {
		return false
	}

	// Re-resolve destination AFTER detach: removing from a sibling slice can
	// reallocate and invalidate pointers into that backing array.
	var dest *[]entity.CollectionItem
	if toParentItemID == "" {
		dest = &toCol.Items
	} else {
		parent := findItemByID(&toCol.Items, toParentItemID)
		if parent == nil || parent.Request != nil {
			log.Printf("[collections] RelocateItem: parent lost after detach %s", toParentItemID)
			// Best-effort: put item back at collection root.
			toCol.Items = append(toCol.Items, item)
			s.workspace.PublishWorkspace(ws)
			return false
		}
		dest = &parent.Item
	}
	if toIndex > len(*dest) {
		toIndex = len(*dest)
	}
	if toIndex < 0 {
		toIndex = 0
	}
	*dest = append((*dest)[:toIndex], append([]entity.CollectionItem{item}, (*dest)[toIndex:]...)...)

	log.Printf("[collections] RelocateItem item=%s %s → %s/%s idx=%d",
		itemID, fromCollectionID, toCollectionID, toParentItemID, toIndex)
	s.workspace.PublishWorkspace(ws)

	cur := s.current()
	if cur.ItemID == itemID {
		path := findItemPath(toCol.Items, itemID)
		kind := entity.SelectionFolder
		auth := item.Auth
		var req *entity.ItemRequest
		if item.Request != nil {
			kind = entity.SelectionRequest
			auth = item.Request.Auth
			req = item.Request
		}
		s.SetSelection(entity.Selection{
			Kind:           kind,
			CollectionID:   toCollectionID,
			CollectionType: toCol.Type,
			ItemID:         itemID,
			Path:           path,
			Name:           item.Name,
			Auth:           auth,
			Request:        req,
		})
	}
	return true
}

func newFolderItem() entity.CollectionItem {
	return entity.CollectionItem{
		Id:   utils.NewID("f"),
		Name: entity.DefaultNewFolderName,
		Auth: entity.Auth{Type: constants.AuthTypeInherited},
		Item: []entity.CollectionItem{},
	}
}

func newRequestItem(kind constants.RequestKind) entity.CollectionItem {
	req := &entity.ItemRequest{
		Auth: entity.Auth{Type: constants.AuthTypeInherited},
	}
	switch kind {
	case constants.RequestKindGRPC:
		req.Grpc = &entity.GrpcRequest{}
	case constants.RequestKindWS:
		req.Ws = &entity.WsRequest{}
	case constants.RequestKindSocketIO:
		req.SocketIO = &entity.SocketIORequest{}
	case constants.RequestKindNATS:
		req.Nats = &entity.NatsRequest{}
	case constants.RequestKindKafka:
		req.Kafka = &entity.KafkaRequest{}
	default:
		req.Method = constants.GET
	}
	return entity.CollectionItem{
		Id:      utils.NewID("r"),
		Name:    constants.DefaultNewRequestName(kind),
		Auth:    entity.Auth{Type: constants.AuthTypeInherited},
		Request: req,
	}
}

func cloneRequestItem(src entity.CollectionItem) entity.CollectionItem {
	out := entity.CollectionItem{
		Id:   utils.NewID("r"),
		Name: src.Name + " copy",
		Auth: cloneAuth(src.Auth),
	}
	if src.Request == nil {
		return out
	}
	r := *src.Request
	r.Header = cloneVariables(src.Request.Header)
	r.FormData = cloneVariables(src.Request.FormData)
	r.URLEncoded = cloneVariables(src.Request.URLEncoded)
	r.Auth = cloneAuth(src.Request.Auth)
	r.Url.Variable = cloneVariables(src.Request.Url.Variable)
	if src.Request.Grpc != nil {
		g := *src.Request.Grpc
		g.Metadata = cloneVariables(src.Request.Grpc.Metadata)
		g.ProtoFiles = cloneGrpcProtoFiles(src.Request.Grpc.ProtoFiles)
		r.Grpc = &g
	}
	if src.Request.Ws != nil {
		w := *src.Request.Ws
		w.Headers = cloneVariables(src.Request.Ws.Headers)
		r.Ws = &w
	}
	if src.Request.SocketIO != nil {
		sio := *src.Request.SocketIO
		sio.Headers = cloneVariables(src.Request.SocketIO.Headers)
		sio.Query = cloneVariables(src.Request.SocketIO.Query)
		r.SocketIO = &sio
	}
	if src.Request.Nats != nil {
		n := *src.Request.Nats
		n.Headers = cloneVariables(src.Request.Nats.Headers)
		r.Nats = &n
	}
	if src.Request.Kafka != nil {
		k := *src.Request.Kafka
		k.Headers = cloneVariables(src.Request.Kafka.Headers)
		r.Kafka = &k
	}
	r.Event = cloneEvent(src.Request.Event)
	out.Request = &r
	return out
}

func cloneAuth(a entity.Auth) entity.Auth {
	out := entity.Auth{
		Type: constants.NormalizeAuthType(a.Type),
		Data: cloneVariables(a.Data),
	}
	if out.Type == "" {
		out.Type = constants.AuthTypeNoAuth
	}
	if out.Type == constants.AuthTypeBearer {
		hasPrefix := false
		for _, v := range out.Data {
			if v.Key == constants.AuthDataPrefix {
				hasPrefix = true
				break
			}
		}
		if !hasPrefix {
			out.Data = append(append([]entity.Variable{}, out.Data...), entity.Variable{
				Key: constants.AuthDataPrefix, Value: constants.AuthDefaultTokenPrefix,
			})
		}
	}
	return out
}

func cloneVariables(in []entity.Variable) []entity.Variable {
	if in == nil {
		return nil
	}
	out := make([]entity.Variable, len(in))
	copy(out, in)
	return out
}

func findCollection(ws *entity.Workspace, id string) (*entity.Collection, bool) {
	for i := range ws.Collections {
		if ws.Collections[i].Id == id {
			return &ws.Collections[i], true
		}
	}
	return nil, false
}

func findItemByID(items *[]entity.CollectionItem, id string) *entity.CollectionItem {
	for i := range *items {
		if (*items)[i].Id == id {
			return &(*items)[i]
		}
		if found := findItemByID(&(*items)[i].Item, id); found != nil {
			return found
		}
	}
	return nil
}

func (s *SelectionStore) folderNestingLimit() int {
	if s == nil || s.workspace == nil {
		return entity.DefaultFolderNestingLimit
	}
	return s.workspace.GetSelectedWorkspace().GetFolderNestingLimit()
}

func (s *SelectionStore) folderNestingAllowed(items []entity.CollectionItem, parentItemID string, extraDepth int) bool {
	limit := s.folderNestingLimit()
	if limit < 0 {
		return true
	}
	return parentFolderDepth(items, parentItemID)+extraDepth <= limit
}

func parentFolderDepth(items []entity.CollectionItem, parentItemID string) int {
	if parentItemID == "" {
		return 0
	}
	return len(findItemPath(items, parentItemID))
}

func subtreeFolderDepth(item entity.CollectionItem) int {
	if item.Request != nil {
		return 0
	}
	maxChild := 0
	for _, ch := range item.Item {
		if d := subtreeFolderDepth(ch); d > maxChild {
			maxChild = d
		}
	}
	return 1 + maxChild
}

func findItemPath(items []entity.CollectionItem, id string) []string {
	for i := range items {
		if items[i].Id == id {
			return []string{items[i].Id}
		}
		if sub := findItemPath(items[i].Item, id); sub != nil {
			return append([]string{items[i].Id}, sub...)
		}
	}
	return nil
}

func findSiblingList(items *[]entity.CollectionItem, id string) (*[]entity.CollectionItem, int) {
	for i := range *items {
		if (*items)[i].Id == id {
			return items, i
		}
		if list, idx := findSiblingList(&(*items)[i].Item, id); list != nil {
			return list, idx
		}
	}
	return nil, -1
}

func deleteItemByID(items *[]entity.CollectionItem, id string) bool {
	_, ok := detachItemByID(items, id)
	return ok
}

func detachItemByID(items *[]entity.CollectionItem, id string) (entity.CollectionItem, bool) {
	for i := range *items {
		if (*items)[i].Id == id {
			item := (*items)[i]
			*items = append((*items)[:i], (*items)[i+1:]...)
			return item, true
		}
		if item, ok := detachItemByID(&(*items)[i].Item, id); ok {
			return item, true
		}
	}
	return entity.CollectionItem{}, false
}

func itemInPath(path []string, id string) bool {
	for _, p := range path {
		if p == id {
			return true
		}
	}
	return false
}

func clampIndex(idx, length int) int {
	if length <= 0 {
		return 0
	}
	if idx < 0 {
		return 0
	}
	if idx >= length {
		return length - 1
	}
	return idx
}

func moveSliceItem[T any](items []T, from, to int) []T {
	if from == to || from < 0 || to < 0 || from >= len(items) || to >= len(items) {
		return items
	}
	item := items[from]
	items = append(items[:from], items[from+1:]...)
	items = append(items[:to], append([]T{item}, items[to:]...)...)
	return items
}
