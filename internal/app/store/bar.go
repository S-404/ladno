package store

import (
	"fmt"
	"fyne.io/fyne/v2/data/binding"
	"github.com/s-404/goose/internal/app/entity"
	"github.com/s-404/goose/internal/app/service"
	"github.com/s-404/goose/internal/app/utils"
)

type IBarStore interface {
	DoSomething()
	GetItems() *binding.UntypedList
	GetIsFetching() *binding.Bool
	FetchAsync()
	Fetch()
	Clear()
	GetDataItem(item binding.DataItem) *entity.Person
}

type BarStore struct {
	Items      binding.UntypedList
	IsFetching binding.Bool
	barService service.IBarService
}

func NewBarStore(service service.IBarService) *BarStore {
	store := BarStore{
		Items:      binding.NewUntypedList(),
		IsFetching: binding.NewBool(),
		barService: service,
	}

	return &store
}

func (s *BarStore) DoSomething() {
	fmt.Println(fmt.Sprintf("'%s' doing smth", "bar store"))
}

func (s *BarStore) GetItems() *binding.UntypedList {
	return &s.Items
}

func (s *BarStore) GetIsFetching() *binding.Bool {
	return &s.IsFetching
}

func (s *BarStore) FetchAsync() {
	s.IsFetching.Set(true)
	s.barService.FetchPersonsAsync(func(data []entity.Person, err error) {
		fmt.Println("fetched async")
		if err != nil {
			fmt.Printf("fetch async with err: %s", err.Error())
			return
		}
		s.Items.Set(utils.UnpackArray(data))
		s.IsFetching.Set(false)
	})
}

func (s *BarStore) Fetch() {
	data, err := s.barService.FetchPersons()
	if err != nil {
		fmt.Printf("fetch with err: %s", err.Error())
		return
	}
	s.Items.Set(utils.UnpackArray(data))
}

func (s *BarStore) Clear() {
	s.Items.Set(make([]any, 0))
}

func (s *BarStore) GetDataItem(item binding.DataItem) *entity.Person {
	val, err := item.(binding.Untyped).Get()
	if err != nil {
		return nil
	}

	person, ok := val.(entity.Person)
	if !ok {
		return nil
	}

	return &person
}
