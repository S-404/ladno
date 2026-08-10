package service

import (
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/repository"
)

type IWorkspaceService interface {
	List(cb func([]entity.WorkspaceLightWeight, error))
	FindAll(cb func([]*entity.Workspace, error))
	Find(id string, cb func(*entity.Workspace, error))
}

type WorkspaceService struct {
	workspaceRepository repository.IWorkspaceRepository
}

func newWorkspaceService(workspaceRepository repository.IWorkspaceRepository) *WorkspaceService {
	return &WorkspaceService{
		workspaceRepository: workspaceRepository,
	}
}

func (s *WorkspaceService) List(cb func([]entity.WorkspaceLightWeight, error)) {
	go func() {
		// Искусственная задержка
		//time.Sleep(1 * time.Second)
		data := s.workspaceRepository.FindAllLightweight()
		cb(data, nil)
	}()
}

func (s *WorkspaceService) FindAll(cb func([]*entity.Workspace, error)) {
	go func() {
		// Искусственная задержка
		//time.Sleep(1 * time.Second)
		data := s.workspaceRepository.FindAll()
		cb(data, nil)
	}()
}

func (s *WorkspaceService) Find(id string, cb func(*entity.Workspace, error)) {
	go func() {
		// Искусственная задержка
		//time.Sleep(1 * time.Second)
		data := s.workspaceRepository.FindById(id)
		cb(data, nil)
	}()
}
