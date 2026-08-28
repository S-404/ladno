package buildinfo

import "testing"

func TestVersion(t *testing.T) {
	if Version() == "" {
		t.Fatal("empty version")
	}
}
