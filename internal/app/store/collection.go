package store

import (
	"log"
	"time"

	"fyne.io/fyne/v2/data/binding"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
	"github.com/s-404/ladno/internal/app/utils"
)

type ISelectionStore interface {
	GetSelection() binding.Untyped
	SetSelection(sel entity.Selection)
	ClearSelection()

	UpdateCollection(id string, name string, auth entity.Auth, nats *entity.NatsConnection)
	UpdateFolder(collectionID, itemID, name string, auth entity.Auth)
	UpdateRequestAuth(collectionID, itemID string, auth entity.Auth)
	UpdateRequestName(collectionID, itemID, name string)

	CreateCollection(colType constants.CollectionType) (collectionID string, ok bool)
	DeleteCollection(id string) bool
	AddFolder(collectionID, parentItemID string) (itemID string, path []string, ok bool)
	AddRequest(collectionID, parentItemID string) (itemID string, path []string, ok bool)
	DuplicateRequest(collectionID, itemID string) (newID string, path []string, ok bool)
	DeleteItem(collectionID, itemID string) bool
}

type SelectionStore struct {
	Selection binding.Untyped
	workspace IWorkspaceStore
}

func NewSelectionStore(workspace IWorkspaceStore) *SelectionStore {
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

func (s *SelectionStore) UpdateCollection(id string, name string, auth entity.Auth, nats *entity.NatsConnection) {
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
		s.workspace.PublishWorkspace(ws)

		cur := s.current()
		if cur.Kind == entity.SelectionCollection && cur.CollectionID == id {
			var natsSel *entity.NatsConnection
			if ws.Collections[i].Nats != nil {
				cp := *ws.Collections[i].Nats
				natsSel = &cp
			}
			s.SetSelection(entity.Selection{
				Kind:           entity.SelectionCollection,
				CollectionID:   id,
				CollectionType: ws.Collections[i].Type,
				Name:           name,
				Auth:           auth,
				Nats:           natsSel,
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
		Name:      "new",
		Auth:      entity.Auth{Type: constants.AuthTypeNoAuth},
		Items:     []entity.CollectionItem{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if colType == constants.CollectionTypeNATS {
		col.Nats = &entity.NatsConnection{Host: "{{natsHost}}", Port: "{{natsPort}}"}
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

func (s *SelectionStore) AddFolder(collectionID, parentItemID string) (string, []string, bool) {
	return s.addItem(collectionID, parentItemID, newFolderItem())
}

func (s *SelectionStore) AddRequest(collectionID, parentItemID string) (string, []string, bool) {
	ws := s.workspace.GetSelectedWorkspace()
	if ws == nil {
		return "", nil, false
	}
	col, ok := findCollection(ws, collectionID)
	if !ok {
		return "", nil, false
	}
	return s.addItem(collectionID, parentItemID, newRequestItem(col.Type))
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

func newFolderItem() entity.CollectionItem {
	return entity.CollectionItem{
		Id:   utils.NewID("f"),
		Name: "new",
		Auth: entity.Auth{Type: constants.AuthTypeInherited},
		Item: []entity.CollectionItem{},
	}
}

func newRequestItem(colType constants.CollectionType) entity.CollectionItem {
	req := &entity.ItemRequest{
		Auth: entity.Auth{Type: constants.AuthTypeInherited},
	}
	switch constants.NormalizeCollectionType(colType) {
	case constants.CollectionTypeGRPC:
		req.Grpc = &entity.GrpcRequest{}
	case constants.CollectionTypeWS:
		req.Ws = &entity.WsRequest{}
	case constants.CollectionTypeNATS:
		req.Nats = &entity.NatsRequest{Subject: "{{natsSubject}}"}
	default:
		req.Method = constants.GET
		req.Url = entity.RequestUrl{Raw: "{{baseUrl}}"}
	}
	return entity.CollectionItem{
		Id:      utils.NewID("r"),
		Name:    "new",
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
	r.Auth = cloneAuth(src.Request.Auth)
	r.Url.Variable = cloneVariables(src.Request.Url.Variable)
	if src.Request.Grpc != nil {
		g := *src.Request.Grpc
		g.Metadata = cloneVariables(src.Request.Grpc.Metadata)
		r.Grpc = &g
	}
	if src.Request.Ws != nil {
		w := *src.Request.Ws
		w.Headers = cloneVariables(src.Request.Ws.Headers)
		r.Ws = &w
	}
	if src.Request.Nats != nil {
		n := *src.Request.Nats
		n.Headers = cloneVariables(src.Request.Nats.Headers)
		r.Nats = &n
	}
	out.Request = &r
	return out
}

func cloneAuth(a entity.Auth) entity.Auth {
	return entity.Auth{
		Type: a.Type,
		Data: cloneVariables(a.Data),
	}
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
	for i := range *items {
		if (*items)[i].Id == id {
			*items = append((*items)[:i], (*items)[i+1:]...)
			return true
		}
		if deleteItemByID(&(*items)[i].Item, id) {
			return true
		}
	}
	return false
}

func itemInPath(path []string, id string) bool {
	for _, p := range path {
		if p == id {
			return true
		}
	}
	return false
}
