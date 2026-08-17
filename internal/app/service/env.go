package service

import (
	"github.com/s-404/ladno/internal/app/entity"
)

// envRepository is the persistence surface EnvService needs.
type envRepository interface {
	FindAll() []*entity.Env
	FindById(id string) *entity.Env
	Create(env *entity.Env) (*entity.Env, error)
	Update(env *entity.Env) (*entity.Env, error)
	Delete(id string) error
	Clone(id string) (*entity.Env, error)
	Move(id string, toIndex int) error
}

type EnvService struct {
	repo envRepository
}

func NewEnvService(repo envRepository) *EnvService {
	return &EnvService{repo: repo}
}

func (s *EnvService) List(cb func([]*entity.Env, error)) {
	go func() {
		cb(s.repo.FindAll(), nil)
	}()
}

func (s *EnvService) Find(id string, cb func(*entity.Env, error)) {
	go func() {
		cb(s.repo.FindById(id), nil)
	}()
}

func (s *EnvService) Create(env *entity.Env, cb func(*entity.Env, error)) {
	go func() {
		cb(s.repo.Create(env))
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

func (s *EnvService) Move(id string, toIndex int, cb func(error)) {
	go func() {
		cb(s.repo.Move(id, toIndex))
	}()
}
