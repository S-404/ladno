package store

import (
	"fmt"
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

	GetIsFetching() *binding.Bool
}

type WorkspaceStore struct {
	Items            binding.UntypedList
	Item             binding.Untyped
	IsFetching       binding.Bool
	workspaceService service.IWorkspaceService
}

func NewWorkspaceStore(service service.IWorkspaceService) *WorkspaceStore {
	store := WorkspaceStore{
		Items:            binding.NewUntypedList(),
		Item:             binding.NewUntyped(),
		IsFetching:       binding.NewBool(),
		workspaceService: service,
	}

	return &store
}

func (s *WorkspaceStore) GetItems() *binding.UntypedList {
	return &s.Items
}

func (s *WorkspaceStore) GetItem() binding.Untyped {
	return s.Item
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
	s.Item.Set(binding.NewUntyped())
	s.workspaceService.Find(id, func(data *entity.Workspace, err error) {
		if err != nil {
			fmt.Printf("fetch async with err: %s", err.Error())
			return
		}
		if data == nil {
			fmt.Printf("not found")
			return
		}

		s.Item.Set(data)
		s.IsFetching.Set(false)
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
		if err != nil {
			fmt.Printf("fetch async with err: %s", err.Error())
			return
		}
		if data == nil {
			return
		}
		s.Items.Set(utils.UnpackArray(data))
		s.IsFetching.Set(false)
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
	debugBindingData(item)
	ws, ok := val.(*entity.Workspace)
	if !ok {
		return nil
	}

	return ws
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

func debugBindingData(item binding.DataItem) {
	fmt.Println("=== Debug Binding Data ===")

	if untyped, ok := item.(binding.Untyped); ok {
		data, err := untyped.Get()
		if err != nil {
			fmt.Printf("Error getting data: %v\n", err)
			return
		}

		fmt.Printf("Data type: %T\n", data)
		fmt.Printf("Data value: %+v\n", data)

		// Попробуем рекурсивно проверить если это другой binding
		if innerUntyped, ok := data.(binding.Untyped); ok {
			fmt.Println("Found nested binding.Untyped")
			debugBindingData(innerUntyped)
		} else if innerDataItem, ok := data.(binding.DataItem); ok {
			fmt.Println("Found nested DataItem")
			debugBindingData(innerDataItem)
		}
	}
	fmt.Println("=== End Debug ===")
}
