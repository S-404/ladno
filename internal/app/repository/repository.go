package repository

type Repository struct {
	Foo       IFooRepository
	Workspace IWorkspaceRepository
}

func NewRepository() *Repository {
	return &Repository{
		Foo:       NewFooRepository(),
		Workspace: NewWorkspaceRepository(),
	}
}
