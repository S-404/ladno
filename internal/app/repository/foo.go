package repository

import "fmt"

type IFooRepository interface {
	DoSomething()
}

type FooRepository struct {
	name string
}

func NewFooRepository() *FooRepository {
	return &FooRepository{
		name: "foo repo",
	}
}

func (r *FooRepository) DoSomething() {
	fmt.Printf("'%s' doing smth", r.name)
}
