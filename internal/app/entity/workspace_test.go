package entity

import "testing"

func TestWorkspaceFolderNestingLimit(t *testing.T) {
	var ws *Workspace
	if got := ws.GetFolderNestingLimit(); got != DefaultFolderNestingLimit {
		t.Fatalf("nil workspace: got %d", got)
	}

	w := &Workspace{}
	if got := w.GetFolderNestingLimit(); got != DefaultFolderNestingLimit {
		t.Fatalf("unset: got %d", got)
	}

	if got := w.SetFolderNestingLimit(-3); got != -1 {
		t.Fatalf("clamp below -1: got %d", got)
	}
	if got := w.GetFolderNestingLimit(); got != -1 {
		t.Fatalf("stored unlimited: got %d", got)
	}

	zero := 0
	w.FolderNestingLimit = &zero
	if got := w.GetFolderNestingLimit(); got != 0 {
		t.Fatalf("explicit zero: got %d", got)
	}
}
