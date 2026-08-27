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
				Type: constants.CollectionTypeHTTP,
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
				Type: constants.CollectionTypeHTTP,
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

func TestRelocateHTTPMixedKindsBetweenCollections(t *testing.T) {
	ws := &entity.Workspace{
		Id: "ws-1",
		Collections: []entity.Collection{
			{
				Id:   "c-http-1",
				Type: constants.CollectionTypeHTTP,
				Items: []entity.CollectionItem{
					{Id: "r-rest", Name: "req", Request: &entity.ItemRequest{Method: constants.GET}},
					{Id: "r-grpc", Name: "GetUser", Request: &entity.ItemRequest{Grpc: &entity.GrpcRequest{Method: "svc/Get"}}},
					{Id: "r-ws", Name: "Echo", Request: &entity.ItemRequest{Ws: &entity.WsRequest{URL: "ws://localhost"}}},
					{Id: "f-1", Name: "folder", Item: []entity.CollectionItem{}},
				},
			},
			{
				Id:    "c-http-2",
				Type:  constants.CollectionTypeHTTP,
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

	if !sel.RelocateItem("c-http-1", "r-grpc", "c-http-2", "", 0) {
		t.Fatal("grpc should move between HTTP collections")
	}
	if !sel.RelocateItem("c-http-1", "r-ws", "c-http-1", "f-1", 0) {
		t.Fatal("ws should move into folder in same HTTP collection")
	}
	if sel.RelocateItem("c-http-1", "r-rest", "c-nats", "", 0) {
		t.Fatal("must not move HTTP item into NATS collection")
	}

	http2 := findItemByID(&ws.Collections[1].Items, "r-grpc")
	if http2 == nil || http2.Request == nil || http2.Request.Grpc == nil {
		t.Fatalf("grpc item missing in dest: %+v", ws.Collections[1].Items)
	}
	f1 := findItemByID(&ws.Collections[0].Items, "f-1")
	if f1 == nil || len(f1.Item) != 1 || f1.Item[0].Id != "r-ws" {
		t.Fatalf("ws should be in folder, got %+v", f1)
	}
}

func TestAddRequestKindsInHTTPCollection(t *testing.T) {
	ws := &entity.Workspace{
		Id: "ws-1",
		Collections: []entity.Collection{{
			Id:   "c-http",
			Type: constants.CollectionTypeHTTP,
		}},
	}
	sel := newTestSelectionStore(ws)

	id, _, ok := sel.AddRequest("c-http", "", constants.RequestKindREST)
	if !ok {
		t.Fatal("add rest")
	}
	item := findItemByID(&ws.Collections[0].Items, id)
	if item == nil || item.Request == nil || item.Request.Kind() != constants.RequestKindREST || item.Name != "New request" {
		t.Fatalf("rest item: %+v", item)
	}

	id, _, ok = sel.AddRequest("c-http", "", constants.RequestKindGRPC)
	if !ok {
		t.Fatal("add grpc")
	}
	item = findItemByID(&ws.Collections[0].Items, id)
	if item == nil || item.Request == nil || item.Request.Grpc == nil || item.Name != "New method" {
		t.Fatalf("grpc item: %+v", item)
	}

	id, _, ok = sel.AddRequest("c-http", "", constants.RequestKindWS)
	if !ok {
		t.Fatal("add ws")
	}
	item = findItemByID(&ws.Collections[0].Items, id)
	if item == nil || item.Request == nil || item.Request.Ws == nil || item.Name != "New connection" {
		t.Fatalf("ws item: %+v", item)
	}

	id, _, ok = sel.AddRequest("c-http", "", constants.RequestKindSocketIO)
	if !ok {
		t.Fatal("add socket.io")
	}
	item = findItemByID(&ws.Collections[0].Items, id)
	if item == nil || item.Request == nil || item.Request.SocketIO == nil || item.Name != "New Socket.IO" {
		t.Fatalf("socket.io item: %+v", item)
	}

	id, _, ok = sel.AddRequest("c-http", "", constants.RequestKindNATS)
	if !ok {
		t.Fatal("nats kind in HTTP should fall back to rest")
	}
	item = findItemByID(&ws.Collections[0].Items, id)
	if item == nil || item.Request.Kind() != constants.RequestKindREST {
		t.Fatalf("nats kind should not create nats item in HTTP col: %+v", item)
	}
}
