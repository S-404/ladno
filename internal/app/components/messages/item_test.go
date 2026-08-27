package messages

import (
	"strings"
	"testing"
	"time"
)

func TestVisibleItemsLatestAndFilter(t *testing.T) {
	items := []Item{
		{Time: time.Date(2026, 1, 1, 15, 4, 5, 0, time.UTC), Dir: DirOut, Body: "hello"},
		{Time: time.Date(2026, 1, 1, 15, 4, 6, 0, time.UTC), Dir: DirIn, Body: `{"ok":true}`},
	}
	got := VisibleItems(items, "", false)
	if len(got) != 1 || got[0].Dir != DirIn {
		t.Fatalf("latest: %+v", got)
	}
	got = VisibleItems(items, "hello", true)
	if len(got) != 1 || got[0].Dir != DirOut {
		t.Fatalf("filter: %+v", got)
	}
}

func TestFormatItem(t *testing.T) {
	got := FormatItem(Item{
		Time: time.Date(2026, 1, 1, 15, 4, 5, 0, time.UTC),
		Dir:  DirIn,
		Body: "hi",
	})
	if !strings.Contains(got, "15:04:05.000") || !strings.Contains(got, "←") || !strings.Contains(got, "hi") {
		t.Fatalf("%q", got)
	}
}
