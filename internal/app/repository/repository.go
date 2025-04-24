package repository

type Repository struct {
	Foo IFooRepository
}

func NewRepository() *Repository {
	return &Repository{
		Foo: NewFooRepository(),
	}
}
