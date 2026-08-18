package store

import (
	"sync"

	"fyne.io/fyne/v2/data/binding"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
)

// draftWorkspace is the workspace surface DraftStore persists through.
type draftWorkspace interface {
	GetSelectedWorkspace() *entity.Workspace
	PublishWorkspace(ws *entity.Workspace)
}

// draftSelection is the selection surface DraftStore updates after save.
type draftSelection interface {
	GetSelection() binding.Untyped
}

// draftEnv is the env persistence surface DraftStore needs.
type draftEnv interface {
	PersistEnv(id, name string, vars []entity.EnvVariable) bool
}

type DraftStore struct {
	mu sync.Mutex

	requests    map[string]entity.RequestDraft
	folders     map[string]entity.FolderDraft
	collections map[string]entity.CollectionDraft
	envs        map[string]entity.EnvDraft

	dirtyReq map[string]bool
	dirtyFol map[string]bool
	dirtyCol map[string]bool
	dirtyEnv map[string]bool

	listeners []func()

	workspace draftWorkspace
	selection draftSelection
	env       draftEnv
}

func NewDraftStore(workspace draftWorkspace, selection draftSelection, env draftEnv) *DraftStore {
	return &DraftStore{
		requests:    map[string]entity.RequestDraft{},
		folders:     map[string]entity.FolderDraft{},
		collections: map[string]entity.CollectionDraft{},
		envs:        map[string]entity.EnvDraft{},
		dirtyReq:    map[string]bool{},
		dirtyFol:    map[string]bool{},
		dirtyCol:    map[string]bool{},
		dirtyEnv:    map[string]bool{},
		workspace:   workspace,
		selection:   selection,
		env:         env,
	}
}

func (s *DraftStore) AddDirtyListener(fn func()) {
	if fn == nil {
		return
	}
	s.mu.Lock()
	s.listeners = append(s.listeners, fn)
	s.mu.Unlock()
}

func (s *DraftStore) NotifyDirty() {
	s.mu.Lock()
	ls := append([]func(){}, s.listeners...)
	s.mu.Unlock()
	for _, fn := range ls {
		fn()
	}
}

func cloneItemRequest(req *entity.ItemRequest) entity.ItemRequest {
	if req == nil {
		return entity.ItemRequest{Auth: entity.Auth{Type: constants.AuthTypeInherited}}
	}
	out := *req
	out.Header = cloneVariables(req.Header)
	out.FormData = cloneVariables(req.FormData)
	out.Auth = cloneAuth(req.Auth)
	out.Url.Variable = cloneVariables(req.Url.Variable)
	if req.Grpc != nil {
		g := *req.Grpc
		g.Metadata = cloneVariables(req.Grpc.Metadata)
		out.Grpc = &g
	}
	if req.Ws != nil {
		w := *req.Ws
		w.Headers = cloneVariables(req.Ws.Headers)
		out.Ws = &w
	}
	if req.Nats != nil {
		n := *req.Nats
		n.Headers = cloneVariables(req.Nats.Headers)
		out.Nats = &n
	}
	if req.Kafka != nil {
		k := *req.Kafka
		k.Headers = cloneVariables(req.Kafka.Headers)
		out.Kafka = &k
	}
	return out
}

func (s *DraftStore) selectionPtr() *entity.Selection {
	val, err := s.selection.GetSelection().Get()
	if err != nil || val == nil {
		return nil
	}
	sel, ok := val.(*entity.Selection)
	if !ok {
		return nil
	}
	return sel
}

func (s *DraftStore) EnsureRequestDraft(collectionID, itemID string, name string, req *entity.ItemRequest) entity.RequestDraft {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d, ok := s.requests[itemID]; ok {
		return d
	}
	d := entity.RequestDraft{
		CollectionID: collectionID,
		Name:         name,
		Request:      cloneItemRequest(req),
	}
	s.requests[itemID] = d
	return d
}

func (s *DraftStore) GetRequestDraft(itemID string) (entity.RequestDraft, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.requests[itemID]
	return d, ok
}

func (s *DraftStore) PutRequestDraft(itemID string, draft entity.RequestDraft, markDirty bool) {
	s.mu.Lock()
	if markDirty {
		if prev, ok := s.requests[itemID]; ok && requestDraftEqual(prev, draft) {
			markDirty = false
		}
	}
	s.requests[itemID] = draft
	if markDirty {
		s.dirtyReq[itemID] = true
	}
	s.mu.Unlock()
	if markDirty {
		s.NotifyDirty()
	}
}

func (s *DraftStore) IsRequestDirty(itemID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dirtyReq[itemID]
}

func (s *DraftStore) SaveRequest(collectionID, itemID string) bool {
	s.mu.Lock()
	draft, ok := s.requests[itemID]
	if !ok {
		s.mu.Unlock()
		return false
	}
	s.mu.Unlock()

	ws := s.workspace.GetSelectedWorkspace()
	if ws == nil {
		return false
	}
	col, ok := findCollection(ws, collectionID)
	if !ok {
		return false
	}
	item := findItemByID(&col.Items, itemID)
	if item == nil || item.Request == nil {
		return false
	}
	item.Name = draft.Name
	req := cloneItemRequest(&draft.Request)
	item.Request = &req
	item.Auth = req.Auth
	s.workspace.PublishWorkspace(ws)

	s.mu.Lock()
	delete(s.dirtyReq, itemID)
	s.requests[itemID] = draft
	s.mu.Unlock()

	if cur := s.selectionPtr(); cur != nil && cur.Kind == entity.SelectionRequest && cur.ItemID == itemID {
		cur.Name = draft.Name
		cur.Auth = req.Auth
		cur.Request = item.Request
	}
	s.NotifyDirty()
	return true
}

func (s *DraftStore) EnsureFolderDraft(collectionID, itemID, name string, auth entity.Auth) entity.FolderDraft {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d, ok := s.folders[itemID]; ok {
		return d
	}
	d := entity.FolderDraft{CollectionID: collectionID, Name: name, Auth: cloneAuth(auth)}
	s.folders[itemID] = d
	return d
}

func (s *DraftStore) GetFolderDraft(itemID string) (entity.FolderDraft, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.folders[itemID]
	return d, ok
}

func (s *DraftStore) PutFolderDraft(itemID string, draft entity.FolderDraft, markDirty bool) {
	s.mu.Lock()
	s.folders[itemID] = draft
	if markDirty {
		s.dirtyFol[itemID] = true
	}
	s.mu.Unlock()
	if markDirty {
		s.NotifyDirty()
	}
}

func (s *DraftStore) IsFolderDirty(itemID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dirtyFol[itemID]
}

func (s *DraftStore) SaveFolder(collectionID, itemID string) bool {
	s.mu.Lock()
	draft, ok := s.folders[itemID]
	if !ok {
		s.mu.Unlock()
		return false
	}
	s.mu.Unlock()

	ws := s.workspace.GetSelectedWorkspace()
	if ws == nil {
		return false
	}
	col, ok := findCollection(ws, collectionID)
	if !ok {
		return false
	}
	item := findItemByID(&col.Items, itemID)
	if item == nil || item.Request != nil {
		return false
	}
	item.Name = draft.Name
	item.Auth = cloneAuth(draft.Auth)
	s.workspace.PublishWorkspace(ws)

	s.mu.Lock()
	delete(s.dirtyFol, itemID)
	s.mu.Unlock()

	if cur := s.selectionPtr(); cur != nil && cur.Kind == entity.SelectionFolder && cur.ItemID == itemID {
		cur.Name = draft.Name
		cur.Auth = item.Auth
	}
	s.NotifyDirty()
	return true
}

func (s *DraftStore) EnsureCollectionDraft(col entity.Collection) entity.CollectionDraft {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d, ok := s.collections[col.Id]; ok {
		return d
	}
	d := entity.CollectionDraft{Name: col.Name, Auth: cloneAuth(col.Auth)}
	if col.Nats != nil {
		cp := *col.Nats
		d.Nats = &cp
	}
	if col.Kafka != nil {
		cp := *col.Kafka
		d.Kafka = &cp
	}
	s.collections[col.Id] = d
	return d
}

func (s *DraftStore) GetCollectionDraft(collectionID string) (entity.CollectionDraft, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.collections[collectionID]
	return d, ok
}

func (s *DraftStore) PutCollectionDraft(collectionID string, draft entity.CollectionDraft, markDirty bool) {
	s.mu.Lock()
	s.collections[collectionID] = draft
	if markDirty {
		s.dirtyCol[collectionID] = true
	}
	s.mu.Unlock()
	if markDirty {
		s.NotifyDirty()
	}
}

func (s *DraftStore) IsCollectionDirty(collectionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dirtyCol[collectionID]
}

func (s *DraftStore) SaveCollection(collectionID string) bool {
	s.mu.Lock()
	draft, ok := s.collections[collectionID]
	if !ok {
		s.mu.Unlock()
		return false
	}
	s.mu.Unlock()

	ws := s.workspace.GetSelectedWorkspace()
	if ws == nil {
		return false
	}
	for i := range ws.Collections {
		if ws.Collections[i].Id != collectionID {
			continue
		}
		ws.Collections[i].Name = draft.Name
		ws.Collections[i].Auth = cloneAuth(draft.Auth)
		if draft.Nats != nil {
			cp := *draft.Nats
			ws.Collections[i].Nats = &cp
		} else {
			ws.Collections[i].Nats = nil
		}
		if draft.Kafka != nil {
			cp := *draft.Kafka
			ws.Collections[i].Kafka = &cp
		} else {
			ws.Collections[i].Kafka = nil
		}
		s.workspace.PublishWorkspace(ws)

		s.mu.Lock()
		delete(s.dirtyCol, collectionID)
		s.mu.Unlock()

		if cur := s.selectionPtr(); cur != nil && cur.Kind == entity.SelectionCollection && cur.CollectionID == collectionID {
			cur.Name = draft.Name
			cur.Auth = ws.Collections[i].Auth
			cur.Nats = ws.Collections[i].Nats
			cur.Kafka = ws.Collections[i].Kafka
		}
		s.NotifyDirty()
		return true
	}
	return false
}

func (s *DraftStore) EnsureEnvDraft(env *entity.Env) entity.EnvDraft {
	if env == nil {
		return entity.EnvDraft{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if d, ok := s.envs[env.Id]; ok {
		return d
	}
	vars := make([]entity.EnvVariable, len(env.Variables))
	copy(vars, env.Variables)
	d := entity.EnvDraft{Name: env.Name, Variables: vars}
	s.envs[env.Id] = d
	return d
}

func (s *DraftStore) GetEnvDraft(envID string) (entity.EnvDraft, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.envs[envID]
	return d, ok
}

func (s *DraftStore) PutEnvDraft(envID string, draft entity.EnvDraft, markDirty bool) {
	s.mu.Lock()
	s.envs[envID] = draft
	if markDirty {
		s.dirtyEnv[envID] = true
	}
	s.mu.Unlock()
	if markDirty {
		s.NotifyDirty()
	}
}

func (s *DraftStore) IsEnvDirty(envID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dirtyEnv[envID]
}

func (s *DraftStore) SaveEnv(envID string) bool {
	s.mu.Lock()
	draft, ok := s.envs[envID]
	if !ok {
		s.mu.Unlock()
		return false
	}
	s.mu.Unlock()

	if !s.env.PersistEnv(envID, draft.Name, draft.Variables) {
		return false
	}
	s.mu.Lock()
	delete(s.dirtyEnv, envID)
	s.mu.Unlock()
	s.NotifyDirty()
	return true
}

func (s *DraftStore) IsItemDirty(itemID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dirtyReq[itemID] || s.dirtyFol[itemID]
}

func (s *DraftStore) RequestDisplayName(itemID, fallback string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d, ok := s.requests[itemID]; ok {
		return d.Name
	}
	return fallback
}

func (s *DraftStore) FolderDisplayName(itemID, fallback string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d, ok := s.folders[itemID]; ok {
		return d.Name
	}
	return fallback
}

func (s *DraftStore) CollectionDisplayName(collectionID, fallback string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d, ok := s.collections[collectionID]; ok {
		return d.Name
	}
	return fallback
}

func requestDraftEqual(a, b entity.RequestDraft) bool {
	if a.CollectionID != b.CollectionID || a.Name != b.Name {
		return false
	}
	return itemRequestEqual(a.Request, b.Request)
}

func itemRequestEqual(a, b entity.ItemRequest) bool {
	ma, mb := string(a.Method), string(b.Method)
	if ma == "" {
		ma = string(constants.GET)
	}
	if mb == "" {
		mb = string(constants.GET)
	}
	if ma != mb {
		return false
	}
	if a.Url.Raw != b.Url.Raw {
		return false
	}
	am := a.BodyMode
	bm := b.BodyMode
	if am == "" {
		am = entity.RestBodyRaw
	}
	if bm == "" {
		bm = entity.RestBodyRaw
	}
	if am != bm || a.Body != b.Body {
		return false
	}
	return variablesEqual(a.Header, b.Header) &&
		variablesEqual(a.FormData, b.FormData) &&
		variablesEqual(a.Url.Variable, b.Url.Variable) &&
		authEqual(a.Auth, b.Auth)
}

func authEqual(a, b entity.Auth) bool {
	at := constants.NormalizeAuthType(a.Type)
	bt := constants.NormalizeAuthType(b.Type)
	if at != bt {
		return false
	}
	switch at {
	case constants.AuthTypeInherited, constants.AuthTypeNoAuth:
		return true
	case constants.AuthTypeBasic:
		return entity.AuthVar(a.Data, constants.AuthDataUsername) == entity.AuthVar(b.Data, constants.AuthDataUsername) &&
			entity.AuthVar(a.Data, constants.AuthDataPassword) == entity.AuthVar(b.Data, constants.AuthDataPassword)
	case constants.AuthTypeBearer:
		pa := constants.AuthDefaultTokenPrefix
		pb := constants.AuthDefaultTokenPrefix
		if entity.AuthHasVar(a.Data, constants.AuthDataPrefix) {
			pa = entity.AuthVar(a.Data, constants.AuthDataPrefix)
		}
		if entity.AuthHasVar(b.Data, constants.AuthDataPrefix) {
			pb = entity.AuthVar(b.Data, constants.AuthDataPrefix)
		}
		return pa == pb &&
			entity.AuthVar(a.Data, constants.AuthDataToken) == entity.AuthVar(b.Data, constants.AuthDataToken)
	case constants.AuthTypeAPIKey:
		aa := entity.AuthVar(a.Data, constants.AuthDataAddTo)
		ab := entity.AuthVar(b.Data, constants.AuthDataAddTo)
		if aa == "" {
			aa = constants.AuthAddToHeader
		}
		if ab == "" {
			ab = constants.AuthAddToHeader
		}
		return entity.AuthVar(a.Data, constants.AuthDataKey) == entity.AuthVar(b.Data, constants.AuthDataKey) &&
			entity.AuthVar(a.Data, constants.AuthDataValue) == entity.AuthVar(b.Data, constants.AuthDataValue) &&
			aa == ab
	default:
		return variablesEqual(a.Data, b.Data)
	}
}

func variablesEqual(a, b []entity.Variable) bool {
	am := varMap(a)
	bm := varMap(b)
	if len(am) != len(bm) {
		return false
	}
	for k, v := range am {
		if bm[k] != v {
			return false
		}
	}
	return true
}

func varMap(in []entity.Variable) map[string]string {
	out := make(map[string]string, len(in))
	for _, v := range in {
		if v.Key == "" || v.Value == "" {
			continue
		}
		out[v.Key] = v.Value
	}
	return out
}
