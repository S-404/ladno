package repository

type Repository struct {
	Workspace IWorkspaceRepository
}

func NewRepository() *Repository {
	return &Repository{
		Workspace: NewWorkspaceRepository(),
	}
}
