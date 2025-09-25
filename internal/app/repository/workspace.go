package repository

import (
	"fmt"
	"github.com/s-404/goose/internal/app/entity"
	"github.com/s-404/goose/internal/app/repository/mock"
	"strings"
	"sync"
)

type IWorkspaceRepository interface {
	FindById(id string) *entity.Workspace
	List(titleSearch string) []*entity.Workspace
	FindAllLightweight() []entity.WorkspaceListItem
	FindAll() []*entity.Workspace
	Save(workspace *entity.Workspace) error
	Delete(id string) error
}

type WorkspaceRepository struct {
	mu         sync.RWMutex
	workspaces []*entity.Workspace
}

func NewWorkspaceRepository() *WorkspaceRepository {
	return &WorkspaceRepository{
		workspaces: mock.MockWorkspaceData(),
	}
}

func (r *WorkspaceRepository) FindById(id string) *entity.Workspace {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, workspace := range r.workspaces {
		if workspace.Id == id {
			return workspace
		}
	}
	return nil
}

func (r *WorkspaceRepository) List(titleSearch string) []*entity.Workspace {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*entity.Workspace
	for _, workspace := range r.workspaces {
		if strings.Contains(strings.ToLower(workspace.Title), strings.ToLower(titleSearch)) {
			result = append(result, workspace)
		}
	}
	return result
}

func (r *WorkspaceRepository) FindAll() []*entity.Workspace {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Возвращаем копию чтобы избежать изменений извне
	result := make([]*entity.Workspace, len(r.workspaces))
	copy(result, r.workspaces)
	return result
}

func (r *WorkspaceRepository) FindAllLightweight() []entity.WorkspaceListItem {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]entity.WorkspaceListItem, len(r.workspaces))
	for i, workspace := range r.workspaces {
		items[i] = entity.WorkspaceListItem{
			Id:    workspace.Id,
			Title: workspace.Title,
		}
	}
	return items
}

func (r *WorkspaceRepository) Save(workspace *entity.Workspace) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Проверяем существует ли уже workspace с таким ID
	for i, existing := range r.workspaces {
		if existing.Id == workspace.Id {
			r.workspaces[i] = workspace
			return nil
		}
	}

	// Если не существует, добавляем новый
	r.workspaces = append(r.workspaces, workspace)
	return nil
}

func (r *WorkspaceRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, workspace := range r.workspaces {
		if workspace.Id == id {
			r.workspaces = append(r.workspaces[:i], r.workspaces[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("workspace with id %s not found", id)
}
