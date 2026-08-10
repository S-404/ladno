package store

import (
	"fyne.io/fyne/v2/data/binding"
	"github.com/s-404/ladno/internal/app/entity"
)

type ISelectionStore interface {
	GetSelection() binding.Untyped
	SetSelection(sel entity.Selection)
	ClearSelection()

	UpdateCollection(id string, name string, auth entity.Auth, nats *entity.NatsConnection)
	UpdateFolder(collectionID, itemID, name string, auth entity.Auth)
	UpdateRequestAuth(collectionID, itemID string, auth entity.Auth)
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
