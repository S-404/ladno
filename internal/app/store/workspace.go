package store

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/utils"
)

// workspaceService is the async persistence surface WorkspaceStore needs.
type workspaceService interface {
	List(cb func([]entity.WorkspaceLightWeight, error))
	Find(id string, cb func(*entity.Workspace, error))
	Create(name string, cb func(*entity.Workspace, error))
	Save(workspace *entity.Workspace, cb func(error))
	Delete(id string, cb func(error))
}

type WorkspaceStore struct {
	Items            binding.UntypedList
	SelectedItem     binding.Untyped
	IsFetching       binding.Bool
	workspaceService workspaceService
}

func NewWorkspaceStore(svc workspaceService) *WorkspaceStore {
	store := WorkspaceStore{
		Items:            binding.NewUntypedList(),
		SelectedItem:     binding.NewUntyped(),
		IsFetching:       binding.NewBool(),
		workspaceService: svc,
	}

	return &store
}

func (s *WorkspaceStore) GetItems() *binding.UntypedList {
	return &s.Items
}

func (s *WorkspaceStore) GetItem() binding.Untyped {
	return s.SelectedItem
}

func (s *WorkspaceStore) GetIsFetching() *binding.Bool {
	return &s.IsFetching
}

func (s *WorkspaceStore) FetchWorkspace(id string) {
	if isFetching, _ := s.IsFetching.Get(); isFetching {
		fmt.Println("already fetching")
		return
	}
	s.IsFetching.Set(true)
	_ = s.SelectedItem.Set(nil)
	s.workspaceService.Find(id, func(data *entity.Workspace, err error) {
		fyne.Do(func() {
			if err != nil {
				fmt.Printf("fetch async with err: %s", err.Error())
				s.IsFetching.Set(false)
				return
			}
			if data == nil {
				fmt.Printf("not found")
				s.IsFetching.Set(false)
				return
			}

			_ = s.SelectedItem.Set(data)
			s.IsFetching.Set(false)
		})
	})
}

func (s *WorkspaceStore) FetchWorkspaceList() {
	if isFetching, _ := s.IsFetching.Get(); isFetching {
		fmt.Println("already fetching")
		return
	}
	s.IsFetching.Set(true)
	s.ClearList()
	s.workspaceService.List(func(data []entity.WorkspaceLightWeight, err error) {
		fyne.Do(func() {
			if err != nil {
				fmt.Printf("fetch async with err: %s", err.Error())
				s.IsFetching.Set(false)
				return
			}
			if data == nil {
				s.IsFetching.Set(false)
				return
			}
			_ = s.Items.Set(utils.UnpackArray(data))
			s.IsFetching.Set(false)
		})
	})
}

func (s *WorkspaceStore) ClearList() {
	s.Items.Set(make([]any, 0))
}

func (s *WorkspaceStore) GetWorkspaceDataItem(item binding.DataItem) *entity.Workspace {
	val, err := item.(binding.Untyped).Get()
	if err != nil {
		return nil
	}
	ws, ok := val.(*entity.Workspace)
	if !ok {
		return nil
	}

	return ws
}

func (s *WorkspaceStore) GetSelectedWorkspace() *entity.Workspace {
	return s.GetWorkspaceDataItem(s.SelectedItem)
}

// PublishWorkspace persists workspace to storage and re-publishes UI bindings.
// Untyped binding compares with ==, so same pointer would skip listeners —
// always publish a new struct pointer after in-place mutations.
func (s *WorkspaceStore) PublishWorkspace(ws *entity.Workspace) {
	if ws == nil {
		return
	}
	cp := *ws
	if ws.Collections != nil {
		cp.Collections = append([]entity.Collection(nil), ws.Collections...)
	}
	log.Printf("[collections] PublishWorkspace id=%s collections=%d ptr=%p→%p",
		cp.Id, len(cp.Collections), ws, &cp)

	saved := cp
	s.workspaceService.Save(&saved, func(err error) {
		if err != nil {
			log.Printf("[storage] workspace save error: %v", err)
			return
		}
		fyne.Do(func() {
			// Refresh lightweight list name if needed.
			s.refreshListItemName(saved.Id, saved.Name)
		})
	})
	_ = s.SelectedItem.Set(&cp)
}

func (s *WorkspaceStore) refreshListItemName(id, name string) {
	items, err := s.Items.Get()
	if err != nil || len(items) == 0 {
		return
	}
	changed := false
	for i, item := range items {
		switch v := item.(type) {
		case entity.WorkspaceLightWeight:
			if v.Id == id && v.Name != name {
				items[i] = entity.WorkspaceLightWeight{Id: id, Name: name}
				changed = true
			}
		case *entity.WorkspaceLightWeight:
			if v != nil && v.Id == id && v.Name != name {
				items[i] = entity.WorkspaceLightWeight{Id: id, Name: name}
				changed = true
			}
		}
	}
	if changed {
		_ = s.Items.Set(items)
	}
}

func (s *WorkspaceStore) UpdateSelectedWorkspace(name, connectionConfig string, folderNestingLimit int) bool {
	ws := s.GetSelectedWorkspace()
	if ws == nil {
		return false
	}
	ws.Name = name
	ws.ConnectionConfig = connectionConfig
	ws.SetFolderNestingLimit(folderNestingLimit)
	s.PublishWorkspace(ws)
	return true
}

func (s *WorkspaceStore) Create(name string, cb func(*entity.Workspace, error)) {
	s.workspaceService.Create(name, func(ws *entity.Workspace, err error) {
		fyne.Do(func() {
			if err != nil {
				log.Printf("[storage] workspace create error: %v", err)
				if cb != nil {
					cb(nil, err)
				}
				return
			}
			items, _ := s.Items.Get()
			items = append(items, entity.WorkspaceLightWeight{Id: ws.Id, Name: ws.Name})
			_ = s.Items.Set(items)
			if cb != nil {
				cb(ws, nil)
			}
		})
	})
}

func (s *WorkspaceStore) Delete(id string, cb func(error)) {
	s.workspaceService.Delete(id, func(err error) {
		fyne.Do(func() {
			if err != nil {
				log.Printf("[storage] workspace delete error: %v", err)
				if cb != nil {
					cb(err)
				}
				return
			}
			s.removeListItem(id)
			if sel := s.GetSelectedWorkspace(); sel != nil && sel.Id == id {
				_ = s.SelectedItem.Set(nil)
			}
			if cb != nil {
				cb(nil)
			}
		})
	})
}

func (s *WorkspaceStore) removeListItem(id string) {
	items, err := s.Items.Get()
	if err != nil || len(items) == 0 {
		return
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		switch v := item.(type) {
		case entity.WorkspaceLightWeight:
			if v.Id == id {
				continue
			}
		case *entity.WorkspaceLightWeight:
			if v != nil && v.Id == id {
				continue
			}
		}
		out = append(out, item)
	}
	_ = s.Items.Set(out)
}

func (s *WorkspaceStore) GetWorkspaceListItemDataItem(item binding.DataItem) *entity.WorkspaceLightWeight {
	val, err := item.(binding.Untyped).Get()
	if err != nil {
		return nil
	}

	ws, ok := val.(entity.WorkspaceLightWeight)
	if !ok {
		return nil
	}

	return &ws
}

func (s *WorkspaceStore) GetWorkspaceListItemByIndex(index int) *entity.WorkspaceLightWeight {
	item, err := s.Items.GetItem(index)
	if err != nil {
		return nil
	}

	untyped, err := item.(binding.Untyped).Get()
	if err != nil {
		return nil
	}

	v := untyped.(entity.WorkspaceLightWeight)
	return &v
}
