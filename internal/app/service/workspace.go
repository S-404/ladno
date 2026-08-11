package service

import (
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/repository"
)

type IWorkspaceService interface {
	List(cb func([]entity.WorkspaceLightWeight, error))
	FindAll(cb func([]*entity.Workspace, error))
	Find(id string, cb func(*entity.Workspace, error))
	Create(name string, cb func(*entity.Workspace, error))
	Save(workspace *entity.Workspace, cb func(error))
	Delete(id string, cb func(error))
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
		data := s.workspaceRepository.FindById(id)
		cb(data, nil)
	}()
}

func (s *WorkspaceService) Create(name string, cb func(*entity.Workspace, error)) {
	go func() {
		ws, err := s.workspaceRepository.Create(name)
		if cb != nil {
			cb(ws, err)
		}
	}()
}

func (s *WorkspaceService) Save(workspace *entity.Workspace, cb func(error)) {
	go func() {
		err := s.workspaceRepository.Save(workspace)
		if cb != nil {
			cb(err)
		}
	}()
}

func (s *WorkspaceService) Delete(id string, cb func(error)) {
	go func() {
		err := s.workspaceRepository.Delete(id)
		if cb != nil {
			cb(err)
		}
	}()
}
