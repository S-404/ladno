package store

import (
	"fmt"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/utils"
)

// envService is the async persistence surface EnvStore needs.
type envService interface {
	List(cb func([]*entity.Env, error))
	Create(env *entity.Env, cb func(*entity.Env, error))
	Update(env *entity.Env, cb func(*entity.Env, error))
	Delete(id string, cb func(error))
	Clone(id string, cb func(*entity.Env, error))
	Move(id string, toIndex int, cb func(error))
}

// envSettings is the preferences surface EnvStore needs.
type envSettings interface {
	GetActiveEnvID() string
	SetActiveEnvID(id string)
}

// envDraftSync keeps open env drafts in sync with script-driven env writes.
type envDraftSync interface {
	SyncEnvVar(envID, key, value string)
	RemoveEnvVar(envID, key string)
}

type EnvStore struct {
	Items      binding.UntypedList
	Selected   binding.Untyped
	ActiveID   binding.String
	IsFetching binding.Bool
	envService envService
	settings   envSettings
	draftSync  envDraftSync
}

func NewEnvStore(svc envService, settings envSettings) *EnvStore {
	s := &EnvStore{
		Items:      binding.NewUntypedList(),
		Selected:   binding.NewUntyped(),
		ActiveID:   binding.NewString(),
		IsFetching: binding.NewBool(),
		envService: svc,
		settings:   settings,
	}
	if settings != nil {
		if id := settings.GetActiveEnvID(); id != "" {
			_ = s.ActiveID.Set(id)
		}
	}
	return s
}

func (s *EnvStore) SetDraftSync(sync envDraftSync) {
	s.draftSync = sync
}

func (s *EnvStore) GetItems() *binding.UntypedList {
	return &s.Items
}

func (s *EnvStore) GetSelected() binding.Untyped {
	return s.Selected
}

func (s *EnvStore) GetActiveID() binding.String {
	return s.ActiveID
}

func (s *EnvStore) GetIsFetching() *binding.Bool {
	return &s.IsFetching
}

func (s *EnvStore) FetchList() {
	if fetching, _ := s.IsFetching.Get(); fetching {
		return
	}
	_ = s.IsFetching.Set(true)
	s.envService.List(func(data []*entity.Env, err error) {
		fyne.Do(func() {
			defer s.IsFetching.Set(false)
			if err != nil {
				fmt.Printf("env list error: %v\n", err)
				return
			}
			_ = s.Items.Set(utils.UnpackArray(data))

			activeID, _ := s.ActiveID.Get()
			if activeID != "" && !envListContains(data, activeID) {
				activeID = ""
				_ = s.ActiveID.Set("")
				s.persistActiveID("")
			}
			if activeID == "" && len(data) > 0 {
				_ = s.ActiveID.Set(data[0].Id)
				s.persistActiveID(data[0].Id)
				activeID = data[0].Id
			}
			if sel := s.selectedEnv(); sel == nil && len(data) > 0 {
				// Prefer active env as selected when opening list.
				for _, env := range data {
					if env != nil && env.Id == activeID {
						_ = s.Selected.Set(env)
						return
					}
				}
				_ = s.Selected.Set(data[0])
			}
		})
	})
}

func (s *EnvStore) Select(id string) {
	items, err := s.Items.Get()
	if err != nil {
		return
	}
	for _, item := range items {
		env, ok := item.(*entity.Env)
		if ok && env != nil && env.Id == id {
			_ = s.Selected.Set(env)
			return
		}
	}
}

func (s *EnvStore) SetActive(id string) {
	_ = s.ActiveID.Set(id)
	s.persistActiveID(id)
}

func (s *EnvStore) Create(name string) {
	s.envService.Create(&entity.Env{Name: name, Variables: []entity.EnvVariable{}}, func(created *entity.Env, err error) {
		fyne.Do(func() {
			if err != nil {
				fmt.Printf("env create error: %v\n", err)
				return
			}
			s.appendItem(created)
			_ = s.Selected.Set(created)
			if active, _ := s.ActiveID.Get(); active == "" {
				_ = s.ActiveID.Set(created.Id)
				s.persistActiveID(created.Id)
			}
		})
	})
}

func (s *EnvStore) SaveSelected(name string, vars []entity.EnvVariable) {
	sel := s.selectedEnv()
	if sel == nil {
		return
	}
	s.PersistEnv(sel.Id, name, vars)
}

// PersistEnv writes env to memory + disk (explicit Save).
func (s *EnvStore) PersistEnv(id, name string, vars []entity.EnvVariable) bool {
	if id == "" {
		return false
	}
	copied := make([]entity.EnvVariable, len(vars))
	copy(copied, vars)
	updated := &entity.Env{
		Id:        id,
		Name:      name,
		Variables: copied,
	}
	s.replaceItem(updated)
	if sel := s.selectedEnv(); sel != nil && sel.Id == id {
		_ = s.Selected.Set(updated)
	}
	s.envService.Update(updated, func(saved *entity.Env, err error) {
		fyne.Do(func() {
			if err != nil {
				fmt.Printf("env update error: %v\n", err)
				return
			}
			s.replaceItem(saved)
			if sel := s.selectedEnv(); sel != nil && sel.Id == id {
				_ = s.Selected.Set(saved)
			}
		})
	})
	return true
}

func (s *EnvStore) GetEnvByID(id string) *entity.Env {
	items, err := s.Items.Get()
	if err != nil {
		return nil
	}
	for _, item := range items {
		env, ok := item.(*entity.Env)
		if ok && env != nil && env.Id == id {
			return env
		}
	}
	return nil
}

func (s *EnvStore) DeleteSelected() {
	sel := s.selectedEnv()
	if sel == nil {
		return
	}
	id := sel.Id
	s.envService.Delete(id, func(err error) {
		fyne.Do(func() {
			if err != nil {
				fmt.Printf("env delete error: %v\n", err)
				return
			}
			s.removeItem(id)
			if active, _ := s.ActiveID.Get(); active == id {
				items, _ := s.Items.Get()
				if len(items) > 0 {
					if env, ok := items[0].(*entity.Env); ok && env != nil {
						_ = s.ActiveID.Set(env.Id)
						s.persistActiveID(env.Id)
						_ = s.Selected.Set(env)
						return
					}
				}
				_ = s.ActiveID.Set("")
				s.persistActiveID("")
				_ = s.Selected.Set(nil)
			} else {
				items, _ := s.Items.Get()
				if len(items) > 0 {
					if env, ok := items[0].(*entity.Env); ok {
						_ = s.Selected.Set(env)
						return
					}
				}
				_ = s.Selected.Set(nil)
			}
		})
	})
}

func (s *EnvStore) CloneSelected() {
	sel := s.selectedEnv()
	if sel == nil {
		return
	}
	s.envService.Clone(sel.Id, func(cloned *entity.Env, err error) {
		fyne.Do(func() {
			if err != nil {
				fmt.Printf("env clone error: %v\n", err)
				return
			}
			s.appendItem(cloned)
			_ = s.Selected.Set(cloned)
		})
	})
}

// MoveEnv перемещает env на toIndex (индекс после удаления из старого места) и сохраняет порядок.
func (s *EnvStore) MoveEnv(id string, toIndex int) bool {
	if id == "" {
		return false
	}
	items, err := s.Items.Get()
	if err != nil {
		return false
	}
	from := -1
	for i, item := range items {
		env, ok := item.(*entity.Env)
		if ok && env != nil && env.Id == id {
			from = i
			break
		}
	}
	if from < 0 {
		return false
	}
	if toIndex < 0 {
		toIndex = 0
	}
	if toIndex >= len(items) {
		toIndex = len(items) - 1
	}
	if from == toIndex {
		return false
	}

	item := items[from]
	items = append(items[:from], items[from+1:]...)
	items = append(items[:toIndex], append([]any{item}, items[toIndex:]...)...)
	// Note: fyne UntypedList does not fire list-level listeners on same-length reorder.
	// EnvList applies the order locally; we still update Items for Get()/persist path.
	_ = s.Items.Set(items)

	s.envService.Move(id, toIndex, func(err error) {
		if err != nil {
			fyne.Do(func() {
				fmt.Printf("env move error: %v\n", err)
				s.FetchList()
			})
		}
	})
	return true
}

func (s *EnvStore) ActiveVariables() map[string]string {
	activeID, _ := s.ActiveID.Get()
	if activeID == "" {
		return nil
	}
	items, err := s.Items.Get()
	if err != nil {
		return nil
	}
	for _, item := range items {
		env, ok := item.(*entity.Env)
		if !ok || env == nil || env.Id != activeID {
			continue
		}
		out := make(map[string]string)
		for _, v := range env.Variables {
			if v.Enabled && v.Key != "" {
				out[v.Key] = v.Value
			}
		}
		return out
	}
	return nil
}

// UpsertActiveVar sets key=value on the active environment (creates key if missing) and persists.
func (s *EnvStore) UpsertActiveVar(key, value string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	activeID, _ := s.ActiveID.Get()
	env := s.GetEnvByID(activeID)
	if env == nil {
		return false
	}
	vars := make([]entity.EnvVariable, len(env.Variables))
	copy(vars, env.Variables)
	found := false
	for i := range vars {
		if vars[i].Key == key {
			vars[i].Value = value
			vars[i].Enabled = true
			found = true
			break
		}
	}
	if !found {
		vars = append(vars, entity.EnvVariable{Key: key, Value: value, Enabled: true})
	}
	if s.draftSync != nil {
		s.draftSync.SyncEnvVar(activeID, key, value)
	}
	return s.PersistEnv(activeID, env.Name, vars)
}

// ClearActiveVar clears the value of key on the active environment (keeps the key) and persists.
func (s *EnvStore) ClearActiveVar(key string) bool {
	return s.UpsertActiveVar(key, "")
}

// ActiveEnvKeys returns enabled variable keys of the active environment.
func (s *EnvStore) ActiveEnvKeys() []string {
	vars := s.ActiveVariables()
	if len(vars) == 0 {
		return nil
	}
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (s *EnvStore) GetItemByIndex(index int) *entity.Env {
	item, err := s.Items.GetItem(index)
	if err != nil {
		return nil
	}
	return s.GetEnvDataItem(item)
}

func (s *EnvStore) GetEnvDataItem(item binding.DataItem) *entity.Env {
	val, err := item.(binding.Untyped).Get()
	if err != nil {
		return nil
	}
	env, ok := val.(*entity.Env)
	if !ok {
		return nil
	}
	return env
}

func (s *EnvStore) selectedEnv() *entity.Env {
	val, err := s.Selected.Get()
	if err != nil || val == nil {
		return nil
	}
	env, ok := val.(*entity.Env)
	if !ok {
		return nil
	}
	return env
}

func (s *EnvStore) appendItem(env *entity.Env) {
	items, _ := s.Items.Get()
	items = append(items, env)
	_ = s.Items.Set(items)
}

func (s *EnvStore) replaceItem(env *entity.Env) {
	items, _ := s.Items.Get()
	for i, item := range items {
		existing, ok := item.(*entity.Env)
		if ok && existing != nil && existing.Id == env.Id {
			items[i] = env
			_ = s.Items.Set(items)
			return
		}
	}
}

func (s *EnvStore) removeItem(id string) {
	items, _ := s.Items.Get()
	next := make([]any, 0, len(items))
	for _, item := range items {
		env, ok := item.(*entity.Env)
		if ok && env != nil && env.Id == id {
			continue
		}
		next = append(next, item)
	}
	_ = s.Items.Set(next)
}

func (s *EnvStore) persistActiveID(id string) {
	if s.settings != nil {
		s.settings.SetActiveEnvID(id)
	}
}

func envListContains(data []*entity.Env, id string) bool {
	for _, env := range data {
		if env != nil && env.Id == id {
			return true
		}
	}
	return false
}
