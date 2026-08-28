package service

import (
	"github.com/s-404/ladno/internal/app/entity"
)

// envRepository is the persistence surface EnvService needs.
type envRepository interface {
	FindAll(workspaceID string) []*entity.Env
	FindById(id string) *entity.Env
	Create(workspaceID string, env *entity.Env) (*entity.Env, error)
	Update(env *entity.Env) (*entity.Env, error)
	Delete(id string) error
	Clone(id string) (*entity.Env, error)
	Move(workspaceID, id string, toIndex int) error
}

type EnvService struct {
	repo envRepository
}

func NewEnvService(repo envRepository) *EnvService {
	return &EnvService{repo: repo}
}

func (s *EnvService) List(workspaceID string, cb func([]*entity.Env, error)) {
	go func() {
		cb(s.repo.FindAll(workspaceID), nil)
	}()
}

func (s *EnvService) Find(id string, cb func(*entity.Env, error)) {
	go func() {
		cb(s.repo.FindById(id), nil)
	}()
}

func (s *EnvService) Create(workspaceID string, env *entity.Env, cb func(*entity.Env, error)) {
	go func() {
		cb(s.repo.Create(workspaceID, env))
	}()
}

func (s *EnvService) Update(env *entity.Env, cb func(*entity.Env, error)) {
	go func() {
		cb(s.repo.Update(env))
	}()
}

func (s *EnvService) Delete(id string, cb func(error)) {
	go func() {
		cb(s.repo.Delete(id))
	}()
}

func (s *EnvService) Clone(id string, cb func(*entity.Env, error)) {
	go func() {
		cb(s.repo.Clone(id))
	}()
}

func (s *EnvService) Move(workspaceID, id string, toIndex int, cb func(error)) {
	go func() {
		cb(s.repo.Move(workspaceID, id, toIndex))
	}()
}
