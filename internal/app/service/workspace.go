package service

import (
	"github.com/s-404/ladno/internal/app/entity"
)

// workspaceRepository is the persistence surface WorkspaceService needs.
type workspaceRepository interface {
	FindById(id string) *entity.Workspace
	FindAllLightweight() []entity.WorkspaceLightWeight
	FindAll() []*entity.Workspace
	Create(name string) (*entity.Workspace, error)
	Save(workspace *entity.Workspace) error
	Delete(id string) error
}

type WorkspaceService struct {
	workspaceRepository workspaceRepository
}

func newWorkspaceService(workspaceRepository workspaceRepository) *WorkspaceService {
	return &WorkspaceService{
		workspaceRepository: workspaceRepository,
	}
}

func (s *WorkspaceService) List(cb func([]entity.WorkspaceLightWeight, error)) {
	go func() {
		data := s.workspaceRepository.FindAllLightweight()
		cb(data, nil)
	}()
}

func (s *WorkspaceService) FindAll(cb func([]*entity.Workspace, error)) {
	go func() {
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
