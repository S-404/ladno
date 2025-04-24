package store

import (
	"fmt"
	"fyne.io/fyne/v2/data/binding"
	"github.com/s-404/goose/internal/app/service"
	"time"
)

type IFooStore interface {
	DoSomething()
	GetName() *binding.ExternalString
	GetNames() *binding.ExternalStringList
	UpdateName(newValue string)
	AddName(name string)
	UpdateElementName(index int, newValue string)
}

type FooStore struct {
	srv   service.IFooService
	Name  binding.ExternalString
	Names binding.ExternalStringList
}

type WsClients struct {
	Aaaa string
}

type Ws struct {
	Nmae    string
	Guid    string
	Clients []WsClients
}

func NewFooStore(fooService service.IFooService) *FooStore {
	name := ""
	names := []string{
		fmt.Sprint(time.Now()),
		fmt.Sprint(time.Now()),
		fmt.Sprint(time.Now()),
		fmt.Sprint(time.Now()),
		fmt.Sprint(time.Now()),
		fmt.Sprint(time.Now()),
	}
	return &FooStore{
		srv:   fooService,
		Name:  binding.BindString(&name),
		Names: binding.BindStringList(&names),
	}
}

func (s *FooStore) DoSomething() {
	s.srv.DoSomething()
	fmt.Println(fmt.Sprintf("'%s' doing smth", *s.GetNameValue()))
}

func (s *FooStore) GetName() *binding.ExternalString {
	return &s.Name
}

func (s *FooStore) GetNameValue() *string {
	value, _ := s.Name.Get()
	return &value
}

func (s *FooStore) UpdateName(newValue string) {
	s.Name.Set(newValue)
}

func (s *FooStore) GetNames() *binding.ExternalStringList {
	return &s.Names
}

func (s *FooStore) AddName(name string) {
	s.Names.Append(name)
}

func (s *FooStore) UpdateElementName(index int, newValue string) {

	s.Names.SetValue(index, newValue)
}
