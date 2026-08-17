package store

import (
	"testing"

	"fyne.io/fyne/v2/data/binding"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
)

type memWS struct {
	ws        *entity.Workspace
	publishes int
}

func (m *memWS) FetchWorkspaceList() {}
func (m *memWS) GetItems() *binding.UntypedList {
	l := binding.NewUntypedList()
	return &l
}
func (m *memWS) GetWorkspaceListItemDataItem(binding.DataItem) *entity.WorkspaceLightWeight {
	return nil
}
func (m *memWS) GetWorkspaceListItemByIndex(int) *entity.WorkspaceLightWeight { return nil }
func (m *memWS) FetchWorkspace(string)                                        {}
func (m *memWS) GetItem() binding.Untyped                                     { return binding.NewUntyped() }
func (m *memWS) GetWorkspaceDataItem(binding.DataItem) *entity.Workspace      { return m.ws }
func (m *memWS) GetSelectedWorkspace() *entity.Workspace                      { return m.ws }
func (m *memWS) PublishWorkspace(ws *entity.Workspace) {
	m.publishes++
	if ws != nil {
		m.ws = ws
	}
}
func (m *memWS) UpdateSelectedWorkspace(string, string) bool   { return false }
func (m *memWS) Create(string, func(*entity.Workspace, error)) {}
func (m *memWS) Delete(string, func(error))                    {}
func (m *memWS) GetIsFetching() *binding.Bool {
	b := binding.NewBool()
	return &b
}

type memEnv struct {
	envs     map[string]*entity.Env
	persists int
}

func (m *memEnv) FetchList()                                {}
func (m *memEnv) GetItems() *binding.UntypedList            { l := binding.NewUntypedList(); return &l }
func (m *memEnv) GetSelected() binding.Untyped              { return binding.NewUntyped() }
func (m *memEnv) Select(string)                             {}
func (m *memEnv) GetActiveID() binding.String               { return binding.NewString() }
func (m *memEnv) SetActive(string)                          {}
func (m *memEnv) GetIsFetching() *binding.Bool              { b := binding.NewBool(); return &b }
func (m *memEnv) Create(string)                             {}
func (m *memEnv) SaveSelected(string, []entity.EnvVariable) {}
func (m *memEnv) PersistEnv(id, name string, vars []entity.EnvVariable) bool {
	m.persists++
	if m.envs == nil {
		m.envs = map[string]*entity.Env{}
	}
	cp := make([]entity.EnvVariable, len(vars))
	copy(cp, vars)
	m.envs[id] = &entity.Env{Id: id, Name: name, Variables: cp}
	return true
}
func (m *memEnv) DeleteSelected()                             {}
func (m *memEnv) CloneSelected()                              {}
func (m *memEnv) MoveEnv(string, int) bool                    { return false }
func (m *memEnv) ActiveVariables() map[string]string          { return nil }
func (m *memEnv) GetItemByIndex(int) *entity.Env              { return nil }
func (m *memEnv) GetEnvDataItem(binding.DataItem) *entity.Env { return nil }
func (m *memEnv) GetEnvByID(id string) *entity.Env {
	if m.envs == nil {
		return nil
	}
	return m.envs[id]
}

func newDraftTestSelection(ws IWorkspaceStore) *SelectionStore {
	return &SelectionStore{
		Selection: binding.NewUntyped(),
		workspace: ws,
	}
}

func TestDraftRequestSaveAndDirty(t *testing.T) {
	ws := &entity.Workspace{
		Id: "ws",
		Collections: []entity.Collection{{
			Id:   "c1",
			Type: constants.CollectionTypeREST,
			Name: entity.DefaultNewCollectionName,
			Items: []entity.CollectionItem{{
				Id:   "r1",
				Name: entity.DefaultNewRequestName,
				Request: &entity.ItemRequest{
					Method: constants.GET,
					Url:    entity.RequestUrl{Raw: ""},
				},
			}},
		}},
	}
	mem := &memWS{ws: ws}
	sel := newDraftTestSelection(mem)
	env := &memEnv{}
	d := NewDraftStore(mem, sel, env)

	draft := d.EnsureRequestDraft("c1", "r1", entity.DefaultNewRequestName, ws.Collections[0].Items[0].Request)
	if d.IsRequestDirty("r1") {
		t.Fatal("ensure should not dirty")
	}
	draft.Request.Url.Raw = "https://example.com"
	draft.Name = "Get example"
	d.PutRequestDraft("r1", draft, true)
	if !d.IsRequestDirty("r1") {
		t.Fatal("expected dirty")
	}
	if mem.publishes != 0 {
		t.Fatalf("edit should not publish, got %d", mem.publishes)
	}

	again := d.EnsureRequestDraft("c1", "r1", "ignored", nil)
	if again.Request.Url.Raw != "https://example.com" {
		t.Fatalf("draft lost: %q", again.Request.Url.Raw)
	}

	if !d.SaveRequest("c1", "r1") {
		t.Fatal("save failed")
	}
	if d.IsRequestDirty("r1") {
		t.Fatal("dirty after save")
	}
	if mem.publishes != 1 {
		t.Fatalf("want 1 publish, got %d", mem.publishes)
	}
	if mem.ws.Collections[0].Items[0].Request.Url.Raw != "https://example.com" {
		t.Fatal("url not persisted")
	}
	if mem.ws.Collections[0].Items[0].Name != "Get example" {
		t.Fatal("name not persisted")
	}
}

func TestDraftEnvSave(t *testing.T) {
	mem := &memWS{}
	sel := newDraftTestSelection(mem)
	env := &memEnv{envs: map[string]*entity.Env{
		"e1": {Id: "e1", Name: entity.DefaultNewEnvName},
	}}
	d := NewDraftStore(mem, sel, env)
	d.EnsureEnvDraft(env.envs["e1"])
	d.PutEnvDraft("e1", entity.EnvDraft{
		Name: "Local",
		Variables: []entity.EnvVariable{
			{Key: "host", Value: "localhost", Enabled: true},
		},
	}, true)
	if env.persists != 0 {
		t.Fatal("edit should not persist env")
	}
	if !d.SaveEnv("e1") {
		t.Fatal("save env failed")
	}
	if d.IsEnvDirty("e1") {
		t.Fatal("dirty after save")
	}
	if env.persists != 1 {
		t.Fatalf("persists=%d", env.persists)
	}
	got := env.envs["e1"]
	if got.Name != "Local" || len(got.Variables) != 1 || got.Variables[0].Key != "host" {
		t.Fatalf("bad env: %+v", got)
	}
}

func TestCreateDefaultNames(t *testing.T) {
	if newRequestItem(constants.CollectionTypeREST).Name != entity.DefaultNewRequestName {
		t.Fatal("request name")
	}
	if newFolderItem().Name != entity.DefaultNewFolderName {
		t.Fatal("folder name")
	}
	ws := &entity.Workspace{Id: "ws"}
	mem := &memWS{ws: ws}
	sel := newDraftTestSelection(mem)
	id, ok := sel.CreateCollection(constants.CollectionTypeREST)
	if !ok || id == "" {
		t.Fatal("create collection")
	}
	if ws.Collections[0].Name != entity.DefaultNewCollectionName {
		t.Fatalf("collection name %q", ws.Collections[0].Name)
	}
	if mem.publishes < 1 {
		t.Fatal("create should persist")
	}
}

func TestDirtyOnlyEditedIDs(t *testing.T) {
	ws := &entity.Workspace{
		Id: "ws",
		Collections: []entity.Collection{{
			Id:   "c1",
			Type: constants.CollectionTypeREST,
			Name: "Col",
			Items: []entity.CollectionItem{
				{Id: "r1", Name: "A", Request: &entity.ItemRequest{Method: constants.GET}},
				{Id: "r2", Name: "B", Request: &entity.ItemRequest{Method: constants.GET}},
				{Id: "f1", Name: "Folder", Auth: entity.Auth{Type: constants.AuthTypeInherited}},
			},
		}},
	}
	mem := &memWS{ws: ws}
	sel := newDraftTestSelection(mem)
	d := NewDraftStore(mem, sel, &memEnv{})

	d.EnsureRequestDraft("c1", "r1", "A", ws.Collections[0].Items[0].Request)
	d.EnsureRequestDraft("c1", "r2", "B", ws.Collections[0].Items[1].Request)
	d.EnsureFolderDraft("c1", "f1", "Folder", ws.Collections[0].Items[2].Auth)
	d.EnsureCollectionDraft(ws.Collections[0])

	d.PutRequestDraft("r1", entity.RequestDraft{
		CollectionID: "c1", Name: "A2",
		Request: entity.ItemRequest{Method: constants.GET, Url: entity.RequestUrl{Raw: "x"}},
	}, true)

	if !d.IsRequestDirty("r1") || d.IsRequestDirty("r2") || d.IsFolderDirty("f1") || d.IsCollectionDirty("c1") {
		t.Fatalf("dirty flags: r1=%v r2=%v f1=%v c1=%v",
			d.IsRequestDirty("r1"), d.IsRequestDirty("r2"), d.IsFolderDirty("f1"), d.IsCollectionDirty("c1"))
	}
	if d.RequestDisplayName("r1", "A") != "A2" {
		t.Fatal("display name")
	}
}
