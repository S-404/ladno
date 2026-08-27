package store

import (
	"testing"

	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
)

func newTestSelectionStoreWithLimit(ws *entity.Workspace, limit int) *SelectionStore {
	ws.SetFolderNestingLimit(limit)
	return newTestSelectionStore(ws)
}

func httpWorkspace(items ...entity.CollectionItem) *entity.Workspace {
	return &entity.Workspace{
		Id:   "ws-1",
		Name: "Test",
		Collections: []entity.Collection{
			{
				Id:    "c1",
				Name:  "HTTP",
				Type:  constants.CollectionTypeREST,
				Items: items,
			},
		},
	}
}

func TestClampFolderNestingLimit(t *testing.T) {
	if got := entity.ClampFolderNestingLimit(-5); got != -1 {
		t.Fatalf("below -1: got %d", got)
	}
	if got := entity.ClampFolderNestingLimit(-1); got != -1 {
		t.Fatalf("unlimited: got %d", got)
	}
	if got := entity.ClampFolderNestingLimit(0); got != 0 {
		t.Fatalf("zero: got %d", got)
	}
	if got := entity.ClampFolderNestingLimit(5); got != 5 {
		t.Fatalf("default: got %d", got)
	}
}

func TestAddFolderRespectsNestingLimit(t *testing.T) {
	ws := httpWorkspace()
	sel := newTestSelectionStoreWithLimit(ws, 1)

	id, _, ok := sel.AddFolder("c1", "")
	if !ok || id == "" {
		t.Fatal("top-level folder should be allowed at limit 1")
	}
	if sel.CanAddFolder("c1", id) {
		t.Fatal("nested folder should be blocked at limit 1")
	}
	if _, _, ok := sel.AddFolder("c1", id); ok {
		t.Fatal("AddFolder nested should fail at limit 1")
	}
	if len(findItemByID(&ws.Collections[0].Items, id).Item) != 0 {
		t.Fatal("nested folder was still inserted")
	}
}

func TestAddFolderNestedWhenLimitAllows(t *testing.T) {
	ws := httpWorkspace()
	sel := newTestSelectionStoreWithLimit(ws, 2)

	parent, _, ok := sel.AddFolder("c1", "")
	if !ok {
		t.Fatal("root folder")
	}
	child, _, ok := sel.AddFolder("c1", parent)
	if !ok {
		t.Fatal("nested folder should be allowed at limit 2")
	}
	if sel.CanAddFolder("c1", child) {
		t.Fatal("third level should be blocked at limit 2")
	}
}

func TestAddFolderUnlimited(t *testing.T) {
	ws := httpWorkspace()
	sel := newTestSelectionStoreWithLimit(ws, -1)

	parent := ""
	for i := 0; i < 8; i++ {
		id, _, ok := sel.AddFolder("c1", parent)
		if !ok {
			t.Fatalf("unlimited add failed at depth %d", i+1)
		}
		parent = id
	}
}

func TestAddFolderLimitZero(t *testing.T) {
	ws := httpWorkspace()
	sel := newTestSelectionStoreWithLimit(ws, 0)
	if sel.CanAddFolder("c1", "") {
		t.Fatal("limit 0 should block folders")
	}
	if _, _, ok := sel.AddFolder("c1", ""); ok {
		t.Fatal("AddFolder should fail at limit 0")
	}
}

func TestRelocateFolderRespectsNestingLimit(t *testing.T) {
	ws := httpWorkspace(
		entity.CollectionItem{Id: "fa", Name: "A", Item: []entity.CollectionItem{}},
		entity.CollectionItem{Id: "fb", Name: "B", Item: []entity.CollectionItem{}},
	)
	sel := newTestSelectionStoreWithLimit(ws, 1)

	if sel.RelocateItem("c1", "fa", "c1", "fb", 0) {
		t.Fatal("moving a folder into another should fail at limit 1")
	}
	root := ws.Collections[0].Items
	if len(root) != 2 || root[0].Id != "fa" || root[1].Id != "fb" {
		t.Fatalf("tree changed: %+v", root)
	}

	if !sel.RelocateItem("c1", "fa", "c1", "", 1) {
		t.Fatal("same-parent reorder should still work")
	}
	root = ws.Collections[0].Items
	if len(root) != 2 || root[0].Id != "fb" || root[1].Id != "fa" {
		t.Fatalf("reorder failed: %+v", root)
	}
}

func TestRelocateFolderSubtreeExceedsLimit(t *testing.T) {
	ws := httpWorkspace(
		entity.CollectionItem{
			Id:   "f1",
			Name: "one",
			Item: []entity.CollectionItem{
				{Id: "f2", Name: "two", Item: []entity.CollectionItem{}},
			},
		},
		entity.CollectionItem{Id: "f3", Name: "three", Item: []entity.CollectionItem{}},
	)
	sel := newTestSelectionStoreWithLimit(ws, 2)

	if sel.RelocateItem("c1", "f1", "c1", "f3", 0) {
		t.Fatal("moving a 2-deep subtree under another folder should fail at limit 2")
	}
	if !sel.RelocateItem("c1", "f2", "c1", "f3", 0) {
		t.Fatal("moving a leaf folder under f3 should stay within limit 2")
	}
	f3 := findItemByID(&ws.Collections[0].Items, "f3")
	if f3 == nil || len(f3.Item) != 1 || f3.Item[0].Id != "f2" {
		t.Fatalf("f2 should be under f3, got %+v", f3)
	}
}

func TestDefaultFolderNestingLimitIsFive(t *testing.T) {
	ws := httpWorkspace()
	sel := newTestSelectionStore(ws)
	parent := ""
	for i := 0; i < entity.DefaultFolderNestingLimit; i++ {
		id, _, ok := sel.AddFolder("c1", parent)
		if !ok {
			t.Fatalf("default limit should allow depth %d", i+1)
		}
		parent = id
	}
	if sel.CanAddFolder("c1", parent) {
		t.Fatal("default limit should block depth 6")
	}
}
