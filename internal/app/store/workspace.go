package store

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/service"
	"github.com/s-404/ladno/internal/app/utils"
)

type IWorkspaceStore interface {
	FetchWorkspaceList()
	GetItems() *binding.UntypedList
	GetWorkspaceListItemDataItem(item binding.DataItem) *entity.WorkspaceLightWeight
	GetWorkspaceListItemByIndex(index int) *entity.WorkspaceLightWeight

	FetchWorkspace(id string)
	GetItem() binding.Untyped
	GetWorkspaceDataItem(item binding.DataItem) *entity.Workspace
	GetSelectedWorkspace() *entity.Workspace
	PublishWorkspace(ws *entity.Workspace)
	UpdateSelectedWorkspace(name, connectionConfig string) bool

	GetIsFetching() *binding.Bool
}

type WorkspaceStore struct {
	Items            binding.UntypedList
	SelectedItem     binding.Untyped
	IsFetching       binding.Bool
	workspaceService service.IWorkspaceService
}

func NewWorkspaceStore(service service.IWorkspaceService) *WorkspaceStore {
	store := WorkspaceStore{
		Items:            binding.NewUntypedList(),
		SelectedItem:     binding.NewUntyped(),
		IsFetching:       binding.NewBool(),
		workspaceService: service,
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

// PublishWorkspace re-publishes the selected workspace so UI bindings refresh.
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
	_ = s.SelectedItem.Set(&cp)
}

func (s *WorkspaceStore) UpdateSelectedWorkspace(name, connectionConfig string) bool {
	ws := s.GetSelectedWorkspace()
	if ws == nil {
		return false
	}
	ws.Name = name
	ws.ConnectionConfig = connectionConfig
	s.PublishWorkspace(ws)
	return true
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
