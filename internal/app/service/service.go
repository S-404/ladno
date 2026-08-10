package service

import "github.com/s-404/ladno/internal/app/repository"

type Service struct {
	Env       IEnvService
	Rest      IRestService
	Workspace IWorkspaceService
}

func NewService(repos *repository.Repository) *Service {
	return &Service{
		Env:       NewEnvService(repos.Env),
		Rest:      NewRestService(),
		Workspace: newWorkspaceService(repos.Workspace),
	}
}
