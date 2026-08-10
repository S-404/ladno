package service

import "github.com/s-404/ladno/internal/app/repository"

type Service struct {
	Rest      IRestService
	Workspace IWorkspaceService
}

func NewService(repos *repository.Repository) *Service {
	return &Service{
		Rest:      NewRestService(),
		Workspace: newWorkspaceService(repos.Workspace),
	}
}
