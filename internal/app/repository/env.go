package repository

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/repository/mock"
)

type IEnvRepository interface {
	FindAll() []*entity.Env
	FindById(id string) *entity.Env
	Create(env *entity.Env) (*entity.Env, error)
	Update(env *entity.Env) (*entity.Env, error)
	Delete(id string) error
	Clone(id string) (*entity.Env, error)
}

type EnvRepository struct {
	mu   sync.RWMutex
	envs []*entity.Env
}

func NewEnvRepository() *EnvRepository {
	return &EnvRepository{
		envs: mock.EnvData(),
	}
}

func (r *EnvRepository) FindAll() []*entity.Env {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]*entity.Env, len(r.envs))
	for i, e := range r.envs {
		out[i] = cloneEnv(e)
	}
	return out
}

func (r *EnvRepository) FindById(id string) *entity.Env {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, e := range r.envs {
		if e.Id == id {
			return cloneEnv(e)
		}
	}
	return nil
}

func (r *EnvRepository) Create(env *entity.Env) (*entity.Env, error) {
	if env == nil {
		return nil, fmt.Errorf("env is nil")
	}
	name := strings.TrimSpace(env.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	created := &entity.Env{
		Id:        newEnvID(),
		Name:      name,
		Variables: cloneVariables(env.Variables),
	}
	r.envs = append(r.envs, created)
	return cloneEnv(created), nil
}

func (r *EnvRepository) Update(env *entity.Env) (*entity.Env, error) {
	if env == nil || env.Id == "" {
		return nil, fmt.Errorf("env id is required")
	}
	name := strings.TrimSpace(env.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for i, existing := range r.envs {
		if existing.Id == env.Id {
			updated := &entity.Env{
				Id:        existing.Id,
				Name:      name,
				Variables: cloneVariables(env.Variables),
			}
			r.envs[i] = updated
			return cloneEnv(updated), nil
		}
	}
	return nil, fmt.Errorf("env %s not found", env.Id)
}

func (r *EnvRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, e := range r.envs {
		if e.Id == id {
			r.envs = append(r.envs[:i], r.envs[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("env %s not found", id)
}

func (r *EnvRepository) Clone(id string) (*entity.Env, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, e := range r.envs {
		if e.Id == id {
			cloned := &entity.Env{
				Id:        newEnvID(),
				Name:      e.Name + " (copy)",
				Variables: cloneVariables(e.Variables),
			}
			r.envs = append(r.envs, cloned)
			return cloneEnv(cloned), nil
		}
	}
	return nil, fmt.Errorf("env %s not found", id)
}

func newEnvID() string {
	return fmt.Sprintf("env-%d", time.Now().UnixNano())
}

func cloneEnv(e *entity.Env) *entity.Env {
	if e == nil {
		return nil
	}
	return &entity.Env{
		Id:        e.Id,
		Name:      e.Name,
		Variables: cloneVariables(e.Variables),
	}
}

func cloneVariables(vars []entity.EnvVariable) []entity.EnvVariable {
	if vars == nil {
		return []entity.EnvVariable{}
	}
	out := make([]entity.EnvVariable, len(vars))
	copy(out, vars)
	return out
}
