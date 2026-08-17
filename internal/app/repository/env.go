package repository

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/repository/mock"
	"github.com/s-404/ladno/internal/app/storage"
)

const envsFileName = "envs.json"

type envsFile struct {
	Version int           `json:"version"`
	Envs    []*entity.Env `json:"envs"`
}

type EnvRepository struct {
	mu    sync.RWMutex
	envs  []*entity.Env
	store *storage.Store
}

func NewEnvRepository(store *storage.Store) *EnvRepository {
	r := &EnvRepository{
		store: store,
		envs:  []*entity.Env{},
	}
	if err := r.load(); err != nil {
		log.Printf("[storage] env load: %v", err)
	}
	return r
}

func (r *EnvRepository) load() error {
	if r.store == nil {
		r.envs = cloneEnvList(mock.EnvData())
		return nil
	}

	var file envsFile
	err := r.store.LoadJSON(envsFileName, &file)
	if errors.Is(err, storage.ErrNotExist) {
		r.envs = cloneEnvList(mock.EnvData())
		return r.persistLocked()
	}
	if err != nil {
		return err
	}

	r.envs = make([]*entity.Env, 0, len(file.Envs))
	for _, e := range file.Envs {
		if e == nil {
			continue
		}
		r.envs = append(r.envs, cloneEnv(e))
	}
	return nil
}

func (r *EnvRepository) persist() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.persistLocked()
}

func (r *EnvRepository) persistLocked() error {
	if r.store == nil {
		return nil
	}
	file := envsFile{
		Version: 1,
		Envs:    cloneEnvList(r.envs),
	}
	if err := r.store.SaveJSON(envsFileName, file); err != nil {
		return err
	}
	return nil
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
	if err := r.persistLocked(); err != nil {
		r.envs = r.envs[:len(r.envs)-1]
		return nil, err
	}
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
		if existing.Id != env.Id {
			continue
		}
		prev := r.envs[i]
		updated := &entity.Env{
			Id:        existing.Id,
			Name:      name,
			Variables: cloneVariables(env.Variables),
		}
		r.envs[i] = updated
		if err := r.persistLocked(); err != nil {
			r.envs[i] = prev
			return nil, err
		}
		return cloneEnv(updated), nil
	}
	return nil, fmt.Errorf("env %s not found", env.Id)
}

func (r *EnvRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, e := range r.envs {
		if e.Id != id {
			continue
		}
		prev := append([]*entity.Env(nil), r.envs...)
		r.envs = append(r.envs[:i], r.envs[i+1:]...)
		if err := r.persistLocked(); err != nil {
			r.envs = prev
			return err
		}
		return nil
	}
	return fmt.Errorf("env %s not found", id)
}

func (r *EnvRepository) Clone(id string) (*entity.Env, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, e := range r.envs {
		if e.Id != id {
			continue
		}
		cloned := &entity.Env{
			Id:        newEnvID(),
			Name:      e.Name + " (copy)",
			Variables: cloneVariables(e.Variables),
		}
		r.envs = append(r.envs, cloned)
		if err := r.persistLocked(); err != nil {
			r.envs = r.envs[:len(r.envs)-1]
			return nil, err
		}
		return cloneEnv(cloned), nil
	}
	return nil, fmt.Errorf("env %s not found", id)
}

// Move ставит env с id на позицию toIndex (индекс после удаления из старого места).
func (r *EnvRepository) Move(id string, toIndex int) error {
	if id == "" {
		return fmt.Errorf("env id is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	from := -1
	for i, e := range r.envs {
		if e != nil && e.Id == id {
			from = i
			break
		}
	}
	if from < 0 {
		return fmt.Errorf("env %s not found", id)
	}
	if toIndex < 0 {
		toIndex = 0
	}
	if toIndex >= len(r.envs) {
		toIndex = len(r.envs) - 1
	}
	if from == toIndex {
		return nil
	}

	prev := append([]*entity.Env(nil), r.envs...)
	r.envs = moveEnvSlice(r.envs, from, toIndex)
	if err := r.persistLocked(); err != nil {
		r.envs = prev
		return err
	}
	return nil
}

func moveEnvSlice(items []*entity.Env, from, to int) []*entity.Env {
	if from == to || from < 0 || to < 0 || from >= len(items) || to >= len(items) {
		return items
	}
	item := items[from]
	items = append(items[:from], items[from+1:]...)
	items = append(items[:to], append([]*entity.Env{item}, items[to:]...)...)
	return items
}

func newEnvID() string {
	return fmt.Sprintf("env-%d", time.Now().UnixNano())
}

func cloneEnvList(in []*entity.Env) []*entity.Env {
	out := make([]*entity.Env, 0, len(in))
	for _, e := range in {
		if e == nil {
			continue
		}
		out = append(out, cloneEnv(e))
	}
	return out
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
