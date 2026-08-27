package store

import (
	"testing"

	"fyne.io/fyne/v2/data/binding"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
)

type memWorkspaceStore struct {
	ws *entity.Workspace
}

func (m *memWorkspaceStore) GetSelectedWorkspace() *entity.Workspace { return m.ws }
func (m *memWorkspaceStore) PublishWorkspace(ws *entity.Workspace) {
	if ws != nil {
		m.ws = ws
	}
}

func newTestSelectionStore(ws *entity.Workspace) *SelectionStore {
	// Avoid NewSelectionStore: binding.Set requires a running Fyne app.
	return &SelectionStore{
		Selection: binding.NewUntyped(),
		workspace: &memWorkspaceStore{ws: ws},
	}
}

func TestRelocateItemIntoNestedFolderBeforeSibling(t *testing.T) {
	ws := &entity.Workspace{
		Id:   "ws-1",
		Name: "Test",
		Collections: []entity.Collection{
			{
				Id:   "c-rest",
				Name: "REST",
				Type: constants.CollectionTypeREST,
				Items: []entity.CollectionItem{
					{
						Id:   "f-1",
						Name: "level1",
						Item: []entity.CollectionItem{
							{Id: "r-1", Name: "req", Request: &entity.ItemRequest{Method: constants.GET}},
							{
								Id:   "f-2",
								Name: "level2",
								Item: []entity.CollectionItem{
									{Id: "r-2", Name: "other", Request: &entity.ItemRequest{Method: constants.GET}},
								},
							},
						},
					},
				},
			},
		},
	}
	sel := newTestSelectionStore(ws)

	if !sel.RelocateItem("c-rest", "r-1", "c-rest", "f-2", 0) {
		t.Fatal("relocate failed")
	}
	f1 := findItemByID(&ws.Collections[0].Items, "f-1")
	if f1 == nil {
		t.Fatal("f-1 missing")
	}
	for _, it := range f1.Item {
		if it.Id == "r-1" {
			t.Fatal("r-1 still direct child of f-1")
		}
	}
	f2 := findItemByID(&f1.Item, "f-2")
	if f2 == nil {
		t.Fatal("f-2 missing")
	}
	if len(f2.Item) < 1 || f2.Item[0].Id != "r-1" {
		t.Fatalf("r-1 should be first in f-2, got %+v", f2.Item)
	}
}

func TestRelocateItemIntoFolderAppend(t *testing.T) {
	ws := &entity.Workspace{
		Id:   "ws-1",
		Name: "Test",
		Collections: []entity.Collection{
			{
				Id:   "c-rest",
				Type: constants.CollectionTypeREST,
				Items: []entity.CollectionItem{
					{Id: "r-1", Name: "req", Request: &entity.ItemRequest{Method: constants.GET}},
					{Id: "f-1", Name: "folder", Item: []entity.CollectionItem{}},
				},
			},
		},
	}
	sel := newTestSelectionStore(ws)

	if !sel.RelocateItem("c-rest", "r-1", "c-rest", "f-1", 0) {
		t.Fatal("relocate failed")
	}
	if len(ws.Collections[0].Items) != 1 || ws.Collections[0].Items[0].Id != "f-1" {
		t.Fatalf("root items: %+v", ws.Collections[0].Items)
	}
	f1 := findItemByID(&ws.Collections[0].Items, "f-1")
	if f1 == nil || len(f1.Item) != 1 || f1.Item[0].Id != "r-1" {
		t.Fatalf("expected r-1 inside f-1, folder=%+v", f1)
	}
}

func TestRelocateAcrossCollectionTypes(t *testing.T) {
	ws := &entity.Workspace{
		Id: "ws-1",
		Collections: []entity.Collection{
			{
				Id:   "c-rest",
				Type: constants.CollectionTypeREST,
				Items: []entity.CollectionItem{
					{Id: "r-rest", Name: "req", Request: &entity.ItemRequest{Method: constants.GET}},
					{Id: "f-1", Name: "folder", Item: []entity.CollectionItem{}},
				},
			},
			{
				Id:    "c-rest-2",
				Type:  constants.CollectionTypeREST,
				Items: []entity.CollectionItem{},
			},
			{
				Id:    "c-grpc",
				Type:  constants.CollectionTypeGRPC,
				Items: []entity.CollectionItem{},
			},
			{
				Id:    "c-nats",
				Type:  constants.CollectionTypeNATS,
				Items: []entity.CollectionItem{},
			},
		},
	}
	sel := newTestSelectionStore(ws)

	if !sel.RelocateItem("c-rest", "r-rest", "c-rest-2", "", 0) {
		t.Fatal("rest item should move between rest collections")
	}
	if sel.RelocateItem("c-rest-2", "r-rest", "c-grpc", "", 0) {
		t.Fatal("must not move rest item into grpc collection")
	}
	if sel.RelocateItem("c-rest-2", "r-rest", "c-nats", "", 0) {
		t.Fatal("must not move rest item into NATS collection")
	}
	if sel.RelocateItem("c-rest", "f-1", "c-grpc", "", 0) {
		t.Fatal("must not move folder across collection types")
	}

	if findItemByID(&ws.Collections[1].Items, "r-rest") == nil {
		t.Fatalf("rest item missing in dest: %+v", ws.Collections[1].Items)
	}
}

func TestAddRequestKindsMatchCollection(t *testing.T) {
	ws := &entity.Workspace{
		Id: "ws-1",
		Collections: []entity.Collection{
			{Id: "c-rest", Type: constants.CollectionTypeREST},
			{Id: "c-grpc", Type: constants.CollectionTypeGRPC},
			{Id: "c-ws", Type: constants.CollectionTypeWS},
			{Id: "c-sio", Type: constants.CollectionTypeSocketIO},
		},
	}
	sel := newTestSelectionStore(ws)

	id, _, ok := sel.AddRequest("c-rest", "", constants.RequestKindGRPC)
	if !ok {
		t.Fatal("add rest")
	}
	item := findItemByID(&ws.Collections[0].Items, id)
	if item == nil || item.Request == nil || item.Request.Kind() != constants.RequestKindREST || item.Name != "New request" {
		t.Fatalf("rest collection must force rest: %+v", item)
	}

	id, _, ok = sel.AddRequest("c-grpc", "", constants.RequestKindREST)
	if !ok {
		t.Fatal("add grpc")
	}
	item = findItemByID(&ws.Collections[1].Items, id)
	if item == nil || item.Request == nil || item.Request.Grpc == nil || item.Name != "New method" {
		t.Fatalf("grpc collection must force grpc: %+v", item)
	}

	id, _, ok = sel.AddRequest("c-ws", "", constants.RequestKindREST)
	if !ok {
		t.Fatal("add ws")
	}
	item = findItemByID(&ws.Collections[2].Items, id)
	if item == nil || item.Request == nil || item.Request.Ws == nil || item.Name != "New connection" {
		t.Fatalf("ws collection must force ws: %+v", item)
	}

	id, _, ok = sel.AddRequest("c-sio", "", constants.RequestKindREST)
	if !ok {
		t.Fatal("add socket.io")
	}
	item = findItemByID(&ws.Collections[3].Items, id)
	if item == nil || item.Request == nil || item.Request.SocketIO == nil || item.Name != "New Socket.IO" {
		t.Fatalf("socket.io collection must force socketio: %+v", item)
	}
}
