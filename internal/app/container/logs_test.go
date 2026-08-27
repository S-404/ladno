package container

import "testing"

func TestOffsetAtBottom(t *testing.T) {
	if !offsetAtBottom(0, 100, 100) {
		t.Fatal("content that fits should count as bottom")
	}
	if !offsetAtBottom(400, 500, 100) {
		t.Fatal("offset at max should stick")
	}
	if offsetAtBottom(10, 500, 100) {
		t.Fatal("scrolled away should not stick")
	}
	if !offsetAtBottom(399.5, 500, 100) {
		t.Fatal("within 1px of bottom should stick")
	}
}
