package service

import (
	"fmt"

	"github.com/s-404/goose/internal/app/repository"
)

type IFooService interface {
	DoSomething()
}

type FooService struct {
	name          string
	fooRepository repository.IFooRepository
}

func NewFooService(fooRepository repository.IFooRepository) *FooService {
	return &FooService{
		name:          "foo repo",
		fooRepository: fooRepository,
	}
}

func (s *FooService) DoSomething() {
	fmt.Printf("'%s' doing smth", s.name)
}
