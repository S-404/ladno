package store

import (
	"fmt"
	"fyne.io/fyne/v2/data/binding"
	"github.com/s-404/goose/internal/app/entity"
	"github.com/s-404/goose/internal/app/service"
	"github.com/s-404/goose/internal/app/utils"
)

type IWorkspaceStore interface {
	FetchWorkspaceList()
	GetItems() *binding.UntypedList
	GetWorkspaceListItemDataItem(item binding.DataItem) *entity.WorkspaceListItem
	GetWorkspaceListItemByIndex(index int) *entity.WorkspaceListItem

	FetchWorkspace(id string)
	GetItem() *binding.Untyped
	GetWorkspaceDataItem(item binding.DataItem) *entity.Workspace

	GetIsFetching() *binding.Bool
}

type WorkspaceStore struct {
	Items            binding.UntypedList
	IsFetchingList   binding.Bool
	Item             binding.Untyped
	IsFetching       binding.Bool
	workspaceService service.IWorkspaceService
}

func NewWorkspaceStore(service service.IWorkspaceService) *WorkspaceStore {
	store := WorkspaceStore{
		Items:            binding.NewUntypedList(),
		Item:             binding.NewUntyped(),
		IsFetchingList:   binding.NewBool(),
		IsFetching:       binding.NewBool(),
		workspaceService: service,
	}

	return &store
}

func (s *WorkspaceStore) GetItems() *binding.UntypedList {
	return &s.Items
}

func (s *WorkspaceStore) GetItem() *binding.Untyped {
	return &s.Item
}

func (s *WorkspaceStore) GetIsFetching() *binding.Bool {
	return &s.IsFetchingList
}

func (s *WorkspaceStore) FetchWorkspace(id string) {
	if isFetching, _ := s.IsFetchingList.Get(); isFetching {
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
		s.Item.Set(&data)
		s.IsFetchingList.Set(false)
	})
}

func (s *WorkspaceStore) FetchWorkspaceList() {
	if isFetching, _ := s.IsFetchingList.Get(); isFetching {
		fmt.Println("already fetching")
		return
	}
	s.IsFetchingList.Set(true)
	s.ClearList()
	s.workspaceService.List(func(data []entity.WorkspaceListItem, err error) {
		if err != nil {
			fmt.Printf("fetch async with err: %s", err.Error())
			return
		}
		if data == nil {
			return
		}
		s.Items.Set(utils.UnpackArray(data))
		s.IsFetchingList.Set(false)
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

	ws, ok := val.(entity.Workspace)
	if !ok {
		return nil
	}

	return &ws
}

func (s *WorkspaceStore) GetWorkspaceListItemDataItem(item binding.DataItem) *entity.WorkspaceListItem {
	val, err := item.(binding.Untyped).Get()
	if err != nil {
		return nil
	}

	ws, ok := val.(entity.WorkspaceListItem)
	if !ok {
		return nil
	}

	return &ws
}

func (s *WorkspaceStore) GetWorkspaceListItemByIndex(index int) *entity.WorkspaceListItem {
	item, err := s.Items.GetItem(index)
	if err != nil {
		return nil
	}

	untyped, err := item.(binding.Untyped).Get()
	if err != nil {
		return nil
	}

	v := untyped.(entity.WorkspaceListItem)
	return &v
}
