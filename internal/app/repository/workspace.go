package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
	"github.com/s-404/ladno/internal/app/repository/mock"
	"github.com/s-404/ladno/internal/app/storage"
	"github.com/s-404/ladno/internal/app/utils"
)

const workspacesFileName = "workspaces.json"

type workspacesFile struct {
	Version    int                 `json:"version"`
	Workspaces []*entity.Workspace `json:"workspaces"`
}

type WorkspaceRepository struct {
	mu         sync.RWMutex
	workspaces []*entity.Workspace
	store      *storage.Store
	envRepo    *EnvRepository
}

func NewWorkspaceRepository(store *storage.Store) *WorkspaceRepository {
	r := &WorkspaceRepository{
		store:      store,
		workspaces: []*entity.Workspace{},
	}
	if err := r.load(); err != nil {
		log.Printf("[storage] workspace load: %v", err)
	}
	return r
}

func (r *WorkspaceRepository) load() error {
	if r.store == nil {
		r.workspaces = cloneWorkspaceList(mock.WorkspaceData())
		return nil
	}

	var file workspacesFile
	err := r.store.LoadJSON(workspacesFileName, &file)
	if errors.Is(err, storage.ErrNotExist) {
		r.workspaces = cloneWorkspaceList(mock.WorkspaceData())
		return r.persistLocked()
	}
	if err != nil {
		return err
	}

	r.workspaces = make([]*entity.Workspace, 0, len(file.Workspaces))
	for _, ws := range file.Workspaces {
		if ws == nil {
			continue
		}
		r.workspaces = append(r.workspaces, cloneWorkspace(ws))
	}
	return nil
}

func (r *WorkspaceRepository) persistLocked() error {
	if r.store == nil {
		return nil
	}
	file := workspacesFile{
		Version:    1,
		Workspaces: cloneWorkspaceList(r.workspaces),
	}
	return r.store.SaveJSON(workspacesFileName, file)
}

func (r *WorkspaceRepository) SetEnvRepository(env *EnvRepository) {
	r.envRepo = env
}

func (r *WorkspaceRepository) FindById(id string) *entity.Workspace {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, workspace := range r.workspaces {
		if workspace.Id == id {
			return cloneWorkspace(workspace)
		}
	}
	return nil
}

func (r *WorkspaceRepository) List(titleSearch string) []*entity.Workspace {
	r.mu.RLock()
	defer r.mu.RUnlock()

	q := strings.ToLower(titleSearch)
	var result []*entity.Workspace
	for _, workspace := range r.workspaces {
		if q == "" || strings.Contains(strings.ToLower(workspace.Name), q) {
			result = append(result, cloneWorkspace(workspace))
		}
	}
	return result
}

func (r *WorkspaceRepository) FindAll() []*entity.Workspace {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneWorkspaceList(r.workspaces)
}

func (r *WorkspaceRepository) FindAllLightweight() []entity.WorkspaceLightWeight {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]entity.WorkspaceLightWeight, len(r.workspaces))
	for i, workspace := range r.workspaces {
		items[i] = entity.WorkspaceLightWeight{
			Id:   workspace.Id,
			Name: workspace.Name,
		}
	}
	return items
}

func (r *WorkspaceRepository) Create(name string) (*entity.Workspace, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	created := &entity.Workspace{
		Id:          utils.NewID("ws"),
		Name:        name,
		Collections: []entity.Collection{},
	}
	r.workspaces = append(r.workspaces, created)
	if err := r.persistLocked(); err != nil {
		r.workspaces = r.workspaces[:len(r.workspaces)-1]
		return nil, err
	}
	return cloneWorkspace(created), nil
}

func (r *WorkspaceRepository) Save(workspace *entity.Workspace) error {
	if workspace == nil || workspace.Id == "" {
		return fmt.Errorf("workspace id is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	cp := cloneWorkspace(workspace)
	for i, existing := range r.workspaces {
		if existing.Id == cp.Id {
			prev := r.workspaces[i]
			r.workspaces[i] = cp
			if err := r.persistLocked(); err != nil {
				r.workspaces[i] = prev
				return err
			}
			return nil
		}
	}

	r.workspaces = append(r.workspaces, cp)
	if err := r.persistLocked(); err != nil {
		r.workspaces = r.workspaces[:len(r.workspaces)-1]
		return err
	}
	return nil
}

func (r *WorkspaceRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, workspace := range r.workspaces {
		if workspace.Id != id {
			continue
		}
		prev := append([]*entity.Workspace(nil), r.workspaces...)
		r.workspaces = append(r.workspaces[:i], r.workspaces[i+1:]...)
		if err := r.persistLocked(); err != nil {
			r.workspaces = prev
			return err
		}
		if r.envRepo != nil {
			if envErr := r.envRepo.DeleteByWorkspace(id); envErr != nil {
				log.Printf("[storage] delete workspace envs %s: %v", id, envErr)
			}
		}
		return nil
	}
	return fmt.Errorf("workspace with id %s not found", id)
}

func cloneWorkspaceList(in []*entity.Workspace) []*entity.Workspace {
	out := make([]*entity.Workspace, 0, len(in))
	for _, ws := range in {
		if ws == nil {
			continue
		}
		out = append(out, cloneWorkspace(ws))
	}
	return out
}

func cloneWorkspace(ws *entity.Workspace) *entity.Workspace {
	if ws == nil {
		return nil
	}
	data, err := json.Marshal(ws)
	if err != nil {
		cp := *ws
		normalizeWorkspaceTypes(&cp)
		return &cp
	}
	var out entity.Workspace
	if err := json.Unmarshal(data, &out); err != nil {
		cp := *ws
		normalizeWorkspaceTypes(&cp)
		return &cp
	}
	normalizeWorkspaceTypes(&out)
	return &out
}

func normalizeWorkspaceTypes(ws *entity.Workspace) {
	if ws == nil {
		return
	}
	for i := range ws.Collections {
		ws.Collections[i].Type = constants.NormalizeCollectionType(ws.Collections[i].Type)
	}
}
