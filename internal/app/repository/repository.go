package repository

type Repository struct {
	Env       IEnvRepository
	Workspace IWorkspaceRepository
}

func NewRepository() *Repository {
	return &Repository{
		Env:       NewEnvRepository(),
		Workspace: NewWorkspaceRepository(),
	}
}
